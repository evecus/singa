package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type SingboxConfig struct {
	Inbounds []Inbound `json:"inbounds"`
}

type Inbound struct {
	Type       string `json:"type"`
	Tag        string `json:"tag"`
	Listen     string `json:"listen"`
	ListenPort int    `json:"listen_port"`
}

type ProxyMode string

const (
	ModeTProxy      ProxyMode = "tproxy"
	ModeRedirect    ProxyMode = "redirect"
	ModeTun         ProxyMode = "tun"
	ModeSystemProxy ProxyMode = "system_proxy"
)

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

// DetectPort returns the listen_port of the first inbound matching the proxy mode.
// For tun and system_proxy no port is needed, returns 0.
func DetectPort(cfg *SingboxConfig, mode ProxyMode) (int, error) {
	if mode == ModeSystemProxy || mode == ModeTun {
		return 0, nil
	}
	want := string(mode)
	for _, ib := range cfg.Inbounds {
		if ib.Type == want && ib.ListenPort > 0 {
			return ib.ListenPort, nil
		}
	}
	return 0, fmt.Errorf("no inbound of type %q with listen_port found in config.json", want)
}
