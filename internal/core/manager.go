package core

import (
	"bufio"
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
	"github.com/singa/internal/node"
	"github.com/singa/internal/storage"
	"github.com/singa/internal/sysproxy"
)

const singboxBin = "/usr/bin/sing-box"

// singaGroup is the dedicated system group used to identify sing-box traffic.
const singaGroup = "singa"

// IsReF1ndBuild returns true when the installed sing-box binary was built
// with the reF1nd fork (identified by "reF1nd" in the version string).
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
type StartParams struct {
	ProxyMode  config.ProxyMode  `json:"proxyMode"`
	LanProxy   bool              `json:"lanProxy"`
	IPv6       bool              `json:"ipv6"`
	BlockAds   bool              `json:"blockAds"`
	RouteMode  builder.RouteMode `json:"routeMode"`
	NodeID     string            `json:"nodeId"`
	ConfigMode string            `json:"configMode"` // "upload" | "node"
}

// savedState is persisted to data/state.json to survive restarts.
type savedState struct {
	Params  StartParams `json:"params"`
	Running bool        `json:"running"`
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

	nodeStore  *storage.Store
	stateStore *storage.Store
	nodes      []*node.Node

	logMu   sync.RWMutex
	logBuf  []string
	logSubs []chan string
}

func NewManager(dataDir, runDir, srsDir string) *Manager {
	configsDir := filepath.Join(dataDir, "configs")
	m := &Manager{
		dataDir:    dataDir,
		runDir:     runDir,
		srsDir:     srsDir,
		configsDir: configsDir,
		state:      StateStopped,
		logBuf:     make([]string, 0, 500),
		nodeStore:  storage.New(dataDir, "nodes.json"),
		stateStore: storage.New(dataDir, "state.json"),
	}
	m.loadNodes()
	// Load last saved params so Status() can return them before AutoStart.
	var ss savedState
	if err := m.stateStore.Load(&ss); err == nil {
		m.params = ss.Params
	}
	return m
}

func (m *Manager) ConfigPath() string    { return filepath.Join(m.configsDir, "config.json") }
func (m *Manager) RunConfigPath() string { return filepath.Join(m.runDir, "config.json") }

// AutoStart reads state.json and re-launches sing-box if it was running
// before the last shutdown. Call this once after NewManager, when the
// server is fully initialised.
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

// saveState persists current params and running flag. Must be called with m.mu held.
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

// ── group helpers ──────────────────────────────────────────────────────────

