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

	"github.com/singa/internal/config"
	"github.com/singa/internal/firewall"
)

const singboxBin = "/usr/bin/sing-box"

// State represents the running state of the core.
type State string

const (
	StateStopped State = "stopped"
	StateRunning State = "running"
	StateError   State = "error"
)

// Manager controls the sing-box subprocess and firewall rules.
type Manager struct {
	mu       sync.Mutex
	dataDir  string
	runDir   string
	cmd      *exec.Cmd
	state    State
	errMsg   string
	mode     config.ProxyMode
	port     int
	lanProxy bool
	ipv6     bool

	// Log ring buffer (last 500 lines)
	logMu   sync.RWMutex
	logBuf  []string
	logSubs []chan string
}

func NewManager(dataDir, runDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
		runDir:  runDir,
		state:   StateStopped,
		logBuf:  make([]string, 0, 500),
	}
}

// ConfigPath returns where config.json should live.
func (m *Manager) ConfigPath() string {
	return m.dataDir + "/config.json"
}

// Start applies firewall rules then launches sing-box.
func (m *Manager) Start(mode config.ProxyMode, port int, lanProxy bool, ipv6 bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateRunning {
		return fmt.Errorf("already running")
	}
	srcConfig := m.ConfigPath()
	if _, err := os.Stat(srcConfig); os.IsNotExist(err) {
		return fmt.Errorf("config.json not found — please upload a configuration first")
	}

	// Copy config.json into run directory so sing-box can find it
	dstConfig := filepath.Join(m.runDir, "config.json")
	if err := copyFile(srcConfig, dstConfig); err != nil {
		return fmt.Errorf("copy config to run dir: %w", err)
	}

	m.mode = mode
	m.port = port
	m.lanProxy = lanProxy
	m.ipv6 = ipv6

	// Apply nftables rules
	if err := firewall.Apply(mode, port, lanProxy, ipv6, m.dataDir); err != nil {
		return fmt.Errorf("firewall: %w", err)
	}

	// Launch sing-box
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

	// Stream stdout + stderr into log buffer
	go m.streamLog(stdout)
	go m.streamLog(stderr)

	// Watch for process exit
	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		firewall.Stop()
		m.state = StateStopped
		if err != nil {
			m.errMsg = err.Error()
			m.state = StateError
			m.appendLog("sing-box exited with error: " + err.Error())
		} else {
			m.appendLog("sing-box exited cleanly")
		}
		m.cmd = nil
	}()

	return nil
}

// Stop kills sing-box and cleans up firewall rules.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		// Give it a moment to reap
		done := make(chan struct{})
		go func() {
			_ = m.cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
	firewall.Stop()
	m.state = StateStopped
	m.cmd = nil
}

// Status returns current state information.
func (m *Manager) Status() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	pid := 0
	if m.cmd != nil && m.cmd.Process != nil {
		pid = m.cmd.Process.Pid
	}
	return map[string]interface{}{
		"state":    m.state,
		"mode":     m.mode,
		"port":     m.port,
		"lanProxy": m.lanProxy,
		"ipv6":     m.ipv6,
		"pid":      pid,
		"error":    m.errMsg,
	}
}

// copyFile copies src to dst, creating dst if needed.
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



// RecentLogs returns the last n log lines.
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

// SubscribeLogs returns a channel that receives new log lines.
func (m *Manager) SubscribeLogs() chan string {
	ch := make(chan string, 128)
	m.logMu.Lock()
	m.logSubs = append(m.logSubs, ch)
	m.logMu.Unlock()
	return ch
}

// UnsubscribeLogs removes a subscriber channel.
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

func (m *Manager) streamLog(r interface{ Read([]byte) (int, error) }) {
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
