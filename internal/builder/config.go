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
	modes config.ProxyModes,
	routeMode RouteMode,
	n *node.Node,
	ports Ports,
	lanProxy bool,
	ipv6 bool,
	srsDir string,
	isReF1nd bool,
	blockAds bool,
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
		"log":      M{"disabled": true},
		"dns":      buildDNS(routeMode, ipv6),
		"inbounds": buildInbounds(modes, ports, listenAddr),
		"outbounds": []interface{}{
			proxyOB,
			M{"type": "direct", "tag": "direct"},
			M{"type": "block", "tag": "block"},
		},
		"route":        buildRoute(routeMode, srsDir, isReF1nd, blockAds),
		"experimental": M{},
	}

	return json.MarshalIndent(cfg, "", "  ")
}

// ── Inbounds ───────────────────────────────────────────────────────────────
// NOTE: sniff / sniff_override_destination are removed in sing-box 1.13.0.
// Sniffing is now handled entirely in route rules via {"action":"sniff"}.

func buildInbounds(modes config.ProxyModes, ports Ports, listen string) []interface{} {
	inbounds := []interface{}{
		// DNS inbound: receives raw DNS queries for hijack-dns
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

	// tproxy inbound: needed when TCP=tproxy OR UDP=tproxy.
	// sing-box tproxy inbound handles both TCP and UDP; the nft rules will
	// selectively redirect only the protocol(s) the user configured.
	if modes.NeedsTProxyInbound() {
		inbounds = append(inbounds, M{
			"tag":         "tproxy-in",
			"type":        "tproxy",
			"listen":      listen,
			"listen_port": ports.TProxy,
		})
	}

	// redirect inbound: needed when TCP=redir.
	// iptables/nft REDIRECT is TCP-only by design; UDP is handled separately.
	if modes.NeedsRedirectInbound() {
		inbounds = append(inbounds, M{
			"tag":         "redirect-in",
			"type":        "redirect",
			"listen":      listen,
			"listen_port": ports.Redirect,
		})
	}

	// tun inbound: needed when TCP=tun OR UDP=tun.
	// A single TUN device handles both protocols; nft marks only the
	// protocol(s) the user configured for TUN.
	if modes.NeedsTunInbound() {
		inbounds = append(inbounds, M{
			"tag":            "tun-in",
			"type":           "tun",
			"interface_name": "singa",
			"address":        []string{"172.31.0.1/30", "fdfe:dcba:9876::1/126"},
			"auto_route":     false,
			"auto_redirect":  false,
		})
	}

	return inbounds
}

// ── DNS ────────────────────────────────────────────────────────────────────

func buildDNS(routeMode RouteMode, ipv6 bool) M {
	strategy := "ipv4_only"
	if ipv6 {
		strategy = "prefer_ipv4"
	}

	servers := []interface{}{
		M{
			"type":   "tls",
			"tag":    "remote-dns",
			"server": "1.1.1.1",
			"detour": "proxy",
		},
		M{
			"type":   "udp",
			"tag":    "direct-dns",
			"server": "223.5.5.5",
		},
	}

	var rules []interface{}
	var finalDNS string
	switch routeMode {
	case RouteModeWhitelist:
		rules = append(rules, M{
			"rule_set": []string{"geosite-cn"},
			"action":   "route",
			"server":   "direct-dns",
		})
		finalDNS = "remote-dns"

	case RouteModeGFWList:
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

func buildRoute(routeMode RouteMode, srsDir string, isReF1nd bool, blockAds bool) M {
	defaultResolver := "remote-dns"
	if routeMode == RouteModeGFWList {
		defaultResolver = "direct-dns"
	}

	return M{
		"rules":                   buildRouteRules(routeMode, isReF1nd, blockAds),
		"rule_set":                buildRuleSets(routeMode, srsDir, blockAds),
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

func buildRouteRules(routeMode RouteMode, isReF1nd bool, blockAds bool) []interface{} {
	rules := []interface{}{
		M{"action": "sniff", "timeout": "500ms"},
		M{"inbound": []string{"dns-in"}, "action": "hijack-dns"},
	}

	if blockAds {
		rules = append(rules, M{"action": "reject", "rule_set": []string{"ads"}})
	}

	switch routeMode {
	case RouteModeWhitelist:
		rules = append(rules,
			M{"rule_set": []string{"geosite-cn"}, "outbound": "direct"},
		)
		if isReF1nd {
			rules = append(rules, M{"action": "resolve", "match_only": true})
		}
		rules = append(rules,
			M{"rule_set": []string{"geoip-cn"}, "outbound": "direct"},
		)

	case RouteModeGFWList:
		rules = append(rules,
			M{"rule_set": []string{"geosite-gfw", "geosite-geolocation-!cn"}, "outbound": "proxy"},
			M{"rule_set": []string{"geoip-telegram"}, "outbound": "proxy"},
		)

	case RouteModeGlobal:
		// final="proxy" routes everything
	}

	return rules
}

func buildRuleSets(routeMode RouteMode, srsDir string, blockAds bool) []interface{} {
	var tags []string

	switch routeMode {
	case RouteModeWhitelist:
		tags = append(tags, "geosite-cn", "geoip-cn")
	case RouteModeGFWList:
		tags = append(tags, "geosite-gfw", "geosite-geolocation-!cn", "geoip-telegram")
	}

	out := make([]interface{}, 0, len(tags)+1)
	for _, tag := range tags {
		out = append(out, M{
			"type":   "local",
			"tag":    tag,
			"format": "binary",
			"path":   srsDir + "/" + tag + ".srs",
		})
	}

	if blockAds {
		out = append(out, M{
			"type":   "local",
			"tag":    "ads",
			"format": "binary",
			"path":   srsDir + "/ads.srs",
		})
	}

	return out
}
