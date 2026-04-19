package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// SingboxConfig is a partial representation of sing-box config.json
// We only need to inspect the inbounds array.
type SingboxConfig struct {
	Inbounds []Inbound `json:"inbounds"`
}

type Inbound struct {
	Type        string `json:"type"`
	Tag         string `json:"tag"`
	Listen      string `json:"listen"`
	ListenPort  int    `json:"listen_port"`
}

// ProxyMode mirrors the transparent proxy implementation type.
type ProxyMode string

const (
	ModeTProxy      ProxyMode = "tproxy"
	ModeRedirect    ProxyMode = "redirect"
	ModeTun         ProxyMode = "tun"
	ModeSystemProxy ProxyMode = "system_proxy"
)

// ParseConfig reads and parses config.json from path.
func ParseConfig(path string) (*SingboxConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg SingboxConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// DetectPort returns the listen_port of the first inbound whose type matches
// the requested proxy mode (tproxy → "tproxy", redirect → "redirect",
// tun → "tun"). For system_proxy the port is not needed.
func DetectPort(cfg *SingboxConfig, mode ProxyMode) (int, error) {
	if mode == ModeSystemProxy || mode == ModeTun {
		// tun is managed entirely by sing-box; system_proxy uses fixed ports
		return 0, nil
	}
	wantType := string(mode) // "tproxy" or "redirect"
	for _, ib := range cfg.Inbounds {
		if ib.Type == wantType && ib.ListenPort > 0 {
			return ib.ListenPort, nil
		}
	}
	return 0, fmt.Errorf("no inbound of type %q with listen_port found in config.json", wantType)
}
