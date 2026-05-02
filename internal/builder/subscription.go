package builder

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ── Wizard config Go structs (mirror frontend form) ──────────────────────

type WizardLog struct {
	Disabled  bool   `json:"disabled"`
	Level     string `json:"level"`
	Output    string `json:"output"`
	Timestamp bool   `json:"timestamp"`
}

type WizardClashAPI struct {
	ExternalController              string   `json:"external_controller"`
	ExternalUI                      string   `json:"external_ui"`
	ExternalUIDownloadURL           string   `json:"external_ui_download_url"`
	ExternalUIDownloadDetour        string   `json:"external_ui_download_detour"`
	Secret                          string   `json:"secret"`
	DefaultMode                     string   `json:"default_mode"`
	AccessControlAllowOrigin        []string `json:"access_control_allow_origin"`
	AccessControlAllowPrivateNetwork bool    `json:"access_control_allow_private_network"`
}

type WizardCacheFile struct {
	Enabled     bool   `json:"enabled"`
	Path        string `json:"path"`
	StoreFakeIP bool   `json:"store_fakeip"`
}

// WizardInbound covers mixed/http/socks/redirect/tproxy (all share listen fields)
// and tun (uses interface_name, addressText, etc.)
type WizardInbound struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Tag         string `json:"tag"`
	Enable      bool   `json:"enable"`

	// listen-based (mixed/http/socks/redirect/tproxy)
	Listen       string `json:"listen"`
	ListenPort   int    `json:"listen_port"`
	UsersText    string `json:"usersText"`
	TCPFastOpen  bool   `json:"tcp_fast_open"`
	TCPMultiPath bool   `json:"tcp_multi_path"`
	UDPFragment  bool   `json:"udp_fragment"`
	RouteAddress string `json:"route_address"` // redirect/tproxy

	// tun
	InterfaceName             string `json:"interface_name"`
	AddressText               string `json:"addressText"`
	MTU                       int    `json:"mtu"`
	AutoRoute                 bool   `json:"auto_route"`
	StrictRoute               bool   `json:"strict_route"`
	EndpointIndependentNAT    bool   `json:"endpoint_independent_nat"`
	Stack                     string `json:"stack"`
	RouteAddressText          string `json:"route_address_text"`
	RouteExcludeAddressText   string `json:"route_exclude_address_text"`
}

// WizardOutboundRef is a reference to another outbound inside a selector/urltest
type WizardOutboundRef struct {
	ID   string `json:"id"`
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

// WizardOutbound covers selector / urltest / direct / block
type WizardOutbound struct {
	ID                       string              `json:"id"`
	Tag                      string              `json:"tag"`
	Type                     string              `json:"type"` // selector|urltest|direct|block
	Outbounds                []WizardOutboundRef `json:"outbounds"`
	Hidden                   bool                `json:"hidden"`
	Include                  string              `json:"include"`
	Exclude                  string              `json:"exclude"`
	Icon                     string              `json:"icon"`
	URL                      string              `json:"url"`
	Interval                 string              `json:"interval"`
	Tolerance                int                 `json:"tolerance"`
	InterruptExistConnections bool               `json:"interrupt_exist_connections"`
}

type WizardRuleset struct {
	ID             string `json:"id"`
	Type           string `json:"type"`   // local|remote|inline
	Tag            string `json:"tag"`
	Format         string `json:"format"` // binary|source
	URL            string `json:"url"`
	DownloadDetour string `json:"download_detour"`
	UpdateInterval string `json:"update_interval"`
	Path           string `json:"path"`
	Rules          string `json:"rules"` // inline JSON
}

type WizardRouteRule struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Enable   bool     `json:"enable"`
	Payload  string   `json:"payload"`
	Action   string   `json:"action"`
	Outbound string   `json:"outbound"`
	Invert   bool     `json:"invert"`
	Sniffer  []string `json:"sniffer"`
	Strategy string   `json:"strategy"`
	Server   string   `json:"server"`
}

