package core

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/singa/internal/config"
)

// patchConfig reads the run-time config.json, injects inbounds /
// experimental / log from SingaSettings, and writes it back.
// It is called for every config mode (node / subscription / upload)
// just before sing-box is started.
func patchConfig(path string, modes config.ProxyModes, ss SingaSettings, lanProxy bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config for patch: %w", err)
	}

	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	// ── inbounds ────────────────────────────────────────────────────────────
	// Strategy:
	//   1. Parse existing inbounds array.
	//   2. Remove any singa-managed inbound tags (dns-in, redirect-in,
	//      tproxy-in, tun-in) from whatever was there before.
	//   3. Re-insert managed inbounds according to current ProxyModes.
	// This means upload configs also get the correct inbounds injected.

	var inbounds []map[string]json.RawMessage
	if raw, ok := cfg["inbounds"]; ok {
		if err := json.Unmarshal(raw, &inbounds); err != nil {
			inbounds = nil
		}
	}
	inbounds = removeManagedInbounds(inbounds)
	inbounds = injectManagedInbounds(inbounds, modes, ss, lanProxy)

	inboundsRaw, err := json.Marshal(inbounds)
	if err != nil {
		return fmt.Errorf("marshal inbounds: %w", err)
	}
	cfg["inbounds"] = inboundsRaw

	// ── experimental ────────────────────────────────────────────────────────
	expRaw, err := json.Marshal(buildExperimental(ss.Experimental))
	if err != nil {
		return fmt.Errorf("marshal experimental: %w", err)
	}
	cfg["experimental"] = expRaw

	// ── log ─────────────────────────────────────────────────────────────────
	logRaw, err := json.Marshal(buildLog(ss.Log))
	if err != nil {
		return fmt.Errorf("marshal log: %w", err)
	}
	cfg["log"] = logRaw

	// ── write back ──────────────────────────────────────────────────────────
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal patched config: %w", err)
	}
	return os.WriteFile(path, out, 0644)
}

// managedTags is the set of inbound tags owned by singa.
// These are always removed and re-injected so settings changes take effect.
var managedTags = map[string]bool{
	"dns-in":      true,
	"redirect-in": true,
	"tproxy-in":   true,
	"tun-in":      true,
}

func removeManagedInbounds(ibs []map[string]json.RawMessage) []map[string]json.RawMessage {
	out := ibs[:0]
	for _, ib := range ibs {
		var tag string
		if raw, ok := ib["tag"]; ok {
			_ = json.Unmarshal(raw, &tag)
		}
		if !managedTags[tag] {
			out = append(out, ib)
		}
	}
	return out
}

func injectManagedInbounds(
	ibs []map[string]json.RawMessage,
	modes config.ProxyModes,
	ss SingaSettings,
	lanProxy bool,
) []map[string]json.RawMessage {
	listen := "127.0.0.1"
	if lanProxy {
		listen = "::"
	}

	in := ss.Inbound

	// dns-in: always present in every config
	ibs = prependInbound(ibs, jsonMap{
		"tag":         "dns-in",
		"type":        "direct",
		"listen":      listen,
		"listen_port": in.DNSPort,
	})

	// tproxy-in: when TCP=tproxy OR UDP=tproxy
	if modes.NeedsTProxyInbound() {
		ibs = append(ibs, toRawMap(jsonMap{
			"tag":         "tproxy-in",
			"type":        "tproxy",
			"listen":      listen,
			"listen_port": in.TProxyPort,
		}))
	}

	// redirect-in: when TCP=redir
	if modes.NeedsRedirectInbound() {
		ibs = append(ibs, toRawMap(jsonMap{
			"tag":         "redirect-in",
			"type":        "redirect",
			"listen":      listen,
			"listen_port": in.RedirectPort,
		}))
	}

	// tun-in: when TCP=tun OR UDP=tun
	if modes.NeedsTunInbound() {
		ibs = append(ibs, toRawMap(jsonMap{
			"tag":            "tun-in",
			"type":           "tun",
			"interface_name": in.TunInterface,
			"address":        in.TunAddress,
			"auto_route":     false,
			"auto_redirect":  false,
		}))
	}

	return ibs
}

// prependInbound puts the dns-in at the front of the slice.
func prependInbound(ibs []map[string]json.RawMessage, m jsonMap) []map[string]json.RawMessage {
	return append([]map[string]json.RawMessage{toRawMap(m)}, ibs...)
}

// ── experimental builder ──────────────────────────────────────────────────

type jsonMap map[string]interface{}

func buildExperimental(e ExperimentalSettings) jsonMap {
	exp := jsonMap{}

	if e.CacheEnabled {
		exp["cache_file"] = jsonMap{
			"enabled": true,
			"path":    e.CachePath,
		}
	}

	if e.ClashAPIEnabled && e.ClashAPIListen != "" {
		api := jsonMap{
			"external_controller": e.ClashAPIListen,
			"default_mode":        orDefault(e.ClashAPIMode, "rule"),
		}
		if e.ClashAPIUI != "" {
			api["external_ui"] = e.ClashAPIUI
		}
		if e.ClashAPIUIURL != "" {
			api["external_ui_download_url"] = e.ClashAPIUIURL
		}
		if e.ClashAPIDetour != "" {
			api["external_ui_download_detour"] = e.ClashAPIDetour
		}
		exp["clash_api"] = api
	}

	return exp
}

// ── log builder ───────────────────────────────────────────────────────────

func buildLog(l LogSettings) jsonMap {
	if l.Disabled {
		return jsonMap{"disabled": true}
	}
	return jsonMap{
		"disabled":  false,
		"level":     orDefault(l.Level, "warn"),
		"timestamp": true,
	}
}

// ── helpers ───────────────────────────────────────────────────────────────

func toRawMap(m jsonMap) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		b, _ := json.Marshal(v)
		out[k] = b
	}
	return out
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
