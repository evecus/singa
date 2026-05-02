package firewall

import (
	"fmt"
	"log"
	"sync"

	"github.com/singa/internal/config"
	"github.com/singa/internal/ipfilter"
)

var mu sync.Mutex

// activeTunDevice remembers the tun interface name used in the last Apply()
// so that Stop() / Cleanup() can remove the correct ip route/rule entries.
var activeTunDevice string

// Ports holds the listen ports that nftables needs to know about.
type Ports struct {
	DNS      int
	TProxy   int
	Redirect int
}

// Apply sets up nftables rules for the chosen TCP/UDP proxy modes.
// tunDevice is the TUN interface name configured by the user (e.g. "singa",
// "tun0"). It is used in both the nft iifname match and the ip route rules.
func Apply(modes config.ProxyModes, ports Ports, lanProxy bool, ipv6 bool, bypassCN bool, tunDevice string, dataDir string, gid uint32, ipf ipfilter.Config) error {
	mu.Lock()
	defer mu.Unlock()

	SetNftConfPath(dataDir)
	cleanup(activeTunDevice)

	if tunDevice == "" {
		tunDevice = "singa"
	}
	activeTunDevice = tunDevice

	if modes.IsSystemProxyOnly() {
		log.Println("firewall: system_proxy only — no nftables rules")
		return nil
	}

	if err := setup(modes, ports, lanProxy, ipv6, bypassCN, tunDevice, gid, ipf); err != nil {
		return fmt.Errorf("nft setup: %w", err)
	}

	SyncLocalIPs()
	return nil
}

// Stop tears down nftables rules using the last known tun device name.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	cleanup(activeTunDevice)
	activeTunDevice = ""
}
