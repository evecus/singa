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

func setupTproxy(port int, lanProxy bool, ipv6 bool) error {
	conf := buildTproxyTable(port, ipv6)
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

func buildTproxyTable(port int, ipv6 bool) string {
	nfproto := "meta nfproto { ipv4, ipv6 }"
	if !ipv6 {
		nfproto = "meta nfproto ipv4"
	}

	tproxyV6 := ""
	if ipv6 {
		tproxyV6 = fmt.Sprintf(
			"        %s meta l4proto { tcp, udp } mark & 0xc0 == 0x40 tproxy ip6 to [::1]:%d\n",
			nfproto, port,
		)
	}

	return fmt.Sprintf(`table inet v2raya {
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

    chain tp_mark {
        tcp flags & (fin | syn | rst | ack) == syn meta mark set mark | 0x40
        meta l4proto udp ct state new meta mark set mark | 0x40
        ct mark set mark
    }

    chain tp_rule {
        meta mark set ct mark
        meta mark & 0xc0 == 0x40 return
        ip daddr @interface return
        ip6 daddr @interface6 return
        meta l4proto { tcp, udp } th dport 53 jump tp_mark
        meta mark & 0xc0 == 0x40 return
        jump tp_mark
    }

    chain tp_pre {
        iifname "lo" mark & 0xc0 != 0x40 return
        %s meta l4proto { tcp, udp } fib saddr type != local fib daddr type != local jump tp_rule
        %s meta l4proto { tcp, udp } mark & 0xc0 == 0x40 tproxy ip to 127.0.0.1:%d
%s    }

    chain tp_out {
        meta mark & 0x80 == 0x80 return
        %s meta l4proto { tcp, udp } fib saddr type local fib daddr type != local jump tp_rule
    }

    chain prerouting {
        type filter hook prerouting priority mangle - 5; policy accept;
        jump tp_pre
    }

    chain output {
        type route hook output priority mangle - 5; policy accept;
        jump tp_out
    }
}
`, nfproto, nfproto, port, tproxyV6, nfproto)
}

// ── redirect ───────────────────────────────────────────────────────────────

func setupRedirect(port int, lanProxy bool, ipv6 bool) error {
	conf := buildRedirectTable(port, ipv6)
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

func buildRedirectTable(port int, ipv6 bool) string {
	nfproto := "meta nfproto { ipv4, ipv6 }"
	if !ipv6 {
		nfproto = "meta nfproto ipv4"
	}
	return fmt.Sprintf(`table inet v2raya {
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
        meta mark & 0x80 == 0x80 return
        %s meta l4proto tcp redirect to :%d
    }

    chain tp_pre {
        type nat hook prerouting priority dstnat - 5; policy accept;
        jump tp_rule
    }

    chain tp_out {
        type nat hook output priority -105; policy accept;
        jump tp_rule
    }
}
`, nfproto, port)
}

// ── Cleanup ────────────────────────────────────────────────────────────────

func Cleanup() {
	_ = runCmd("nft delete table inet v2raya")
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

// ── Interface IP management (called by watcher) ────────────────────────────

func AddInterfaceIP(cidr string) {
	set := "interface"
	if !strings.Contains(cidr, ".") {
		set = "interface6"
	}
	if err := runCmd(fmt.Sprintf("nft add element inet v2raya %s { %s }", set, cidr)); err != nil {
		log.Printf("firewall: add %s: %v", cidr, err)
	}
}

func RemoveInterfaceIP(cidr string) {
	set := "interface"
	if !strings.Contains(cidr, ".") {
		set = "interface6"
	}
	if err := runCmd(fmt.Sprintf("nft delete element inet v2raya %s { %s }", set, cidr)); err != nil {
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
