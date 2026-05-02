package core

// SingaSettings is the unified persistent settings structure that controls
// inbound ports/names, experimental (cache_file + clash_api), and log.
// Saved to data/singa_settings.json and applied to every config file before
// sing-box is started, regardless of config mode (node/subscription/upload).
type SingaSettings struct {
	Inbound      InboundSettings      `json:"inbound"`
	Experimental ExperimentalSettings `json:"experimental"`
	Log          LogSettings          `json:"log"`
}

// InboundSettings controls the ports and interface names used when singa
// injects inbound entries into the running config.
type InboundSettings struct {
	DNSPort       int      `json:"dnsPort"`       // default 5356
	RedirectPort  int      `json:"redirectPort"`  // default 7892
	TProxyPort    int      `json:"tproxyPort"`    // default 7893
	MixedPort     int      `json:"mixedPort"`     // default 2081
	TunInterface  string   `json:"tunInterface"`  // default "singa"
	TunAddress    []string `json:"tunAddress"`    // default ["172.31.0.1/30","fdfe:dcba:9876::1/126"]
}

// ExperimentalSettings mirrors sing-box's experimental block.
type ExperimentalSettings struct {
	CacheEnabled bool   `json:"cacheEnabled"` // default true
	CachePath    string `json:"cachePath"`    // default "cache.db"

	ClashAPIEnabled  bool   `json:"clashAPIEnabled"`  // default false
	ClashAPIListen   string `json:"clashAPIListen"`   // e.g. "0.0.0.0:9090"
	ClashAPIUI       string `json:"clashAPIUI"`       // e.g. "ui"
	ClashAPIUIURL    string `json:"clashAPIUIURL"`    // download URL
	ClashAPIDetour   string `json:"clashAPIDetour"`   // outbound tag for UI download
	ClashAPIMode     string `json:"clashAPIMode"`     // "rule"|"global"|"direct"
}

// LogSettings mirrors sing-box's log block.
type LogSettings struct {
	Disabled bool   `json:"disabled"` // default true
	Level    string `json:"level"`    // "trace"|"debug"|"info"|"warn"|"error"|"fatal"|"panic"
}

// DefaultSingaSettings returns the out-of-box defaults.
func DefaultSingaSettings() SingaSettings {
	return SingaSettings{
		Inbound: InboundSettings{
			DNSPort:      5356,
			RedirectPort: 7892,
			TProxyPort:   7893,
			MixedPort:    2081,
			TunInterface: "singa",
			TunAddress:   []string{"172.31.0.1/30", "fdfe:dcba:9876::1/126"},
		},
		Experimental: ExperimentalSettings{
			CacheEnabled:   true,
			CachePath:      "cache.db",
			ClashAPIEnabled: false,
		},
		Log: LogSettings{
			Disabled: true,
			Level:    "warn",
		},
	}
}

// Filled returns a copy of ss with any zero-value fields replaced by defaults.
func (ss SingaSettings) Filled() SingaSettings {
	d := DefaultSingaSettings()
	if ss.Inbound.DNSPort == 0 {
		ss.Inbound.DNSPort = d.Inbound.DNSPort
	}
	if ss.Inbound.RedirectPort == 0 {
		ss.Inbound.RedirectPort = d.Inbound.RedirectPort
	}
	if ss.Inbound.TProxyPort == 0 {
		ss.Inbound.TProxyPort = d.Inbound.TProxyPort
	}
	if ss.Inbound.MixedPort == 0 {
		ss.Inbound.MixedPort = d.Inbound.MixedPort
	}
	if ss.Inbound.TunInterface == "" {
		ss.Inbound.TunInterface = d.Inbound.TunInterface
	}
	if len(ss.Inbound.TunAddress) == 0 {
		ss.Inbound.TunAddress = d.Inbound.TunAddress
	}
	if ss.Experimental.CachePath == "" {
		ss.Experimental.CachePath = d.Experimental.CachePath
	}
	if ss.Log.Level == "" {
		ss.Log.Level = d.Log.Level
	}
	return ss
}