type WizardRoute struct {
	FindProcess           bool              `json:"find_process"`
	AutoDetectInterface   bool              `json:"auto_detect_interface"`
	DefaultInterface      string            `json:"default_interface"`
	Final                 string            `json:"final"`
	DefaultDomainResolver string            `json:"default_domain_resolver"`
	RuleSet               []WizardRuleset   `json:"rule_set"`
	Rules                 []WizardRouteRule `json:"rules"`
}

type WizardDNSServer struct {
	ID             string `json:"id"`
	Tag            string `json:"tag"`
	Type           string `json:"type"` // udp|tcp|tls|https|quic|h3|fakeip|local
	Server         string `json:"server"`
	ServerPort     string `json:"server_port"`
	Path           string `json:"path"`
	DomainResolver string `json:"domain_resolver"`
	Detour         string `json:"detour"`
	Inet4Range     string `json:"inet4_range"`
	Inet6Range     string `json:"inet6_range"`
}

type WizardDNSRule struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Enable       bool   `json:"enable"`
	Payload      string `json:"payload"`
	Action       string `json:"action"`
	Server       string `json:"server"`
	Invert       bool   `json:"invert"`
	Strategy     string `json:"strategy"`
	DisableCache bool   `json:"disable_cache"`
	ClientSubnet string `json:"client_subnet"`
}

type WizardDNS struct {
	DisableCache     bool              `json:"disable_cache"`
	DisableExpire    bool              `json:"disable_expire"`
	IndependentCache bool              `json:"independent_cache"`
	ClientSubnet     string            `json:"client_subnet"`
	Strategy         string            `json:"strategy"`
	Final            string            `json:"final"`
	Servers          []WizardDNSServer `json:"servers"`
	Rules            []WizardDNSRule   `json:"rules"`
}

type WizardConfig struct {
	Log       WizardLog       `json:"log"`
	ClashAPI  WizardClashAPI  `json:"clashAPI"`
	CacheFile WizardCacheFile `json:"cacheFile"`
	Inbounds  []WizardInbound  `json:"inbounds"`
	Outbounds []WizardOutbound `json:"outbounds"`
	Route     WizardRoute     `json:"route"`
	DNS       WizardDNS       `json:"dns"`
}

// ── ID→Tag index helpers ──────────────────────────────────────────────────

func buildRSIndex(wc *WizardConfig) map[string]string {
	m := map[string]string{}
	for _, rs := range wc.Route.RuleSet {
		m[rs.ID] = rs.Tag
	}
	return m
}

func buildObIndex(wc *WizardConfig) map[string]string {
	m := map[string]string{"direct": "direct", "block": "block"}
	for _, ob := range wc.Outbounds {
		m[ob.ID] = ob.Tag
	}
	return m
}

func buildDNSSrvIndex(wc *WizardConfig) map[string]string {
	m := map[string]string{}
	for _, s := range wc.DNS.Servers {
		m[s.ID] = s.Tag
	}
	return m
}

// ── BuildConfigFromWizard: main entry point ───────────────────────────────

