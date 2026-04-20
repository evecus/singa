package firewall

import (
	"fmt"
	"log"
	"sync"

	"github.com/singa/internal/config"
)

var (
	mu      sync.Mutex
	watcher *LocalIPWatcher
)

// Apply sets up nftables rules for the chosen proxy mode.
func Apply(mode config.ProxyMode, port int, lanProxy bool, ipv6 bool, dataDir string) error {
	mu.Lock()
	defer mu.Unlock()

	SetNftConfPath(dataDir)
	Cleanup()

	switch mode {
	case config.ModeTProxy:
		if err := setupTproxy(port, lanProxy, ipv6); err != nil {
			return fmt.Errorf("tproxy setup: %w", err)
		}
	case config.ModeRedirect:
		if err := setupRedirect(port, lanProxy, ipv6); err != nil {
			return fmt.Errorf("redirect setup: %w", err)
		}
	case config.ModeTun:
		log.Println("firewall: tun mode — rules managed by sing-box")
		return nil
	case config.ModeSystemProxy:
		log.Println("firewall: system_proxy mode — no nftables rules")
		return nil
	default:
		return fmt.Errorf("unknown proxy mode: %q", mode)
	}

	if watcher != nil {
		watcher.Close()
	}
	watcher = NewLocalIPWatcher(AddInterfaceIP, RemoveInterfaceIP)
	return nil
}

// Stop tears down nftables rules and stops the IP watcher.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if watcher != nil {
		watcher.Close()
		watcher = nil
	}
	Cleanup()
}
