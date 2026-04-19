package main

import (
	"embed"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/singa/internal/api"
	"github.com/singa/internal/core"
	"github.com/singa/internal/firewall"
)

//go:embed web/dist
var webFS embed.FS

func main() {
	// Determine base directory (where the singa binary lives)
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("cannot resolve executable path: %v", err)
	}
	baseDir := filepath.Dir(exe)
	dataDir := filepath.Join(baseDir, "data")
	runDir := filepath.Join(dataDir, "run")

	for _, d := range []string{dataDir, runDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			log.Fatalf("cannot create directory %s: %v", d, err)
		}
	}

	manager := core.NewManager(dataDir, runDir)

	// Catch SIGTERM / SIGINT / SIGHUP — clean up before exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		sig := <-sigCh
		log.Printf("singa: received signal %v, shutting down...", sig)
		manager.Stop()
		firewall.Cleanup()
		os.Exit(0)
	}()

	srv := api.NewServer(manager, dataDir, runDir, webFS)
	log.Printf("singa: listening on :8080  (data=%s)", dataDir)
	if err := srv.Run(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