func BuildConfigFromWizard(wizardRaw json.RawMessage, subProxies SubProxies) ([]byte, error) {
	// proxies may be empty if no subscription is referenced — that's valid
	// (e.g. a config that only uses direct/block outbounds)

	// Defaults
	var wc WizardConfig
	wc.Log.Level = "warn"
	wc.Log.Timestamp = true
	wc.ClashAPI.ExternalController = "127.0.0.1:9090"
	wc.ClashAPI.DefaultMode = "rule"
	wc.ClashAPI.AccessControlAllowOrigin = []string{"*"}
	wc.CacheFile.Enabled = true
	wc.CacheFile.Path = "cache.db"
	wc.CacheFile.StoreFakeIP = true
	wc.Route.AutoDetectInterface = true

	if wizardRaw != nil {
		if err := json.Unmarshal(wizardRaw, &wc); err != nil {
			return nil, fmt.Errorf("parse wizard config: %w", err)
		}
	}

	rsIdx := buildRSIndex(&wc)
	obIdx := buildObIndex(&wc)
	dnsSrvIdx := buildDNSSrvIndex(&wc)

	// ── Log ──────────────────────────────────────────────────────────────
	logConf := M{
		"disabled":  wc.Log.Disabled,
		"level":     wc.Log.Level,
		"timestamp": wc.Log.Timestamp,
	}
	if wc.Log.Output != "" {
		logConf["output"] = wc.Log.Output
	}

	// ── Experimental ────────────────────────────────────────────────────
	clashAPI := M{
		"external_controller": wc.ClashAPI.ExternalController,
		"default_mode":        wc.ClashAPI.DefaultMode,
	}
	if len(wc.ClashAPI.AccessControlAllowOrigin) > 0 {
		clashAPI["access_control_allow_origin"] = wc.ClashAPI.AccessControlAllowOrigin
	}
	if wc.ClashAPI.ExternalUI != "" {
		clashAPI["external_ui"] = wc.ClashAPI.ExternalUI
	}
	uiURL := wc.ClashAPI.ExternalUIDownloadURL
	if uiURL == "" {
		uiURL = "https://ghfast.top/https://github.com/Zephyruso/zashboard/releases/download/v3.5.1/dist-misans-only.zip"
	}
	clashAPI["external_ui_download_url"] = uiURL
	if wc.ClashAPI.ExternalUIDownloadDetour != "" {
		clashAPI["external_ui_download_detour"] = wc.ClashAPI.ExternalUIDownloadDetour
	}
	if wc.ClashAPI.Secret != "" {
		clashAPI["secret"] = wc.ClashAPI.Secret
	}
	if wc.ClashAPI.AccessControlAllowPrivateNetwork {
		clashAPI["access_control_allow_private_network"] = true
	}

	cacheFile := M{
		"enabled":      wc.CacheFile.Enabled,
		"path":         wc.CacheFile.Path,
		"store_fakeip": wc.CacheFile.StoreFakeIP,
	}

	experimental := M{
		"clash_api":  clashAPI,
		"cache_file": cacheFile,
	}

	// ── Inbounds ──────────────────────────────────────────────────────────
	inbounds := buildInboundsFromWizard(wc.Inbounds)

	// ── Outbounds ─────────────────────────────────────────────────────────
	outbounds, err := buildOutboundsFromWizard(wc.Outbounds, subProxies, obIdx)
	if err != nil {
		return nil, err
	}

	// ── Route ─────────────────────────────────────────────────────────────
	route := buildRouteFromWizard(&wc, obIdx, rsIdx, dnsSrvIdx)

	// ── DNS ───────────────────────────────────────────────────────────────
	dns := buildDNSFromWizard(&wc, obIdx, rsIdx, dnsSrvIdx)

	cfg := M{
		"log":          logConf,
		"experimental": experimental,
		"inbounds":     inbounds,
		"outbounds":    outbounds,
		"route":        route,
		"dns":          dns,
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// ── Inbounds ──────────────────────────────────────────────────────────────

func buildInboundsFromWizard(ibs []WizardInbound) []M {
	var out []M
	for _, ib := range ibs {
		if !ib.Enable {
			continue
		}
		switch ib.Type {
		case "tun":
			addrs := splitTrim(ib.AddressText, ",")
			if len(addrs) == 0 {
				addrs = []string{"172.18.0.1/30", "fdfe:dcba:9876::1/126"}
			}
			stack := ib.Stack
			if stack == "" {
				stack = "mixed"
			}
			m := M{
				"type":                    "tun",
				"tag":                     ib.Tag,
				"address":                 addrs,
				"auto_route":              ib.AutoRoute,
				"strict_route":            ib.StrictRoute,
				"stack":                   stack,
				"endpoint_independent_nat": ib.EndpointIndependentNAT,
			}
			if ib.InterfaceName != "" {
				m["interface_name"] = ib.InterfaceName
			}
			if ib.MTU > 0 {
				m["mtu"] = ib.MTU
			}
			if ra := splitTrim(ib.RouteAddressText, ","); len(ra) > 0 {
				m["route_address"] = ra
			}
			if rea := splitTrim(ib.RouteExcludeAddressText, ","); len(rea) > 0 {
				m["route_exclude_address"] = rea
			}
			out = append(out, m)

		default: // mixed/http/socks/redirect/tproxy
			listen := ib.Listen
			if listen == "" {
				listen = "127.0.0.1"
			}
			m := M{
				"type":        ib.Type,
				"tag":         ib.Tag,
				"listen":      listen,
				"listen_port": ib.ListenPort,
			}
			if ib.TCPFastOpen {
				m["tcp_fast_open"] = true
			}
			if ib.TCPMultiPath {
				m["tcp_multi_path"] = true
			}
			if ib.UDPFragment {
				m["udp_fragment"] = true
			}
			if users := parseUsers(ib.UsersText); len(users) > 0 {
				m["users"] = users
			}
			out = append(out, m)
		}
	}
	return out
}

// ── Outbounds ─────────────────────────────────────────────────────────────

// SubProxies maps subscription ID → ordered proxy node list for that subscription.
type SubProxies map[string][]map[string]any

func buildOutboundsFromWizard(obs []WizardOutbound, subProxies SubProxies, obIdx map[string]string) ([]interface{}, error) {
	// proxyTagsBySubID: subscription ID → filtered tag list (respecting include/exclude per outbound).
	// We also need a global deduplicated set of all proxy nodes to append at the end.
	globalTagSeen := map[string]bool{}
	var allProxyNodes []map[string]any
	for _, nodes := range subProxies {
		for _, n := range nodes {
			tag, _ := n["tag"].(string)
			if tag == "" {
				tag, _ = n["name"].(string)
			}
			if tag != "" && !globalTagSeen[tag] {
				globalTagSeen[tag] = true
				allProxyNodes = append(allProxyNodes, n)
			}
		}
	}

	// Helper: get proxy tags for a subscription, applying keyword filters.
	subTags := func(subID, include, exclude string) []string {
		nodes := subProxies[subID]
		tags := make([]string, 0, len(nodes))
		for _, n := range nodes {
			tag, _ := n["tag"].(string)
			if tag == "" {
				tag, _ = n["name"].(string)
			}
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		return filterByKeywords(tags, include, exclude)
	}

	var result []interface{}

	for _, ob := range obs {
		switch ob.Type {
		case "direct":
			result = append(result, M{"type": "direct", "tag": ob.Tag})
		case "block":
			result = append(result, M{"type": "block", "tag": ob.Tag})
		case "selector", "urltest":
			// Build outbounds list: expand each ref in order.
			// Subscription refs → inject that subscription's proxy tags (filtered).
			// Built-in / other outbound refs → resolve tag by ID.
			var refs []string
			refSeen := map[string]bool{} // dedupe within this outbound's refs
			for _, ref := range ob.Outbounds {
				if ref.Type == "Subscription" || ref.Type == "Subscribe" {
					for _, t := range subTags(ref.ID, ob.Include, ob.Exclude) {
						if !refSeen[t] {
							refSeen[t] = true
							refs = append(refs, t)
						}
					}
				} else {
					tag := obIdx[ref.ID]
					if tag == "" {
						tag = ref.Tag
					}
					if tag == "" {
						tag = ref.ID
					}
					if !refSeen[tag] {
						refSeen[tag] = true
						refs = append(refs, tag)
					}
				}
			}
			// For urltest with no explicit refs, fall back to all proxies (filtered)
			if ob.Type == "urltest" && len(ob.Outbounds) == 0 {
				var allTags []string
				for _, t := range allProxyNodes {
					tag, _ := t["tag"].(string)
					if tag == "" {
						tag, _ = t["name"].(string)
					}
					if tag != "" {
						allTags = append(allTags, tag)
					}
				}
				refs = filterByKeywords(allTags, ob.Include, ob.Exclude)
			}

			m := M{
				"type":      ob.Type,
				"tag":       ob.Tag,
				"outbounds": refs,
			}
			if ob.Hidden {
				m["hidden"] = true
			}
			if ob.Type == "urltest" {
				url := ob.URL
				if url == "" {
					url = "https://www.gstatic.com/generate_204"
				}
				interval := ob.Interval
				if interval == "" {
					interval = "3m"
				}
				tol := ob.Tolerance
				if tol == 0 {
					tol = 150
				}
				m["url"] = url
				m["interval"] = interval
				m["tolerance"] = tol
			}
			result = append(result, m)
		}
	}

	// Append all proxy nodes (globally deduped) after selector/urltest outbounds
	for _, p := range allProxyNodes {
		result = append(result, p)
	}

	// Always ensure direct/block builtins are present
	hasBuiltinDirect, hasBuiltinBlock := false, false
	for _, ob := range obs {
		if ob.Type == "direct" {
			hasBuiltinDirect = true
		}
		if ob.Type == "block" {
			hasBuiltinBlock = true
		}
	}
	if !hasBuiltinDirect {
		result = append(result, M{"type": "direct", "tag": "direct"})
	}
	if !hasBuiltinBlock {
		result = append(result, M{"type": "block", "tag": "block"})
	}

	return result, nil
}

// filterByKeywords returns proxy tags matching include/exclude keyword rules
func filterByKeywords(tags []string, include, exclude string) []string {
	var out []string
	inclParts := splitNonEmpty(include, "|")
	exclParts := splitNonEmpty(exclude, "|")
	for _, t := range tags {
		tl := strings.ToLower(t)
		if len(inclParts) > 0 {
			matched := false
			for _, kw := range inclParts {
				if strings.Contains(tl, strings.ToLower(kw)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		excluded := false
		for _, kw := range exclParts {
			if strings.Contains(tl, strings.ToLower(kw)) {
				excluded = true
				break
			}
		}
		if !excluded {
			out = append(out, t)
		}
	}
	return out
}

// ── Route ─────────────────────────────────────────────────────────────────

func buildRouteFromWizard(wc *WizardConfig, obIdx, rsIdx, dnsSrvIdx map[string]string) M {
	// rule_set
	ruleSets := make([]M, 0, len(wc.Route.RuleSet))
	for _, rs := range wc.Route.RuleSet {
		entry := M{"tag": rs.Tag, "type": rs.Type}
		switch rs.Type {
		case "remote":
			entry["format"] = rs.Format
			entry["url"] = rs.URL
			if detour := resolveOBTag(rs.DownloadDetour, obIdx); detour != "" {
				entry["download_detour"] = detour
			}
			if rs.UpdateInterval != "" {
				entry["update_interval"] = rs.UpdateInterval
			}
		case "local":
			entry["format"] = rs.Format
			entry["path"] = rs.Path
		case "inline":
			var rules interface{}
			if err := json.Unmarshal([]byte(rs.Rules), &rules); err == nil {
				entry["rules"] = rules
			}
		}
		ruleSets = append(ruleSets, entry)
	}

	// rules
	var rules []M
	for _, rule := range wc.Route.Rules {
		if rule.Type == "InsertionPoint" || !rule.Enable {
			continue
		}
		r := buildRouteRule(rule, obIdx, rsIdx)
		if r != nil {
			rules = append(rules, r)
		}
	}

	// final outbound
	finalTag := resolveOBTag(wc.Route.Final, obIdx)
	if finalTag == "" {
		finalTag = "漏网之鱼"
	}

	route := M{
		"rules":                rules,
		"rule_set":             ruleSets,
		"auto_detect_interface": wc.Route.AutoDetectInterface,
		"final":                finalTag,
	}
	if wc.Route.FindProcess {
		route["find_process"] = true
	}
	if !wc.Route.AutoDetectInterface && wc.Route.DefaultInterface != "" {
		route["default_interface"] = wc.Route.DefaultInterface
	}
	if wc.Route.DefaultDomainResolver != "" {
		if tag, ok := dnsSrvIdx[wc.Route.DefaultDomainResolver]; ok && tag != "" {
			route["default_domain_resolver"] = M{"server": tag}
		}
	}
	return route
}

func buildRouteRule(rule WizardRouteRule, obIdx, rsIdx map[string]string) M {
	r := M{"action": rule.Action}
	if rule.Invert {
		r["invert"] = true
	}
	switch rule.Type {
	case "action":
		// no extra match fields — pure action rule (e.g. sniff, hijack-dns)
	case "inbound":
		// inbound must be an array
		r["inbound"] = []string{rule.Payload}
	case "rule_set":
		tags := resolveRSTags(rule.Payload, rsIdx)
		if len(tags) == 0 {
			return nil
		}
		if len(tags) == 1 {
			r["rule_set"] = tags[0]
		} else {
			r["rule_set"] = tags
		}
	case "clash_mode":
		r["clash_mode"] = rule.Payload
	case "protocol":
		r["protocol"] = rule.Payload
	case "network":
		r["network"] = rule.Payload
	case "network+port":
		// payload format: "udp:443" → network + port
		parts := strings.SplitN(rule.Payload, ":", 2)
		if len(parts) == 2 {
			r["network"] = parts[0]
			if p, err := strconv.Atoi(parts[1]); err == nil {
				r["port"] = p
			} else {
				r["port"] = parts[1]
			}
		}
	case "ip_is_private":
		r["ip_is_private"] = rule.Payload == "true"
	default:
		r[rule.Type] = rule.Payload
	}
	switch rule.Action {
	case "route":
		tag := resolveOBTag(rule.Outbound, obIdx)
		if tag != "" {
			r["outbound"] = tag
		}
	case "sniff":
		if len(rule.Sniffer) > 0 {
			r["sniffer"] = rule.Sniffer
		}
	case "resolve":
		if rule.Strategy != "" && rule.Strategy != "default" {
			r["strategy"] = rule.Strategy
		}
		if rule.Server != "" {
			r["server"] = rule.Server
		}
	}
	return r
}

// ── DNS ───────────────────────────────────────────────────────────────────

func buildDNSFromWizard(wc *WizardConfig, obIdx, rsIdx, srvIdx map[string]string) M {
	// servers
	servers := make([]M, 0, len(wc.DNS.Servers))
	for _, srv := range wc.DNS.Servers {
		entry := M{"tag": srv.Tag, "type": srv.Type}
		switch srv.Type {
		case "fakeip":
			if srv.Inet4Range != "" {
				entry["inet4_range"] = srv.Inet4Range
			}
			if srv.Inet6Range != "" {
				entry["inet6_range"] = srv.Inet6Range
			}
		case "local":
			// no extra fields
		default: // tcp/udp/tls/https/quic/h3
			entry["server"] = srv.Server
			if srv.ServerPort != "" {
				if p, err := strconv.Atoi(srv.ServerPort); err == nil && p > 0 {
					entry["server_port"] = p
				}
			}
			if (srv.Type == "https" || srv.Type == "h3") && srv.Path != "" {
				entry["path"] = srv.Path
			}
			if srv.DomainResolver != "" {
				if tag, ok := srvIdx[srv.DomainResolver]; ok {
					entry["domain_resolver"] = tag
				}
			}
			if srv.Detour != "" {
				if tag := resolveOBTag(srv.Detour, obIdx); tag != "" {
					entry["detour"] = tag
				}
			}
		}
		servers = append(servers, entry)
	}

	// rules
	var rules []M
	for _, rule := range wc.DNS.Rules {
		if rule.Type == "InsertionPoint" || !rule.Enable {
			continue
		}
		r := buildDNSRule(rule, rsIdx, srvIdx)
		if r != nil {
			rules = append(rules, r)
		}
	}

	// final server tag
	finalTag := ""
	if id := wc.DNS.Final; id != "" {
		if tag, ok := srvIdx[id]; ok {
			finalTag = tag
		} else {
			finalTag = id
		}
	}
	if finalTag == "" {
		finalTag = "Remote-DNS"
	}

	dns := M{
		"servers":           servers,
		"rules":             rules,
		"disable_cache":     wc.DNS.DisableCache,
		"disable_expire":    wc.DNS.DisableExpire,
		"independent_cache": wc.DNS.IndependentCache,
		"final":             finalTag,
	}
	if wc.DNS.Strategy != "" && wc.DNS.Strategy != "default" {
		dns["strategy"] = wc.DNS.Strategy
	}
	if wc.DNS.ClientSubnet != "" {
		dns["client_subnet"] = wc.DNS.ClientSubnet
	}
	return dns
}

func buildDNSRule(rule WizardDNSRule, rsIdx, srvIdx map[string]string) M {
	// special: fakeip logical+and rule
	if rule.Payload == "__fakeip__" {
		fakeipTag := srvIdx[rule.Server]
		if fakeipTag == "" {
			fakeipTag = "Fake-IP"
		}
		return M{
			"type": "logical",
			"mode": "and",
			"rules": []M{
				{
					"domain_suffix": []string{
						".lan", ".localdomain", ".example", ".invalid",
						".localhost", ".test", ".local", ".home.arpa",
					},
					"invert": true,
				},
				{"query_type": []string{"A", "AAAA"}},
			},
			"action": "route",
			"server": fakeipTag,
		}
	}

	r := M{"action": rule.Action}
	if rule.Invert {
		r["invert"] = true
	}

	switch rule.Type {
	case "rule_set":
		tags := resolveRSTags(rule.Payload, rsIdx)
		if len(tags) == 0 {
			return nil
		}
		if len(tags) == 1 {
			r["rule_set"] = tags[0]
		} else {
			r["rule_set"] = tags
		}
	case "clash_mode":
		r["clash_mode"] = rule.Payload
	case "ip_is_private":
		r["ip_is_private"] = rule.Payload == "true"
	default:
		r[rule.Type] = rule.Payload
	}

	if rule.Action == "route" {
		serverTag := ""
		if id := rule.Server; id != "" {
			if tag, ok := srvIdx[id]; ok {
				serverTag = tag
			} else {
				serverTag = id
			}
		}
		if serverTag != "" {
			r["server"] = serverTag
		}
		if rule.Strategy != "" && rule.Strategy != "default" {
			r["strategy"] = rule.Strategy
		}
		if rule.DisableCache {
			r["disable_cache"] = true
		}
		if rule.ClientSubnet != "" {
			r["client_subnet"] = rule.ClientSubnet
		}
	}
	return r
}

// ── Tag resolution helpers ────────────────────────────────────────────────

func resolveOBTag(id string, obIdx map[string]string) string {
	if id == "" {
		return ""
	}
	if tag, ok := obIdx[id]; ok && tag != "" {
		return tag
	}
	// fallback: id itself (could be direct/block or raw tag)
	return id
}

func resolveRSTags(payload string, rsIdx map[string]string) []string {
	ids := splitNonEmpty(payload, ",")
	var tags []string
	for _, id := range ids {
		if tag, ok := rsIdx[id]; ok && tag != "" {
			tags = append(tags, tag)
		} else if id != "" {
			tags = append(tags, id)
		}
	}
	return tags
}

// ── String helpers ────────────────────────────────────────────────────────

func splitTrim(s, sep string) []string {
	var out []string
	for _, p := range strings.Split(s, sep) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitNonEmpty(s, sep string) []string {
	var out []string
	for _, p := range strings.Split(s, sep) {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseUsers(text string) []M {
	var users []M
	for _, line := range splitTrim(text, "\n") {
		if idx := strings.Index(line, ":"); idx > 0 {
			users = append(users, M{
				"username": line[:idx],
				"password": line[idx+1:],
			})
		}
	}
	return users
}

// ── ValidateWizardConfig: tag reference validation ─────────────────────────

// ValidationError describes a single reference error.
type ValidationError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
}

// ValidateWizardConfig checks that every tag/ID reference in the wizard config
// points to something that actually exists. Returns a list of errors (empty = OK).
func ValidateWizardConfig(wizardRaw json.RawMessage) []ValidationError {
	var wc WizardConfig
	if err := json.Unmarshal(wizardRaw, &wc); err != nil {
		return []ValidationError{{Location: "root", Message: "invalid JSON: " + err.Error()}}
	}

	obIdx := buildObIndex(&wc)    // id -> tag for outbounds
	rsIdx := buildRSIndex(&wc)    // id -> tag for rulesets
	dnsSrvIdx := buildDNSSrvIndex(&wc) // id -> tag for DNS servers
	// Also build tag sets for direct lookup
	obTags := map[string]bool{"direct": true, "block": true}
	for _, ob := range wc.Outbounds {
		obTags[ob.Tag] = true
	}
	rsTags := map[string]bool{}
	for _, rs := range wc.Route.RuleSet {
		rsTags[rs.Tag] = true
	}
	dnsSrvTags := map[string]bool{}
	for _, s := range wc.DNS.Servers {
		dnsSrvTags[s.Tag] = true
	}

	var errs []ValidationError

	resolveOb := func(ref WizardOutboundRef, loc string) {
		if ref.Type == "Subscription" || ref.Type == "Subscribe" {
			return // subscription refs are validated at runtime
		}
		// Check by ID first, then by tag
		if ref.ID != "" {
			if _, ok := obIdx[ref.ID]; !ok {
				errs = append(errs, ValidationError{
					Location: loc,
					Message:  "引用了不存在的出站 ID: " + ref.ID,
				})
			}
			return
		}
		if ref.Tag != "" {
			if !obTags[ref.Tag] {
				errs = append(errs, ValidationError{
					Location: loc,
					Message:  "引用了不存在的出站 tag: " + ref.Tag,
				})
			}
		}
	}

	// Validate outbound internal references
	for i, ob := range wc.Outbounds {
		for j, ref := range ob.Outbounds {
			resolveOb(ref, fmt.Sprintf("outbound[%d](%s).outbounds[%d]", i, ob.Tag, j))
		}
	}

	// Validate route.final
	if f := wc.Route.Final; f != "" {
		if !obTags[f] {
			// try resolve by checking obIdx values
			found := false
			for _, t := range obIdx {
				if t == f {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, ValidationError{
					Location: "route.final",
					Message:  "引用了不存在的出站 tag: " + f,
				})
			}
		}
	}

	// Validate route rules
	for i, rule := range wc.Route.Rules {
		if rule.Type == "InsertionPoint" {
			continue
		}
		if rule.Outbound != "" && !obTags[rule.Outbound] {
			// Try lookup by ID
			if _, ok := obIdx[rule.Outbound]; !ok {
				errs = append(errs, ValidationError{
					Location: fmt.Sprintf("route.rules[%d]", i),
					Message:  "outbound 引用了不存在的 tag/ID: " + rule.Outbound,
				})
			}
		}
		// route rules reference server (DNS) for DNS action rules
		if rule.Server != "" && !dnsSrvTags[rule.Server] {
			if _, ok := dnsSrvIdx[rule.Server]; !ok {
				errs = append(errs, ValidationError{
					Location: fmt.Sprintf("route.rules[%d]", i),
					Message:  "server 引用了不存在的 DNS 服务器 tag/ID: " + rule.Server,
				})
			}
		}
	}

	// Validate DNS rules
	for i, dr := range wc.DNS.Rules {
		if dr.Server != "" && !dnsSrvTags[dr.Server] {
			if _, ok := dnsSrvIdx[dr.Server]; !ok {
				errs = append(errs, ValidationError{
					Location: fmt.Sprintf("dns.rules[%d]", i),
					Message:  "server 引用了不存在的 DNS 服务器 tag/ID: " + dr.Server,
				})
			}
		}
		if dr.Outbound != "" && !obTags[dr.Outbound] {
			if _, ok := obIdx[dr.Outbound]; !ok {
				errs = append(errs, ValidationError{
					Location: fmt.Sprintf("dns.rules[%d]", i),
					Message:  "outbound 引用了不存在的 tag/ID: " + dr.Outbound,
				})
			}
		}
	}

	// Validate route.default_domain_resolver
	if r := wc.Route.DefaultDomainResolver; r != "" {
		if !dnsSrvTags[r] {
			if _, ok := dnsSrvIdx[r]; !ok {
				errs = append(errs, ValidationError{
					Location: "route.default_domain_resolver",
					Message:  "引用了不存在的 DNS 服务器 tag/ID: " + r,
				})
			}
		}
	}

	return errs
}
