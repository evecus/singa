package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	mu      sync.Mutex
	dataDir string
	list    []*Profile
}

func NewManager(dataDir string) *Manager {
	m := &Manager{dataDir: dataDir}
	m.load()
	return m
}

func (m *Manager) metaPath() string { return filepath.Join(m.dataDir, "profiles.json") }

func (m *Manager) load() {
	data, err := os.ReadFile(m.metaPath())
	if err != nil {
		m.list = []*Profile{}
		return
	}
	_ = json.Unmarshal(data, &m.list)
	if m.list == nil {
		m.list = []*Profile{}
	}
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.metaPath(), data, 0644)
}

func (m *Manager) List() []*Profile {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Profile, len(m.list))
	copy(out, m.list)
	return out
}

func (m *Manager) Add(name, subscriptionID string, wizardConfig json.RawMessage) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := &Profile{
		ID:             uuid.New().String(),
		Name:           name,
		SubscriptionID: subscriptionID,
		UpdatedAt:      time.Now(),
		WizardConfig:   wizardConfig,
	}
	m.list = append(m.list, p)
	if err := m.save(); err != nil {
		m.list = m.list[:len(m.list)-1]
		return nil, err
	}
	return p, nil
}

func (m *Manager) Update(id, name, subscriptionID string, wizardConfig json.RawMessage) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.find(id)
	if p == nil {
		return nil, fmt.Errorf("profile %q not found", id)
	}
	p.Name = name
	p.SubscriptionID = subscriptionID
	p.UpdatedAt = time.Now()
	if wizardConfig != nil {
		p.WizardConfig = wizardConfig
	}
	return p, m.save()
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.list {
		if p.ID == id {
			m.list = append(m.list[:i], m.list[i+1:]...)
			return m.save()
		}
	}
	return fmt.Errorf("profile %q not found", id)
}

func (m *Manager) GetByID(id string) *Profile {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.find(id)
}

func (m *Manager) find(id string) *Profile {
	for _, p := range m.list {
		if p.ID == id {
			return p
		}
	}
	return nil
}
