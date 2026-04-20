package firewall

import (
	"log"
	"net"
	"sync"
	"time"
)

const watchInterval = 3 * time.Second

// LocalIPWatcher periodically syncs local interface CIDRs into nftables sets.
type LocalIPWatcher struct {
	ticker      *time.Ticker
	pool        map[string]struct{}
	mu          sync.Mutex
	AddedFunc   func(cidr string)
	RemovedFunc func(cidr string)
	done        chan struct{}
}

func NewLocalIPWatcher(added, removed func(cidr string)) *LocalIPWatcher {
	w := &LocalIPWatcher{
		ticker:      time.NewTicker(watchInterval),
		pool:        make(map[string]struct{}),
		AddedFunc:   added,
		RemovedFunc: removed,
		done:        make(chan struct{}),
	}
	w.sync()
	go w.loop()
	return w
}

func (w *LocalIPWatcher) Close() {
	w.ticker.Stop()
	close(w.done)
}

func (w *LocalIPWatcher) loop() {
	for {
		select {
		case <-w.ticker.C:
			w.sync()
		case <-w.done:
			return
		}
	}
}

func (w *LocalIPWatcher) sync() {
	cidrs, err := localCIDRs()
	if err != nil {
		log.Printf("watcher: %v", err)
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	current := make(map[string]struct{}, len(cidrs))
	for _, cidr := range cidrs {
		current[cidr] = struct{}{}
		if _, ok := w.pool[cidr]; !ok {
			w.AddedFunc(cidr)
		}
	}
	for cidr := range w.pool {
		if _, ok := current[cidr]; !ok {
			w.RemovedFunc(cidr)
		}
	}
	w.pool = current
}

func localCIDRs() ([]string, error) {
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
