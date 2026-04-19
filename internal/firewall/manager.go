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
// dataDir is used to write the temp nft config file.
func Apply(mode config.ProxyMode, port int, lanProxy bool, dataDir string) error {
	mu.Lock()
	defer mu.Unlock()

	SetNftConfPath(dataDir)
	ipv6 := IsIPv6Supported()

	// Always clean up first to avoid stale rules
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
		// TUN is fully managed by sing-box (auto_route etc.), no nftables needed
		log.Println("firewall: tun mode — rules managed by sing-box")
		return nil
	case config.ModeSystemProxy:
		// system_proxy is handled outside nftables (gsettings / env vars)
		log.Println("firewall: system_proxy mode — no nftables rules")
		return nil
	default:
		return fmt.Errorf("unknown proxy mode: %q", mode)
	}

	// Start IP watcher to keep nftables interface sets in sync
	if watcher != nil {
		watcher.Close()
	}
	watcher = NewLocalIPWatcher(AddInterfaceIP, RemoveInterfaceIP)
	return nil
}

// Stop tears down all nftables rules and stops the watcher.
func Stop() {
	mu.Lock()
	defer mu.Unlock()

	if watcher != nil {
		watcher.Close()
		watcher = nil
	}
	Cleanup()
}
