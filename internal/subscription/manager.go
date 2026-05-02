package subscription

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager stores subscription metadata and node caches on disk.
type Manager struct {
	mu      sync.Mutex
	dataDir string
	list    []*Subscription
}

func NewManager(dataDir string) *Manager {
	m := &Manager{dataDir: dataDir}
	m.load()
	return m
}

func (m *Manager) metaPath() string { return filepath.Join(m.dataDir, "subscriptions.json") }
func (m *Manager) cachePath(id string) string {
	return filepath.Join(m.dataDir, "sub_"+id+".json")
}

func (m *Manager) load() {
	data, err := os.ReadFile(m.metaPath())
	if err != nil {
		m.list = []*Subscription{}
		return
	}
	_ = json.Unmarshal(data, &m.list)
	if m.list == nil {
		m.list = []*Subscription{}
	}
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.metaPath(), data, 0644)
}

// List returns all subscriptions (metadata only, no node details).
func (m *Manager) List() []*Subscription {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Subscription, len(m.list))
	copy(out, m.list)
	return out
}

// Add creates a new subscription entry (does not fetch nodes yet).
func (m *Manager) Add(name, url string, wizardConfig json.RawMessage) (*Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &Subscription{
		ID:           uuid.New().String(),
		Name:         name,
		URL:          url,
		WizardConfig: wizardConfig,
	}
	m.list = append(m.list, s)
	if err := m.save(); err != nil {
		m.list = m.list[:len(m.list)-1]
		return nil, err
	}
	return s, nil
}

// UpdateMeta updates name, url and wizardConfig without re-fetching.
func (m *Manager) UpdateMeta(id, name, url string, wizardConfig json.RawMessage) (*Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.findSub(id)
	if s == nil {
		return nil, fmt.Errorf("subscription %q not found", id)
	}
	s.Name = name
	s.URL = url
	if wizardConfig != nil {
		s.WizardConfig = wizardConfig
	}
	return s, m.save()
}

// Delete removes a subscription and its cache file.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := m.findIdx(id)
	if idx < 0 {
		return fmt.Errorf("subscription %q not found", id)
	}
	m.list = append(m.list[:idx], m.list[idx+1:]...)
	_ = os.Remove(m.cachePath(id))
	return m.save()
}

// Update fetches the subscription URL, parses nodes, saves cache, updates metadata.
func (m *Manager) Update(id string) (*Subscription, error) {
	m.mu.Lock()
	s := m.findSub(id)
	if s == nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("subscription %q not found", id)
	}
	url := s.URL
	m.mu.Unlock()

	proxies, err := Fetch(url)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	s = m.findSub(id)
	if s == nil {
		return nil, fmt.Errorf("subscription %q disappeared", id)
	}
	s.UpdatedAt = time.Now()
	s.Error = errStr
	if err == nil {
		s.NodeCount = len(proxies)
		data, _ := json.MarshalIndent(proxies, "", "  ")
		_ = os.WriteFile(m.cachePath(id), data, 0644)
	}
	_ = m.save()
	return s, err
}

// GetProxies reads the cached proxy list for a subscription.
func (m *Manager) GetProxies(id string) ([]map[string]any, error) {
	data, err := os.ReadFile(m.cachePath(id))
	if err != nil {
		return nil, fmt.Errorf("no cache for subscription %q — update it first", id)
	}
	var proxies []map[string]any
	if err := json.Unmarshal(data, &proxies); err != nil {
		return nil, fmt.Errorf("corrupt cache: %w", err)
	}
	return proxies, nil
}

// DeleteProxy removes a single proxy at index idx from the subscription cache.
func (m *Manager) DeleteProxy(id string, idx int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	proxies, err := m.readProxies(id)
	if err != nil {
		return err
	}
	if idx < 0 || idx >= len(proxies) {
		return fmt.Errorf("proxy index %d out of range", idx)
	}
	proxies = append(proxies[:idx], proxies[idx+1:]...)
	s := m.findSub(id)
	if s != nil {
		s.NodeCount = len(proxies)
		_ = m.save()
	}
	data, _ := json.MarshalIndent(proxies, "", "  ")
	return os.WriteFile(m.cachePath(id), data, 0644)
}

func (m *Manager) readProxies(id string) ([]map[string]any, error) {
	data, err := os.ReadFile(m.cachePath(id))
	if err != nil {
		return nil, fmt.Errorf("no cache for subscription %q", id)
	}
	var proxies []map[string]any
	if err := json.Unmarshal(data, &proxies); err != nil {
		return nil, fmt.Errorf("corrupt cache: %w", err)
	}
	return proxies, nil
}

// GetByID returns a single subscription's metadata.
func (m *Manager) GetByID(id string) *Subscription {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.findSub(id)
}

func (m *Manager) findIdx(id string) int {
	for i, s := range m.list {
		if s.ID == id {
			return i
		}
	}
	return -1
}

func (m *Manager) findSub(id string) *Subscription {
	for _, s := range m.list {
		if s.ID == id {
			return s
		}
	}
	return nil
}
