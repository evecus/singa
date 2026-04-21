package core

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/singa/internal/builder"
	"github.com/singa/internal/config"
	"github.com/singa/internal/firewall"
	"github.com/singa/internal/node"
	"github.com/singa/internal/storage"
)

const singboxBin = "/usr/bin/sing-box"

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
	RouteMode  builder.RouteMode `json:"routeMode"`
	NodeID     string            `json:"nodeId"`
	ConfigMode string            `json:"configMode"` // "upload" | "node"
}

// Manager controls the sing-box subprocess and firewall rules.
type Manager struct {
	mu      sync.Mutex
	dataDir string
	runDir  string
	srsDir  string

	cmd    *exec.Cmd
	state  State
	errMsg string
	params StartParams
	ports  builder.Ports

	nodeStore *storage.Store
	nodes     []*node.Node

	logMu   sync.RWMutex
	logBuf  []string
	logSubs []chan string
}

func NewManager(dataDir, runDir, srsDir string) *Manager {
	m := &Manager{
		dataDir:   dataDir,
		runDir:    runDir,
		srsDir:    srsDir,
		state:     StateStopped,
		logBuf:    make([]string, 0, 500),
		nodeStore: storage.New(dataDir, "nodes.json"),
	}
	m.loadNodes()
	return m
}

func (m *Manager) ConfigPath() string    { return filepath.Join(m.dataDir, "config.json") }
func (m *Manager) RunConfigPath() string { return filepath.Join(m.runDir, "config.json") }

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

	fwPort := ports.TProxy
	if p.ProxyMode == config.ModeRedirect {
		fwPort = ports.Redirect
	}
	if err := firewall.Apply(p.ProxyMode, fwPort, p.LanProxy, p.IPv6, m.dataDir); err != nil {
		return fmt.Errorf("firewall: %w", err)
	}

	cmd := exec.Command(singboxBin, "run", "-D", m.runDir)
	cmd.Dir = m.runDir
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

	go m.streamLog(stdout)
	go m.streamLog(stderr)
	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		firewall.Stop()
		if err != nil {
			m.errMsg = err.Error()
			m.state = StateError
			m.appendLog("sing-box exited: " + err.Error())
		} else {
			m.state = StateStopped
			m.appendLog("sing-box stopped")
		}
		m.cmd = nil
	}()

	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		done := make(chan struct{})
		go func() { _ = m.cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
	firewall.Stop()
	m.state = StateStopped
	m.cmd = nil
}

// ── Config preparation ─────────────────────────────────────────────────────

func (m *Manager) prepareUploadConfig() error {
	if _, err := os.Stat(m.ConfigPath()); os.IsNotExist(err) {
		return fmt.Errorf("config.json not uploaded")
	}
	return copyFile(m.ConfigPath(), m.RunConfigPath())
}

func (m *Manager) prepareNodeConfig(p StartParams, n *node.Node, ports builder.Ports) error {
	data, err := builder.BuildConfig(p.ProxyMode, p.RouteMode, n, ports, p.LanProxy, p.IPv6, m.srsDir)
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
