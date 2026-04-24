package firewall

import (
	"fmt"
	"log"
	"sync"

	"github.com/singa/internal/config"
	"github.com/singa/internal/ipfilter"
)

var mu sync.Mutex

// Apply sets up nftables rules for the chosen proxy mode.
func Apply(mode config.ProxyMode, port int, dnsPort int, lanProxy bool, ipv6 bool, dataDir string, gid uint32, ipf ipfilter.Config) error {
	mu.Lock()
	defer mu.Unlock()

	SetNftConfPath(dataDir)
	Cleanup()

	switch mode {
	case config.ModeTProxy:
		if err := setupTproxy(port, dnsPort, lanProxy, ipv6, gid, ipf); err != nil {
			return fmt.Errorf("tproxy setup: %w", err)
		}
	case config.ModeRedirect:
		if err := setupRedirect(port, dnsPort, lanProxy, ipv6, gid, ipf); err != nil {
			return fmt.Errorf("redirect setup: %w", err)
		}
	case config.ModeTun:
		if err := setupTun(dnsPort, lanProxy, ipv6, gid, ipf); err != nil {
			return fmt.Errorf("tun setup: %w", err)
		}
	case config.ModeSystemProxy:
		log.Println("firewall: system_proxy mode — no nftables rules")
		return nil
	default:
		return fmt.Errorf("unknown proxy mode: %q", mode)
	}

	// Sync local interface IPs into nftables sets once at startup.
	SyncLocalIPs()
	return nil
}

// Stop tears down nftables rules.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	Cleanup()
}
