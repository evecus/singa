package firewall

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var nftConfPath string

func SetNftConfPath(dir string) {
	nftConfPath = filepath.Join(dir, "singa-nft.conf")
}

// ── tproxy ─────────────────────────────────────────────────────────────────

func setupTproxy(port int, dnsPort int, lanProxy bool, ipv6 bool, gid uint32) error {
	conf := buildTproxyTable(port, dnsPort, ipv6, gid)
	if err := os.WriteFile(nftConfPath, []byte(conf), 0644); err != nil {
		return fmt.Errorf("write nft conf: %w", err)
	}

	// Policy routing: fwmark 0x40/0xc0 → table 100 (local route)
	routeCmds := []string{
		"ip rule add fwmark 0x40/0xc0 table 100",
		"ip route add local 0.0.0.0/0 dev lo table 100",
	}
	if ipv6 {
		routeCmds = append(routeCmds,
			"ip -6 rule add fwmark 0x40/0xc0 table 100",
			"ip -6 route add local ::/0 dev lo table 100",
		)
	}
	for _, c := range routeCmds {
		if err := runCmd(c); err != nil {
			log.Printf("firewall: route: %v", err)
		}
	}

	if lanProxy {
		if err := enableIPForward(ipv6); err != nil {
			log.Printf("firewall: ip_forward: %v", err)
		}
	}

	return runCmd("nft -f " + nftConfPath)
}

func buildTproxyTable(port int, dnsPort int, ipv6 bool, gid uint32) string {
	// tp_pre chain uses separate ipv4/ipv6 nfproto matchers so that
	// tproxy statements can specify the correct address family without conflict.
	nfprotoOut := "meta nfproto ipv4"
	if ipv6 {
		nfprotoOut = "meta nfproto { ipv4, ipv6 }"
	}

	// tproxy lines for tp_pre: must use per-family nfproto to avoid conflict
	tproxyLines := fmt.Sprintf(
		"        meta nfproto ipv4 meta l4proto { tcp, udp } mark & 0xc0 == 0x40 tproxy ip to 127.0.0.1:%d\n",
		port,
	)
	if ipv6 {
		tproxyLines += fmt.Sprintf(
			"        meta nfproto ipv6 meta l4proto { tcp, udp } mark & 0xc0 == 0x40 tproxy ip6 to [::1]:%d\n",
			port,
		)
	}

	// tp_pre forward match: for LAN traffic use per-family matchers
	tpPreFwdV4 := "meta nfproto ipv4 meta l4proto { tcp, udp } fib saddr type != local fib daddr type != local jump tp_rule"
	tpPreFwdV6 := ""
	if ipv6 {
		tpPreFwdV6 = "\n        meta nfproto ipv6 meta l4proto { tcp, udp } fib saddr type != local fib daddr type != local jump tp_rule"
	}

	// DNS redirect lines for the nat table.
	// Exempt 127.0.0.1 (loopback) to avoid redirecting sing-box's own DNS.
	dnsRedirectV4 := fmt.Sprintf(
		"        ip daddr != 127.0.0.1 meta l4proto { tcp, udp } th dport 53 redirect to :%d\n",
		dnsPort,
	)
	dnsRedirectV6 := ""
	if ipv6 {
		dnsRedirectV6 = fmt.Sprintf(
			"        ip6 daddr != ::1 meta l4proto { tcp, udp } th dport 53 redirect to :%d\n",
			dnsPort,
		)
	}

	return fmt.Sprintf(`table inet singa {
    set interface {
        type ipv4_addr
        flags interval
        auto-merge
    }
    set interface6 {
        type ipv6_addr
        flags interval
        auto-merge
    }

    # ── mangle: tproxy for normal traffic ──────────────────────────────────

    chain tp_mark {
        tcp flags & (fin | syn | rst | ack) == syn meta mark set mark | 0x40
        meta l4proto udp ct state new meta mark set mark | 0x40
        ct mark set mark
    }

    chain tp_rule {
        meta mark set ct mark
        meta mark & 0xc0 == 0x40 return
        # Hardcoded private-range bypass: prevents sing-box's own connections
        # to 127.x (dns-in, mixed-in, tproxy-in) from being re-captured after
        # conntrack entries expire (UDP entries expire in ~30s), which would
        # otherwise cause a traffic feedback loop with tens of thousands of
        # loopback connections and 100%% CPU usage.
        ip daddr { 127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 169.254.0.0/16, 100.64.0.0/10 } return
        ip6 daddr { ::1, fc00::/7, fe80::/10 } return
        ip daddr @interface return
        ip6 daddr @interface6 return
        # DNS (port 53) is handled by nat redirect below, skip here
        meta l4proto { tcp, udp } th dport 53 return
        jump tp_mark
    }

    chain tp_pre {
        iifname "lo" mark & 0xc0 != 0x40 return
        %s%s
%s    }

    chain tp_out {
        skgid %d return
        %s meta l4proto { tcp, udp } fib saddr type local fib daddr type != local jump tp_rule
    }

    chain prerouting_mangle {
        type filter hook prerouting priority mangle - 5; policy accept;
        jump tp_pre
    }

    chain output_mangle {
        type route hook output priority mangle - 5; policy accept;
        jump tp_out
    }

    # ── nat: redirect port 53 → sing-box dns-in ────────────────────────────

    chain dns_redirect {
        # skip sing-box own traffic
        skgid %d return
        # skip packets already going to our dns-in port (prevents redirect loop)
        meta l4proto { tcp, udp } th dport %d return
        # redirect DNS to dns-in
%s%s    }

    chain prerouting_nat {
        type nat hook prerouting priority dstnat - 5; policy accept;
        jump dns_redirect
    }

    chain output_nat {
        type nat hook output priority -105; policy accept;
        jump dns_redirect
    }
}
`, tpPreFwdV4, tpPreFwdV6, tproxyLines, gid, nfprotoOut, gid, dnsPort, dnsRedirectV4, dnsRedirectV6)
}

