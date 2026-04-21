package firewall

import (
	"encoding/binary"
	"log"
	"net"
	"sync"
	"syscall"
)

// LocalIPWatcher watches for network address changes via Linux netlink
// (RTNLGRP_IPV4_IFADDR | RTNLGRP_IPV6_IFADDR) and syncs the current set
// of local CIDRs into nftables on every change.
//
// No polling — the kernel notifies us on RTM_NEWADDR / RTM_DELADDR.
type LocalIPWatcher struct {
	mu          sync.Mutex
	pool        map[string]struct{}
	AddedFunc   func(cidr string)
	RemovedFunc func(cidr string)
	fd          int
	done        chan struct{}
	wg          sync.WaitGroup
}

// NewLocalIPWatcher opens a netlink socket, does an initial sync, then
// starts a goroutine that wakes on every address-change event.
func NewLocalIPWatcher(added, removed func(cidr string)) *LocalIPWatcher {
	w := &LocalIPWatcher{
		pool:        make(map[string]struct{}),
		AddedFunc:   added,
		RemovedFunc: removed,
		done:        make(chan struct{}),
		fd:          -1,
	}

	fd, err := openNetlink()
	if err != nil {
		log.Printf("firewall: netlink unavailable, falling back to 30s poll: %v", err)
		w.startFallback()
		return w
	}
	w.fd = fd

	w.sync()

	w.wg.Add(1)
	go w.netlinkLoop()
	return w
}

func (w *LocalIPWatcher) Close() {
	close(w.done)
	if w.fd >= 0 {
		syscall.Close(w.fd)
	}
	w.wg.Wait()
}

// netlinkLoop reads netlink messages and re-syncs on every address event.
func (w *LocalIPWatcher) netlinkLoop() {
	defer w.wg.Done()
	buf := make([]byte, 4096)
	for {
		n, err := syscall.Read(w.fd, buf)
		if err != nil {
			select {
			case <-w.done:
			default:
				log.Printf("firewall: netlink read: %v", err)
			}
			return
		}
		if isAddrEvent(buf[:n]) {
			w.sync()
		}
	}
}

// isAddrEvent returns true if the buffer contains RTM_NEWADDR or RTM_DELADDR.
func isAddrEvent(buf []byte) bool {
	for len(buf) >= syscall.NLMSG_HDRLEN {
		msgLen := binary.LittleEndian.Uint32(buf[0:4])
		if msgLen < syscall.NLMSG_HDRLEN || int(msgLen) > len(buf) {
			break
		}
		msgType := binary.LittleEndian.Uint16(buf[4:6])
		if msgType == syscall.RTM_NEWADDR || msgType == syscall.RTM_DELADDR {
			return true
		}
		aligned := (msgLen + 3) &^ 3
		if int(aligned) > len(buf) {
			break
		}
		buf = buf[aligned:]
	}
	return false
}

// openNetlink creates a NETLINK_ROUTE socket subscribed to IPv4 and IPv6
// address-change multicast groups.
func openNetlink() (int, error) {
	fd, err := syscall.Socket(
		syscall.AF_NETLINK,
		syscall.SOCK_RAW|syscall.SOCK_CLOEXEC,
		syscall.NETLINK_ROUTE,
	)
	if err != nil {
		return -1, err
	}
	addr := &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Groups: syscall.RTNLGRP_IPV4_IFADDR | syscall.RTNLGRP_IPV6_IFADDR,
	}
	if err := syscall.Bind(fd, addr); err != nil {
		syscall.Close(fd)
		return -1, err
	}
	return fd, nil
}

// sync diffs current interface CIDRs against the known pool and calls
// AddedFunc / RemovedFunc for each change.
func (w *LocalIPWatcher) sync() {
	cidrs, err := localCIDRs()
	if err != nil {
		log.Printf("firewall: watcher sync: %v", err)
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

// startFallback is used when netlink is unavailable (e.g. unprivileged containers).
func (w *LocalIPWatcher) startFallback() {
	w.sync()
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			// sleep 30s interruptible by done
			ts := syscall.Timespec{Sec: 30}
			syscall.Nanosleep(&ts, nil)
			select {
			case <-w.done:
				return
			default:
				w.sync()
			}
		}
	}()
}

// localCIDRs returns all CIDRs currently assigned to local interfaces.
func localCIDRs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			out = append(out, ipnet.String())
		}
	}
	return out, nil
}
