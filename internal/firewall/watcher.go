package firewall

import (
	"log"
	"net"
	"sync"
	"time"
)

const watchInterval = 3 * time.Second

// LocalIPWatcher periodically checks local interface CIDRs and calls
// AddedFunc / RemovedFunc when addresses appear or disappear.
type LocalIPWatcher struct {
	ticker      *time.Ticker
	cidrPool    map[string]struct{}
	mu          sync.Mutex
	AddedFunc   func(cidr string)
	RemovedFunc func(cidr string)
	done        chan struct{}
}

func NewLocalIPWatcher(added, removed func(cidr string)) *LocalIPWatcher {
	w := &LocalIPWatcher{
		ticker:      time.NewTicker(watchInterval),
		cidrPool:    make(map[string]struct{}),
		AddedFunc:   added,
		RemovedFunc: removed,
		done:        make(chan struct{}),
	}
	w.syncIP()
	go w.loop()
	return w
}

func (w *LocalIPWatcher) loop() {
	for {
		select {
		case <-w.ticker.C:
			w.syncIP()
		case <-w.done:
			return
		}
	}
}

func (w *LocalIPWatcher) Close() {
	w.ticker.Stop()
	close(w.done)
}

func (w *LocalIPWatcher) syncIP() {
	cidrs, err := getLocalCIDRs()
	if err != nil {
		log.Printf("watcher: getLocalCIDRs: %v", err)
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	current := make(map[string]struct{})
	for _, cidr := range cidrs {
		current[cidr] = struct{}{}
		if _, ok := w.cidrPool[cidr]; !ok {
			w.AddedFunc(cidr)
		}
	}
	for cidr := range w.cidrPool {
		if _, ok := current[cidr]; !ok {
			w.RemovedFunc(cidr)
		}
	}
	w.cidrPool = current
}

func getLocalCIDRs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			out = append(out, ipnet.String())
		}
	}
	return out, nil
}
