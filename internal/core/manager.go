package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/singa/internal/builder"
	"github.com/singa/internal/config"
	"github.com/singa/internal/firewall"
	"github.com/singa/internal/ipfilter"
	"github.com/singa/internal/node"
	"github.com/singa/internal/storage"
	"github.com/singa/internal/subscription"
	"github.com/singa/internal/profile"
	"github.com/singa/internal/sysproxy"
)

const singboxBin = "/usr/bin/sing-box"
const singaGroup = "singa"

func IsReF1ndBuild() bool {
	out, err := exec.Command(singboxBin, "version").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "reF1nd")
}

type State string

const (
	StateStopped State = "stopped"
	StateRunning State = "running"
	StateError   State = "error"
)

// StartParams bundles all start-time options.
// ProxyMode/LanProxy/IPv6 are intentionally removed: they are always loaded
// from the persistent ProxySettings store and must not be passed by callers.
type StartParams struct {
	BlockAds       bool              `json:"blockAds"`
	RouteMode      builder.RouteMode `json:"routeMode"`
	NodeID         string            `json:"nodeId"`
	ConfigMode     string            `json:"configMode"` // "upload" | "node" | "subnode" | "subscription" | "profile"
	SubscriptionID string            `json:"subscriptionId"`
	SubNodeIdx     int               `json:"subNodeIdx"` // used by "subnode" mode
	ProfileID      string            `json:"profileId"`
	ClashAPIPort   int               `json:"clashApiPort"`
	ClashAPISecret string            `json:"clashApiSecret"`
}

// savedState is persisted to data/state.json to survive restarts.
type savedState struct {
	Params  StartParams `json:"params"`
	Running bool        `json:"running"`
}

// ProxySettings holds the globally-persistent proxy mode configuration
// saved from the Settings page. Influences nft rules, routing, and inbounds
// for every config mode (node / subscription / upload).
type ProxySettings struct {
	TCPMode  config.TCPMode `json:"tcpMode"`
	UDPMode  config.UDPMode `json:"udpMode"`
	LanProxy bool           `json:"lanProxy"`
	IPv6     bool           `json:"ipv6"`
}

// toProxyModes converts ProxySettings to the config.ProxyModes struct used
// by the builder and firewall packages.
func (ps ProxySettings) toProxyModes() config.ProxyModes {
	tcp := ps.TCPMode
	if tcp == "" {
		tcp = config.TCPModeOff
	}
	udp := ps.UDPMode
	if udp == "" {
		udp = config.UDPModeOff
	}
	return config.ProxyModes{TCP: tcp, UDP: udp}
}

// isSystemProxyOnly returns true when no transparent proxy is configured.
func (ps ProxySettings) isSystemProxyOnly() bool {
	return ps.toProxyModes().IsSystemProxyOnly()
}

// Manager controls the sing-box subprocess and firewall rules.
type Manager struct {
	mu         sync.Mutex
	dataDir    string
	runDir     string
	srsDir     string
	configsDir string

	cmd    *exec.Cmd
	state  State
	errMsg string
	params StartParams
	ports  builder.Ports

	// resolved at Start time, kept for Stop / Status
	activeProxySettings ProxySettings

	nodeStore          *storage.Store
	stateStore         *storage.Store
	ipfilterStore      *storage.Store
	proxySettingsStore *storage.Store
	singaSettingsStore *storage.Store
	nodes              []*node.Node
	subManager         *subscription.Manager
	profileManager     *profile.Manager

	logMu   sync.RWMutex
	logBuf  []string
	logSubs []chan string
}

func NewManager(dataDir, runDir, srsDir string) *Manager {
	configsDir := filepath.Join(dataDir, "configs")
	nodesDir := filepath.Join(dataDir, "nodes")
	profilesDir := filepath.Join(dataDir, "profiles")
	for _, d := range []string{nodesDir, profilesDir} {
		_ = os.MkdirAll(d, 0755)
	}
	m := &Manager{
		dataDir:            dataDir,
		runDir:             runDir,
		srsDir:             srsDir,
		configsDir:         configsDir,
		state:              StateStopped,
		logBuf:             make([]string, 0, 500),
		nodeStore:          storage.New(nodesDir, "nodes.json"),
		stateStore:         storage.New(dataDir, "state.json"),
		ipfilterStore:      storage.New(dataDir, "ipfilter.json"),
		proxySettingsStore: storage.New(dataDir, "proxy_settings.json"),
		singaSettingsStore: storage.New(dataDir, "singa_settings.json"),
		subManager:         subscription.NewManager(nodesDir),
		profileManager:     profile.NewManager(profilesDir),
	}
	m.loadNodes()
	var ss savedState
	if err := m.stateStore.Load(&ss); err == nil {
		m.params = ss.Params
	}
	return m
}

