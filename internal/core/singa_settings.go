package core

// SingaSettings is the unified persistent settings structure.
// Saved to data/singa_settings.json.
type SingaSettings struct {
	Inbound      InboundSettings      `json:"inbound"`
	Experimental ExperimentalSettings `json:"experimental"`
	Log          LogSettings          `json:"log"`
	// Auth settings
	Auth AuthSettings `json:"auth"`
	// Scheduled restart: cron expression (e.g. "15 3 * * *"), empty = disabled.
	// Only effective when core is running.
	ScheduledRestart ScheduledRestartSettings `json:"scheduledRestart"`
	// Custom sing-box working directory for "run -D <path>".
	// Empty = use default runDir.
	SingboxWorkDir string `json:"singboxWorkDir"`
}

// AuthSettings controls login authentication for the web UI.
type AuthSettings struct {
	Enabled      bool   `json:"enabled"`      // default true
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"` // bcrypt hash
}

// ScheduledRestartSettings controls periodic restart of the sing-box core.
type ScheduledRestartSettings struct {
	Enabled  bool   `json:"enabled"`
	Cron     string `json:"cron"` // standard 5-field cron, e.g. "15 3 * * *"
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
		Auth: AuthSettings{
			Enabled: true,
		},
		ScheduledRestart: ScheduledRestartSettings{
			Enabled: false,
			Cron:    "15 3 * * *",
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
	if ss.ScheduledRestart.Cron == "" {
		ss.ScheduledRestart.Cron = d.ScheduledRestart.Cron
	}
	return ss
}
