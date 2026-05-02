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

// ── Per-protocol transparent proxy modes ──────────────────────────────────

// TCPMode controls how TCP traffic is captured transparently.
type TCPMode string

const (
	TCPModeOff    TCPMode = "off"    // no transparent TCP proxy
	TCPModeRedir  TCPMode = "redir"  // nft REDIRECT (NAT, TCP-only)
	TCPModeTProxy TCPMode = "tproxy" // nft TPROXY
	TCPModeTun    TCPMode = "tun"    // TUN virtual NIC
)

// UDPMode controls how UDP traffic is captured transparently.
type UDPMode string

const (
	UDPModeOff    UDPMode = "off"    // no transparent UDP proxy
	UDPModeTProxy UDPMode = "tproxy" // nft TPROXY
	UDPModeTun    UDPMode = "tun"    // TUN virtual NIC
)

// ProxyModes carries the independent TCP and UDP transparent proxy choices.
// All downstream code (nft rule builder, sing-box inbound builder) uses this
// struct directly instead of collapsing it to a single ProxyMode value.
type ProxyModes struct {
	TCP TCPMode `json:"tcp"`
	UDP UDPMode `json:"udp"`
}

// NeedsTProxyInbound returns true when a tproxy inbound must exist in the
// sing-box config (TCP=tproxy OR UDP=tproxy).
func (pm ProxyModes) NeedsTProxyInbound() bool {
	return pm.TCP == TCPModeTProxy || pm.UDP == UDPModeTProxy
}

// NeedsRedirectInbound returns true when a redirect inbound must exist.
func (pm ProxyModes) NeedsRedirectInbound() bool {
	return pm.TCP == TCPModeRedir
}

// NeedsTunInbound returns true when a tun inbound must exist.
func (pm ProxyModes) NeedsTunInbound() bool {
	return pm.TCP == TCPModeTun || pm.UDP == UDPModeTun
}

// IsSystemProxyOnly returns true when no transparent proxy inbound is needed.
func (pm ProxyModes) IsSystemProxyOnly() bool {
	return !pm.NeedsTProxyInbound() && !pm.NeedsRedirectInbound() && !pm.NeedsTunInbound()
}

// ── Legacy ProxyMode kept for upload-config port detection only ───────────

type ProxyMode string

const (
	ModeTProxy      ProxyMode = "tproxy"
	ModeRedirect    ProxyMode = "redirect"
	ModeTun         ProxyMode = "tun"
	ModeSystemProxy ProxyMode = "system_proxy"
)

// ── Helpers ────────────────────────────────────────────────────────────────

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

func DetectDNSPort(cfg *SingboxConfig) int {
	for _, ib := range cfg.Inbounds {
		if ib.Type == "direct" && ib.Tag == "dns-in" && ib.ListenPort > 0 {
			return ib.ListenPort
		}
	}
	return 0
}

func DetectMixedPort(cfg *SingboxConfig) int {
	for _, ib := range cfg.Inbounds {
		if ib.Type == "mixed" && ib.ListenPort > 0 {
			return ib.ListenPort
		}
	}
	return 0
}

// DetectPort is used only for the upload-config path.
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