// ── redirect ───────────────────────────────────────────────────────────────

func setupRedirect(port int, dnsPort int, lanProxy bool, ipv6 bool, gid uint32) error {
	conf := buildRedirectTable(port, dnsPort, ipv6, gid)
	if err := os.WriteFile(nftConfPath, []byte(conf), 0644); err != nil {
		return fmt.Errorf("write nft conf: %w", err)
	}
	if lanProxy {
		if err := enableIPForward(ipv6); err != nil {
			log.Printf("firewall: ip_forward: %v", err)
		}
	}
	return runCmd("nft -f " + nftConfPath)
}

func buildRedirectTable(port int, dnsPort int, ipv6 bool, gid uint32) string {
	nfproto := "meta nfproto { ipv4, ipv6 }"
	if !ipv6 {
		nfproto = "meta nfproto ipv4"
	}

	dnsRedirectV6 := ""
	if ipv6 {
		dnsRedirectV6 = fmt.Sprintf(
			"        ip6 daddr != ::1 meta l4proto { tcp, udp } th dport 53 redirect to :%d\n",
			dnsPort,
		)
	}

	return fmt.Sprintf(`table inet singa {
    set whitelist {
        type ipv4_addr
        flags interval
        auto-merge
        elements = {
            0.0.0.0/32, 10.0.0.0/8, 100.64.0.0/10, 127.0.0.0/8,
            169.254.0.0/16, 172.16.0.0/12, 192.0.0.0/24, 192.0.2.0/24,
            192.88.99.0/24, 192.168.0.0/16, 198.51.100.0/24,
            203.0.113.0/24, 224.0.0.0/4, 240.0.0.0/4
        }
    }
    set whitelist6 {
        type ipv6_addr
        flags interval
        auto-merge
        elements = {
            ::/128, ::1/128, 64:ff9b::/96, 100::/64,
            2001::/32, 2001:20::/28, fe80::/10, ff00::/8
        }
    }
    set interface {
        type ipv4_addr
        flags interval
        auto-merge
    }
    set interface6 {
        type ipv6_addr
        flags interval
        auto-merge
    }

    chain tp_rule {
        ip daddr @whitelist return
        ip daddr @interface return
        ip6 daddr @whitelist6 return
        ip6 daddr @interface6 return
        # Explicit loopback guard: whitelist set covers 127.0.0.0/8 statically
        # but this makes the intent unambiguous and protects against any
        # future set changes breaking the loopback bypass.
        ip daddr { 127.0.0.0/8 } return
        ip6 daddr { ::1 } return
        skgid %d return
        # skip DNS, handled by dns_redirect chain
        meta l4proto { tcp, udp } th dport 53 return
        %s meta l4proto tcp redirect to :%d
    }

    chain dns_redirect {
        skgid %d return
        meta l4proto { tcp, udp } th dport %d return
        ip daddr != 127.0.0.1 meta l4proto { tcp, udp } th dport 53 redirect to :%d
%s    }

    chain tp_pre {
        type nat hook prerouting priority dstnat - 5; policy accept;
        jump dns_redirect
        jump tp_rule
    }

    chain tp_out {
        type nat hook output priority -105; policy accept;
        jump dns_redirect
        jump tp_rule
    }
}
`, gid, nfproto, port, gid, dnsPort, dnsPort, dnsRedirectV6)
}

// ── Cleanup ────────────────────────────────────────────────────────────────

func Cleanup() {
	_ = runCmd("nft delete table inet singa")
	cleanupTproxyRoutes()
	if nftConfPath != "" {
		_ = os.Remove(nftConfPath)
	}
}

func cleanupTproxyRoutes() {
	for _, c := range []string{
		"ip rule del fwmark 0x40/0xc0 table 100",
		"ip route del local 0.0.0.0/0 dev lo table 100",
		"ip -6 rule del fwmark 0x40/0xc0 table 100",
		"ip -6 route del local ::/0 dev lo table 100",
	} {
		_ = runCmd(c)
	}
}

// ── Interface IP management ────────────────────────────────────────────────

func AddInterfaceIP(cidr string) {
	set := "interface"
	if !strings.Contains(cidr, ".") {
		set = "interface6"
	}
	if err := runCmd(fmt.Sprintf("nft add element inet singa %s { %s }", set, cidr)); err != nil {
		log.Printf("firewall: add %s: %v", cidr, err)
	}
}

func RemoveInterfaceIP(cidr string) {
	set := "interface"
	if !strings.Contains(cidr, ".") {
		set = "interface6"
	}
	if err := runCmd(fmt.Sprintf("nft delete element inet singa %s { %s }", set, cidr)); err != nil {
		log.Printf("firewall: remove %s: %v", cidr, err)
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func enableIPForward(ipv6 bool) error {
	if err := runCmd("sysctl -w net.ipv4.ip_forward=1"); err != nil {
		return err
	}
	if ipv6 {
		return runCmd("sysctl -w net.ipv6.conf.all.forwarding=1")
	}
	return nil
}

func runCmd(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w (output: %s)", command, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func IsIPv6Supported() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.IsLoopback() && ipnet.IP.To4() == nil {
					return true
				}
			}
		}
	}
	return false
}
