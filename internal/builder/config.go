package builder

import (
	"encoding/json"
	"fmt"

	"github.com/singa/internal/config"
	"github.com/singa/internal/node"
)

type RouteMode string

const (
	RouteModeWhitelist RouteMode = "whitelist"
	RouteModeGFWList   RouteMode = "gfwlist"
	RouteModeGlobal    RouteMode = "global"
)

// BuildConfig generates a complete sing-box config for node mode.
func BuildConfig(
	proxyMode config.ProxyMode,
	routeMode RouteMode,
	n *node.Node,
	ports Ports,
	lanProxy bool,
	ipv6 bool,
	srsDir string,
) ([]byte, error) {
	proxyOB, err := NodeToOutbound(n, "proxy")
	if err != nil {
		return nil, fmt.Errorf("outbound: %w", err)
	}

	listenAddr := "127.0.0.1"
	if lanProxy {
		listenAddr = "::"
	}

	cfg := M{
		"log": M{
			"level":     "warning",
			"timestamp": true,
		},
		"dns":      buildDNS(routeMode, ipv6),
		"inbounds": buildInbounds(proxyMode, ports, listenAddr),
		"outbounds": []interface{}{
			proxyOB,
			M{"type": "direct", "tag": "direct"},
			M{"type": "block", "tag": "block"},
		},
		"route": buildRoute(routeMode, srsDir),
		"experimental": M{
			"cache_file": M{
				"enabled": true,
				"path":    "cache.db",
			},
			"clash_api": M{
				"external_controller": "0.0.0.0:9090",
				"external_ui":         "ui",
				"default_mode":        "rule",
			},
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}

// ── Inbounds ───────────────────────────────────────────────────────────────
// NOTE: sniff / sniff_override_destination are removed in sing-box 1.13.0.
// Sniffing is now handled entirely in route rules via {"action":"sniff"}.

func buildInbounds(mode config.ProxyMode, ports Ports, listen string) []interface{} {
	inbounds := []interface{}{
		// DNS inbound: always :: on port 1053, receives raw DNS queries for hijack-dns
		M{
			"tag":         "dns-in",
			"type":        "direct",
			"listen":      listen,
			"listen_port": ports.DNS,
		},
		// Mixed (SOCKS5+HTTP) inbound for local proxy usage
		M{
			"tag":         "mixed-in",
			"type":        "mixed",
			"listen":      listen,
			"listen_port": ports.Mixed,
		},
	}

	switch mode {
	case config.ModeTProxy:
		inbounds = append(inbounds, M{
			"tag":         "tproxy-in",
			"type":        "tproxy",
			"listen":      listen,
			"listen_port": ports.TProxy,
		})
	case config.ModeRedirect:
		inbounds = append(inbounds, M{
			"tag":         "redirect-in",
			"type":        "redirect",
			"listen":      listen,
			"listen_port": ports.Redirect,
		})
	case config.ModeTun:
		inbounds = append(inbounds, M{
			"tag":            "tun-in",
			"type":           "tun",
			"interface_name": "singa",
			"address":        []string{"172.31.0.1/30", "fdfe:dcba:9876::1/126"},
			"auto_route":     false,
			"auto_redirect":  false,
		})
		// system_proxy: mixed-in only, no transparent inbound needed
	}
	return inbounds
}

// ── DNS ────────────────────────────────────────────────────────────────────

func buildDNS(routeMode RouteMode, ipv6 bool) M {
	// sing-box 1.12+: new DNS server format requires explicit "type" field.
	// "strategy" moves to dns top-level or per-rule, not per-server.
	// "address_resolver" renamed to "domain_resolver".
	// "independent_cache" deprecated in 1.14, removed.
	strategy := "ipv4_only"
	if ipv6 {
		strategy = "prefer_ipv4"
	}

	servers := []interface{}{
		M{
			"type":            "tls",
			"tag":             "remote-dns",
			"server":          "1.1.1.1",
			"domain_resolver": "bootstrap-dns",
			"detour":          "proxy",
		},
		M{
			"type":            "https",
			"tag":             "direct-dns",
			"server":          "223.5.5.5",
			"domain_resolver": "bootstrap-dns",
		},
		M{
			"type":           "udp",
			"tag":            "bootstrap-dns",
			"server":         "223.5.5.5",
		},
	}

	var rules []interface{}
	var finalDNS string
	switch routeMode {
	case RouteModeWhitelist:
		// CN/private domains → direct-dns; everything else → remote-dns
		rules = append(rules, M{
			"rule_set": []string{"geosite-cn", "geosite-private"},
			"action":   "route",
			"server":   "direct-dns",
		})
		finalDNS = "remote-dns"

	case RouteModeGFWList:
		// GFW/non-CN domains → remote-dns; everything else → direct-dns
		rules = append(rules, M{
			"rule_set": []string{"geosite-gfw", "geosite-geolocation-!cn"},
			"action":   "route",
			"server":   "remote-dns",
		})
		finalDNS = "direct-dns"

	case RouteModeGlobal:
		finalDNS = "remote-dns"
	}

	return M{
		"servers":  servers,
		"rules":    rules,
		"final":    finalDNS,
		"strategy": strategy,
	}
}

// ── Route ──────────────────────────────────────────────────────────────────

func buildRoute(routeMode RouteMode, srsDir string) M {
	// default_domain_resolver is required since sing-box 1.12.0.
	// It tells the router which DNS server to use when resolving domains
	// in route rules. We pick the appropriate resolver based on route mode:
	// - whitelist/global: remote-dns (proxy-side resolver)
	// - gfwlist: direct-dns (direct-side resolver, since most traffic is direct)
	defaultResolver := "remote-dns"
	if routeMode == RouteModeGFWList {
		defaultResolver = "direct-dns"
	}

	return M{
		"rules":                   buildRouteRules(routeMode),
		"rule_set":                buildRuleSets(routeMode, srsDir),
		"final":                   routeFinal(routeMode),
		"auto_detect_interface":   true,
		"default_domain_resolver": defaultResolver,
	}
}

func routeFinal(mode RouteMode) string {
	if mode == RouteModeGFWList {
		return "direct"
	}
	return "proxy"
}

func buildRouteRules(routeMode RouteMode) []interface{} {
	rules := []interface{}{
		// Sniff protocol/domain on all connections (replaces deprecated inbound.sniff)
		M{"action": "sniff", "timeout": "500ms"},
		// Hijack DNS queries received on dns-in inbound
		M{"inbound": []string{"dns-in"}, "action": "hijack-dns"},
	}

	// Block QUIC (UDP 443) for whitelist and gfwlist modes to force TCP,
	// which is more reliably proxied. Not applied in global mode.
	if routeMode == RouteModeWhitelist || routeMode == RouteModeGFWList {
		rules = append(rules, M{
			"network": []string{"udp"},
			"port":    []int{443},
			"action":  "reject",
		})
	}

	// Private domains always go direct (ip_is_private removed intentionally)
	rules = append(rules,
		M{"rule_set": []string{"geosite-private", "geoip-private"}, "outbound": "direct"},
	)

	switch routeMode {
	case RouteModeWhitelist:
		rules = append(rules,
			// CN DNS server IPs → direct (prevent DNS pollution)
			M{"ip_cidr": cnDNS(), "outbound": "direct"},
			// CN domains → direct
			M{"rule_set": []string{"geosite-cn"}, "outbound": "direct"},
			// CN IPs → direct
			M{"rule_set": []string{"geoip-cn"}, "outbound": "direct"},
		)
		// final="proxy" routes everything else

	case RouteModeGFWList:
		rules = append(rules,
			// Foreign DNS IPs → proxy (prevent DNS leak)
			M{"ip_cidr": foreignDNS(), "outbound": "proxy"},
			// GFW/non-CN domains → proxy
			M{"rule_set": []string{"geosite-gfw", "geosite-geolocation-!cn"}, "outbound": "proxy"},
		)
		// final="direct" routes everything else

	case RouteModeGlobal:
		// final="proxy" routes everything
	}

	return rules
}

func buildRuleSets(routeMode RouteMode, srsDir string) []interface{} {
	tags := []string{"geosite-private", "geoip-private"}

	switch routeMode {
	case RouteModeWhitelist:
		tags = append(tags, "geosite-cn", "geoip-cn")
	case RouteModeGFWList:
		tags = append(tags, "geosite-gfw", "geosite-geolocation-!cn")
	}

	out := make([]interface{}, 0, len(tags))
	for _, tag := range tags {
		out = append(out, M{
			"type":   "local",
			"tag":    tag,
			"format": "binary",
			"path":   srsDir + "/" + tag + ".srs",
		})
	}
	return out
}

// ── DNS IP lists ───────────────────────────────────────────────────────────

func cnDNS() []string {
	return []string{
		"223.5.5.5", "223.6.6.6", "119.29.29.29",
		"1.12.12.12", "120.53.53.53", "180.76.76.76",
		"114.114.114.114", "114.114.115.115",
	}
}

func foreignDNS() []string {
	return []string{
		"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4",
		"9.9.9.9", "149.112.112.112",
		"94.140.14.14", "94.140.15.15",
	}
}