// ensureSingaGroup looks up the singa system group, creating it if it
// does not exist. Returns the numeric GID.
// Compatible with standard Linux (groupadd) and OpenWrt (writes /etc/group directly).
func ensureSingaGroup() (uint32, error) {
	// Fast path: group already exists.
	if g, err := user.LookupGroup(singaGroup); err == nil {
		gid, err := strconv.ParseUint(g.Gid, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse gid %q: %w", g.Gid, err)
		}
		return uint32(gid), nil
	}

	log.Printf("group %q not found, creating", singaGroup)

	// Try groupadd (standard Linux / Debian).
	if path, err := exec.LookPath("groupadd"); err == nil {
		out, err := exec.Command(path, "--system", singaGroup).CombinedOutput()
		if err != nil {
			return 0, fmt.Errorf("groupadd: %w (output: %s)", err, strings.TrimSpace(string(out)))
		}
		// Re-lookup after groupadd.
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

	// Fallback: write /etc/group directly (OpenWrt / busybox systems).
	return writeGroupEntry(singaGroup)
}

// writeGroupEntry picks a free GID (starting from 500, skipping existing ones),
// appends "name:x:GID:" to /etc/group, and returns the chosen GID.
func writeGroupEntry(name string) (uint32, error) {
	const groupFile = "/etc/group"

	data, err := os.ReadFile(groupFile)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", groupFile, err)
	}

	// Collect all existing GIDs.
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

	// Pick first free GID starting from 500.
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

	switch p.ConfigMode {
	case "upload":
		if err := m.prepareUploadConfig(); err != nil {
			return err
		}
		cfg, err := config.ParseConfig(m.RunConfigPath())
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
		port, err := config.DetectPort(cfg, p.ProxyMode)
		if err != nil {
			return err
		}
		ports.TProxy = port
		ports.Redirect = port
		if dnsPort := config.DetectDNSPort(cfg); dnsPort > 0 {
			ports.DNS = dnsPort
		}
		m.ports = ports

	case "node":
		n := m.findNode(p.NodeID)
		if n == nil {
			return fmt.Errorf("node %q not found", p.NodeID)
		}
		if err := m.prepareNodeConfig(p, n, ports); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown config mode %q", p.ConfigMode)
	}

	// Ensure the dedicated singa group exists (creates it if needed).
	// nftables skgid rules use this GID to bypass sing-box's own traffic.
	gid, err := ensureSingaGroup()
	if err != nil {
		return fmt.Errorf("singa group: %w", err)
	}

	fwPort := ports.TProxy
	if p.ProxyMode == config.ModeRedirect {
		fwPort = ports.Redirect
	}
	if err := firewall.Apply(p.ProxyMode, fwPort, ports.DNS, p.LanProxy, p.IPv6, m.dataDir, gid); err != nil {
		return fmt.Errorf("firewall: %w", err)
	}

	cmd := exec.Command(singboxBin, "run", "-D", m.runDir)
	cmd.Dir = m.runDir
	// Run sing-box with the singa supplementary group so nftables skgid
	// rules can identify and bypass its own outbound traffic.
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
	m.saveState(true)

	if p.ProxyMode == config.ModeSystemProxy {
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
		if m.params.ProxyMode == config.ModeSystemProxy {
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
	// Grab the process pointer under the lock, then release before waiting.
	// The cmd.Wait() goroutine (started in Start) also needs the lock, so
	// holding it while waiting would deadlock.
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
	if m.params.ProxyMode == config.ModeSystemProxy {
		if err := sysproxy.Clear(); err != nil {
			log.Printf("warn: clear system proxy: %v", err)
		}
	}
	m.state = StateStopped
	m.cmd = nil
	m.saveState(false)
}

// ── Config preparation ─────────────────────────────────────────────────────

func (m *Manager) prepareUploadConfig() error {
	if _, err := os.Stat(m.ConfigPath()); os.IsNotExist(err) {
		return fmt.Errorf("config.json not uploaded")
	}
	return copyFile(m.ConfigPath(), m.RunConfigPath())
}

func (m *Manager) prepareNodeConfig(p StartParams, n *node.Node, ports builder.Ports) error {
	data, err := builder.BuildConfig(p.ProxyMode, p.RouteMode, n, ports, p.LanProxy, p.IPv6, m.srsDir, IsReF1ndBuild(), p.BlockAds)
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

// ── Status ─────────────────────────────────────────────────────────────────

func (m *Manager) Status() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	pid := 0
	if m.cmd != nil && m.cmd.Process != nil {
		pid = m.cmd.Process.Pid
	}
	return map[string]interface{}{
		"state":      m.state,
		"configMode": m.params.ConfigMode,
		"proxyMode":  m.params.ProxyMode,
		"routeMode":  m.params.RouteMode,
		"lanProxy":   m.params.LanProxy,
		"ipv6":       m.params.IPv6,
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

// RecoverState checks whether state.json claims running=true but sing-box
// is not actually running (unclean shutdown / crash). Corrects it to false
// so the UI shows the right state on next page load.
// Must be called at startup before AutoStart.
func (m *Manager) RecoverState() {
	var ss savedState
	if err := m.stateStore.Load(&ss); err != nil || !ss.Running {
		return
	}
	// At startup m.cmd is always nil — if state says running, it's stale.
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