func (m *Manager) ConfigPath() string    { return filepath.Join(m.configsDir, "config.json") }
func (m *Manager) RunConfigPath() string { return filepath.Join(m.runDir, "config.json") }

func (m *Manager) AutoStart() {
	var ss savedState
	if err := m.stateStore.Load(&ss); err != nil || !ss.Running {
		return
	}
	log.Printf("singa: last state was running, auto-starting sing-box")
	if err := m.Start(ss.Params); err != nil {
		log.Printf("singa: auto-start failed: %v", err)
	}
}

func (m *Manager) saveState(running bool) {
	ss := savedState{Params: m.params, Running: running}
	if err := m.stateStore.Save(&ss); err != nil {
		log.Printf("warn: save state: %v", err)
	}
}

// ── Node management ────────────────────────────────────────────────────────

func (m *Manager) loadNodes() {
	m.nodeStore.Load(&m.nodes)
	if m.nodes == nil {
		m.nodes = []*node.Node{}
	}
}

func (m *Manager) GetNodes() []*node.Node {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nodes
}

func (m *Manager) AddNodes(ns []*node.Node) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes = append(m.nodes, ns...)
	m.nodeStore.Save(m.nodes)
}

func (m *Manager) DeleteNode(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, n := range m.nodes {
		if n.ID == id {
			m.nodes = append(m.nodes[:i], m.nodes[i+1:]...)
			m.nodeStore.Save(m.nodes)
			return true
		}
	}
	return false
}

// ── Group helpers ──────────────────────────────────────────────────────────

func ensureSingaGroup() (uint32, error) {
	if g, err := user.LookupGroup(singaGroup); err == nil {
		gid, err := strconv.ParseUint(g.Gid, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse gid %q: %w", g.Gid, err)
		}
		return uint32(gid), nil
	}
	log.Printf("group %q not found, creating", singaGroup)
	if path, err := exec.LookPath("groupadd"); err == nil {
		out, err := exec.Command(path, "--system", singaGroup).CombinedOutput()
		if err != nil {
			return 0, fmt.Errorf("groupadd: %w (output: %s)", err, strings.TrimSpace(string(out)))
		}
		g, err := user.LookupGroup(singaGroup)
		if err != nil {
			return 0, fmt.Errorf("lookup group after create: %w", err)
		}
		gid, err := strconv.ParseUint(g.Gid, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse gid %q: %w", g.Gid, err)
		}
		return uint32(gid), nil
	}
	return writeGroupEntry(singaGroup)
}

func writeGroupEntry(name string) (uint32, error) {
	const groupFile = "/etc/group"
	data, err := os.ReadFile(groupFile)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", groupFile, err)
	}
	usedGIDs := make(map[uint32]bool)
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		if gid, err := strconv.ParseUint(parts[2], 10, 32); err == nil {
			usedGIDs[uint32(gid)] = true
		}
	}
	var chosen uint32
	for candidate := uint32(500); candidate < 65000; candidate++ {
		if !usedGIDs[candidate] {
			chosen = candidate
			break
		}
	}
	if chosen == 0 {
		return 0, fmt.Errorf("no free GID available")
	}
	entry := fmt.Sprintf("%s:x:%d:\n", name, chosen)
	f, err := os.OpenFile(groupFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", groupFile, err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		return 0, fmt.Errorf("write %s: %w", groupFile, err)
	}
	log.Printf("created group %q with GID %d in %s", name, chosen, groupFile)
	return chosen, nil
}

// ── Start / Stop ───────────────────────────────────────────────────────────

