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
// port is the transparent proxy inbound port (tproxy/redirect).
// dnsPort is the sing-box dns-in port to redirect DNS traffic into.
func Apply(mode config.ProxyMode, port int, dnsPort int, lanProxy bool, ipv6 bool, dataDir string) error {
	mu.Lock()
	defer mu.Unlock()

	SetNftConfPath(dataDir)
	Cleanup()

	switch mode {
	case config.ModeTProxy:
		if err := setupTproxy(port, dnsPort, lanProxy, ipv6); err != nil {
			return fmt.Errorf("tproxy setup: %w", err)
		}
	case config.ModeRedirect:
		if err := setupRedirect(port, dnsPort, lanProxy, ipv6); err != nil {
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
