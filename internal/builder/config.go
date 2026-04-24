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
		"route": buildRoute(routeMode, srsDir, isReF1nd, blockAds),
		"experimental": M{
			"cache_file": M{
				"enabled": true,
				"path":    "cache.db",
			},
			"clash_api": M{
				"external_controller": "0.0.0.0:9090",
				"external_ui": "ui",
				"external_ui_download_url": "https://ghfast.top/https://github.com/Zephyruso/zashboard/releases/download/v3.5.1/dist-misans-only.zip",
				"external_ui_download_detour": "direct",
				"default_mode": "rule",
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
		// CN domains → direct-dns; everything else → remote-dns
		rules = append(rules, M{
			"rule_set": []string{"geosite-cn"},
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
		// Sniff protocol/domain on all connections (replaces deprecated inbound.sniff)
		M{"action": "sniff", "timeout": "500ms"},
		// Hijack DNS queries received on dns-in inbound
		M{"inbound": []string{"dns-in"}, "action": "hijack-dns"},
	}

	// Ad blocking: inserted before per-mode rules so ads are rejected
	// regardless of route mode. Only added when blockAds is enabled.
	if blockAds {
		rules = append(rules, M{"action": "reject", "rule_set": []string{"ads"}})
	}

	switch routeMode {
	case RouteModeWhitelist:
		rules = append(rules,
			// CN domains → direct
			M{"rule_set": []string{"geosite-cn"}, "outbound": "direct"},
		)
		// reF1nd build: resolve CN domains before routing to get real IPs,
		// so subsequent geoip-cn rule can match them correctly.
		if isReF1nd {
			rules = append(rules, M{"action": "resolve", "match_only": true})
		}
		rules = append(rules,
			// CN IPs → direct
			M{"rule_set": []string{"geoip-cn"}, "outbound": "direct"},
		)
		// final="proxy" routes everything else

	case RouteModeGFWList:
		rules = append(rules,
			// GFW/non-CN domains → proxy
			M{"rule_set": []string{"geosite-gfw", "geosite-geolocation-!cn"}, "outbound": "proxy"},
			// Well-known foreign service IPs → proxy
			M{"rule_set": []string{"geoip-telegram"}, "outbound": "proxy"},
		)
		// final="direct" routes everything else

	case RouteModeGlobal:
		// final="proxy" routes everything
	}

	return rules
}

func buildRuleSets(routeMode RouteMode, srsDir string, blockAds bool) []interface{} {
	// geosite-private and geoip-private are no longer needed: private/reserved
	// IP bypass is handled entirely by the nft reserved_ip / reserved_ip6 sets.
	var tags []string

	switch routeMode {
	case RouteModeWhitelist:
		tags = append(tags, "geosite-cn", "geoip-cn")
	case RouteModeGFWList:
		tags = append(tags, "geosite-gfw", "geosite-geolocation-!cn",
			"geoip-telegram")
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