func (m *Manager) Start(p StartParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateRunning {
		return fmt.Errorf("already running")
	}

	ports := builder.DefaultPorts()
	m.ports = ports

	// Always load proxy settings from persistent store.
	// This guarantees that every config mode (node/subscription/upload)
	// uses the exact TCP/UDP modes the user configured in Settings.
	ps := m.loadProxySettings()
	modes := ps.toProxyModes()

	// Load singa settings (inbound ports, experimental, log).
	// These override DefaultPorts so the user-configured ports are used.
	ss := m.loadSingaSettings()
	ports.DNS = ss.Inbound.DNSPort
	ports.TProxy = ss.Inbound.TProxyPort
	ports.Redirect = ss.Inbound.RedirectPort
	ports.Mixed = ss.Inbound.MixedPort
	m.ports = ports

	switch p.ConfigMode {
	case "upload":
		if err := m.prepareUploadConfig(); err != nil {
			return err
		}
		cfg, err := config.ParseConfig(m.RunConfigPath())
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
		// For upload config, detect existing inbound ports from the JSON.
		// We scan for tproxy and redirect inbounds independently.
		if modes.NeedsTProxyInbound() {
			if port, err := config.DetectPort(cfg, config.ModeTProxy); err == nil && port > 0 {
				ports.TProxy = port
			}
		}
		if modes.NeedsRedirectInbound() {
			if port, err := config.DetectPort(cfg, config.ModeRedirect); err == nil && port > 0 {
				ports.Redirect = port
			}
		}
		if dnsPort := config.DetectDNSPort(cfg); dnsPort > 0 {
			ports.DNS = dnsPort
		}
		ports.Mixed = config.DetectMixedPort(cfg)
		m.ports = ports

	case "node":
		n := m.findNode(p.NodeID)
		if n == nil {
			return fmt.Errorf("node %q not found", p.NodeID)
		}
		if err := m.prepareNodeConfig(p, modes, ps.LanProxy, ps.IPv6, n, ports); err != nil {
			return err
		}

	case "subnode":
		if p.SubscriptionID == "" {
			return fmt.Errorf("subscriptionId is required")
		}
		proxies, err := m.subManager.GetProxies(p.SubscriptionID)
		if err != nil {
			return fmt.Errorf("subscription cache: %w", err)
		}
		if p.SubNodeIdx < 0 || p.SubNodeIdx >= len(proxies) {
			return fmt.Errorf("subNodeIdx %d out of range (subscription has %d nodes)", p.SubNodeIdx, len(proxies))
		}
		raw := proxies[p.SubNodeIdx]
		n, err := node.FromMap(raw)
		if err != nil {
			return fmt.Errorf("parse subscription node: %w", err)
		}
		if err := m.prepareNodeConfig(p, modes, ps.LanProxy, ps.IPv6, n, ports); err != nil {
			return err
		}

	case "subscription":
		if p.SubscriptionID == "" {
			return fmt.Errorf("subscriptionId is required")
		}
		if err := m.prepareSubscriptionConfig(p, modes, ps.LanProxy, ps.IPv6, ports); err != nil {
			return err
		}

	case "profile":
		if p.ProfileID == "" {
			return fmt.Errorf("profileId is required")
		}
		if err := m.prepareProfileConfig(p, ports); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown config mode %q", p.ConfigMode)
	}

	// Patch the run-time config: inject managed inbounds, experimental, log.
	// This runs for every config mode so the settings always take effect.
	if err := patchConfig(m.RunConfigPath(), modes, ss, ps.LanProxy); err != nil {
		return fmt.Errorf("patch config: %w", err)
	}

	gid, err := ensureSingaGroup()
	if err != nil {
		return fmt.Errorf("singa group: %w", err)
	}

	var ipf ipfilter.Config
	_ = m.ipfilterStore.Load(&ipf)

	fwPorts := firewall.Ports{
		DNS:      ports.DNS,
		TProxy:   ports.TProxy,
		Redirect: ports.Redirect,
	}
	if err := firewall.Apply(modes, fwPorts, ps.LanProxy, ps.IPv6, ss.Inbound.TunInterface, m.dataDir, gid, ipf); err != nil {
		return fmt.Errorf("firewall: %w", err)
	}

	cmd := exec.Command(singboxBin, "run", "-D", m.runDir)
	cmd.Dir = m.runDir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid:         0,
			Gid:         gid,
			Groups:      []uint32{gid},
			NoSetGroups: false,
		},
	}
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		firewall.Stop()
		return fmt.Errorf("start sing-box: %w", err)
	}

	m.cmd = cmd
	m.state = StateRunning
	m.errMsg = ""
	m.params = p
	m.activeProxySettings = ps
	m.saveState(true)

	// System proxy: only when both TCP and UDP are "off" (no transparent proxy).
	if ps.isSystemProxyOnly() {
		if err := sysproxy.Set(ports.Mixed); err != nil {
			m.appendLog("warn: set system proxy: " + err.Error())
		} else {
			m.appendLog(fmt.Sprintf("system proxy set: http/https -> 127.0.0.1:%d", ports.Mixed))
		}
	}

	go m.streamLog(stdout)
	go m.streamLog(stderr)
	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		firewall.Stop()
		if m.activeProxySettings.isSystemProxyOnly() {
			if err := sysproxy.Clear(); err != nil {
				log.Printf("warn: clear system proxy: %v", err)
			}
		}
		if err != nil {
			m.errMsg = err.Error()
			m.state = StateError
			m.appendLog("sing-box exited: " + err.Error())
		} else {
			m.state = StateStopped
			m.appendLog("sing-box stopped")
		}
		m.saveState(false)
		m.cmd = nil
	}()

	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	proc := m.cmd
	m.mu.Unlock()

	if proc != nil && proc.Process != nil {
		_ = proc.Process.Kill()
		done := make(chan struct{})
		go func() { _ = proc.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	firewall.Stop()
	if m.activeProxySettings.isSystemProxyOnly() {
		if err := sysproxy.Clear(); err != nil {
			log.Printf("warn: clear system proxy: %v", err)
		}
	}
	m.state = StateStopped
	m.cmd = nil
	m.saveState(false)
}

// ── Config preparation ─────────────────────────────────────────────────────

func (m *Manager) GetSubManager() *subscription.Manager {
	return m.subManager
}

func (m *Manager) GetProfileManager() *profile.Manager {
	return m.profileManager
}

func (m *Manager) prepareProfileConfig(p StartParams, ports builder.Ports) error {
	prof := m.profileManager.GetByID(p.ProfileID)
	if prof == nil {
		return fmt.Errorf("profile %q not found", p.ProfileID)
	}
	if prof.WizardConfig == nil {
		return fmt.Errorf("profile %q has no wizard config — complete the wizard first", p.ProfileID)
	}

	// Scan WizardConfig outbounds for subscription references and collect
	// proxy nodes from every referenced subscription (deduped by tag).
	proxies, err := m.collectProxiesFromWizard(prof.WizardConfig)
	if err != nil {
		return fmt.Errorf("subscription cache: %w", err)
	}

	data, err := builder.BuildConfigFromWizard(prof.WizardConfig, proxies)
	if err != nil {
		return fmt.Errorf("build config from profile: %w", err)
	}
	return os.WriteFile(m.RunConfigPath(), data, 0644)
}

// collectProxiesFromWizard parses the wizard JSON, finds every outbound ref
// whose type is "Subscription"/"Subscribe", and returns a SubProxies map:
// subscription ID → ordered list of proxy nodes for that subscription.
// Each subscription is fetched only once even if referenced by multiple outbounds.
func (m *Manager) collectProxiesFromWizard(raw []byte) (builder.SubProxies, error) {
	var wc struct {
		Outbounds []struct {
			Outbounds []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"outbounds"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal(raw, &wc); err != nil {
		return nil, fmt.Errorf("parse wizard config: %w", err)
	}

	seen := map[string]bool{} // dedupe subscription IDs already fetched
	result := builder.SubProxies{}

	for _, ob := range wc.Outbounds {
		for _, ref := range ob.Outbounds {
			if ref.Type != "Subscription" && ref.Type != "Subscribe" {
				continue
			}
			if seen[ref.ID] {
				continue
			}
			seen[ref.ID] = true
			nodes, err := m.subManager.GetProxies(ref.ID)
			if err != nil {
				return nil, fmt.Errorf("subscription %q: %w", ref.ID, err)
			}
			result[ref.ID] = nodes
		}
	}
	return result, nil
}

func (m *Manager) prepareSubscriptionConfig(p StartParams, modes config.ProxyModes, lanProxy bool, ipv6 bool, ports builder.Ports) error {
	sub := m.subManager.GetByID(p.SubscriptionID)
	if sub == nil {
		return fmt.Errorf("subscription %q not found", p.SubscriptionID)
	}
	nodes, err := m.subManager.GetProxies(p.SubscriptionID)
	if err != nil {
		return fmt.Errorf("subscription cache: %w", err)
	}
	// For a standalone subscription config, wrap nodes under the subscription ID
	subProxies := builder.SubProxies{p.SubscriptionID: nodes}
	data, err := builder.BuildConfigFromWizard(sub.WizardConfig, subProxies)
	if err != nil {
		return fmt.Errorf("build config from wizard: %w", err)
	}
	return os.WriteFile(m.RunConfigPath(), data, 0644)
}

func (m *Manager) prepareUploadConfig() error {
	if _, err := os.Stat(m.ConfigPath()); os.IsNotExist(err) {
		return fmt.Errorf("config.json not uploaded")
	}
	return copyFile(m.ConfigPath(), m.RunConfigPath())
}

func (m *Manager) prepareNodeConfig(p StartParams, modes config.ProxyModes, lanProxy bool, ipv6 bool, n *node.Node, ports builder.Ports) error {
	data, err := builder.BuildConfig(modes, p.RouteMode, n, ports, lanProxy, ipv6, m.srsDir, IsReF1ndBuild(), p.BlockAds)
	if err != nil {
		return fmt.Errorf("build config: %w", err)
	}
	return os.WriteFile(m.RunConfigPath(), data, 0644)
}

func (m *Manager) findNode(id string) *node.Node {
	for _, n := range m.nodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}

// ── Settings persistence ───────────────────────────────────────────────────

func (m *Manager) GetIPFilter() ipfilter.Config {
	var cfg ipfilter.Config
	_ = m.ipfilterStore.Load(&cfg)
	if cfg.Mode == "" {
		cfg.Mode = ipfilter.ModeOff
	}
	return cfg
}

func (m *Manager) SaveIPFilter(cfg ipfilter.Config) error {
	return m.ipfilterStore.Save(&cfg)
}

func (m *Manager) loadProxySettings() ProxySettings {
	var ps ProxySettings
	_ = m.proxySettingsStore.Load(&ps)
	if ps.TCPMode == "" {
		ps.TCPMode = config.TCPModeOff
	}
	if ps.UDPMode == "" {
		ps.UDPMode = config.UDPModeOff
	}
	return ps
}

func (m *Manager) GetProxySettings() ProxySettings {
	return m.loadProxySettings()
}

func (m *Manager) SaveProxySettings(ps ProxySettings) error {
	return m.proxySettingsStore.Save(&ps)
}

func (m *Manager) loadSingaSettings() SingaSettings {
	var ss SingaSettings
	_ = m.singaSettingsStore.Load(&ss)
	return ss.Filled()
}

func (m *Manager) GetSingaSettings() SingaSettings {
	return m.loadSingaSettings()
}

func (m *Manager) SaveSingaSettings(ss SingaSettings) error {
	return m.singaSettingsStore.Save(&ss)
}

// ── Status ─────────────────────────────────────────────────────────────────

func (m *Manager) Status() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	pid := 0
	if m.cmd != nil && m.cmd.Process != nil {
		pid = m.cmd.Process.Pid
	}
	ps := m.activeProxySettings
	return map[string]interface{}{
		"state":      m.state,
		"configMode": m.params.ConfigMode,
		"tcpMode":    ps.TCPMode,
		"udpMode":    ps.UDPMode,
		"lanProxy":   ps.LanProxy,
		"ipv6":       ps.IPv6,
		"routeMode":  m.params.RouteMode,
		"blockAds":   m.params.BlockAds,
		"nodeId":     m.params.NodeID,
		"pid":        pid,
		"ports":      m.ports,
		"error":      m.errMsg,
	}
}

// ── Logging ────────────────────────────────────────────────────────────────

func (m *Manager) RecentLogs(n int) []string {
	m.logMu.RLock()
	defer m.logMu.RUnlock()
	if n > len(m.logBuf) {
		n = len(m.logBuf)
	}
	out := make([]string, n)
	copy(out, m.logBuf[len(m.logBuf)-n:])
	return out
}

func (m *Manager) SubscribeLogs() chan string {
	ch := make(chan string, 128)
	m.logMu.Lock()
	m.logSubs = append(m.logSubs, ch)
	m.logMu.Unlock()
	return ch
}

func (m *Manager) UnsubscribeLogs(ch chan string) {
	m.logMu.Lock()
	defer m.logMu.Unlock()
	subs := m.logSubs[:0]
	for _, s := range m.logSubs {
		if s != ch {
			subs = append(subs, s)
		}
	}
	m.logSubs = subs
	close(ch)
}

func (m *Manager) streamLog(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[sing-box] %s", line)
		m.appendLog(line)
	}
}

func (m *Manager) appendLog(line string) {
	m.logMu.Lock()
	defer m.logMu.Unlock()
	if len(m.logBuf) >= 500 {
		m.logBuf = m.logBuf[1:]
	}
	m.logBuf = append(m.logBuf, line)
	for _, ch := range m.logSubs {
		select {
		case ch <- line:
		default:
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func (m *Manager) RecoverState() {
	var ss savedState
	if err := m.stateStore.Load(&ss); err != nil || !ss.Running {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil {
		log.Printf("singa: stale running=true in state.json, correcting to stopped")
		ss.Running = false
		if err := m.stateStore.Save(&ss); err != nil {
			log.Printf("singa: recover state: %v", err)
		}
	}
}

