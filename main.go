package main

import (
	"embed"
	"flag"
	"fmt"
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

//go:embed assets/srs
var srsFS embed.FS

func main() {
	var (
		dirFlag  string
		portFlag int
	)
	flag.StringVar(&dirFlag, "dir", "", "data directory (default: <exe-dir>/data)")
	flag.IntVar(&portFlag, "port", 0, "web UI port (default: 8080)")
	flag.Parse()

	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("executable path: %v", err)
	}
	baseDir := filepath.Dir(exe)

	dataDir := filepath.Join(baseDir, "data")
	if dirFlag != "" {
		abs, err := filepath.Abs(dirFlag)
		if err != nil {
			log.Fatalf("invalid --dir: %v", err)
		}
		dataDir = abs
	}

	listen := ":8080"
	if portFlag != 0 {
		listen = fmt.Sprintf(":%d", portFlag)
	}

	runDir := filepath.Join(dataDir, "run")
	srsDir := filepath.Join(dataDir, "srs")

	for _, d := range []string{dataDir, runDir, srsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			log.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Extract embedded .srs files to data/srs/ (skips if already present)
	if err := extractSRS(srsFS, srsDir); err != nil {
		log.Printf("warn: extract srs: %v", err)
	}

	manager := core.NewManager(dataDir, runDir, srsDir)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		sig := <-sigCh
		log.Printf("singa: signal %v — shutting down", sig)
		manager.Stop()
		firewall.Cleanup()
		os.Exit(0)
	}()

	srv := api.NewServer(manager, dataDir, webFS)
	log.Printf("singa: listening on %s  data=%s", listen, dataDir)
	if err := srv.Run(listen); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func extractSRS(efs embed.FS, dst string) error {
	entries, err := efs.ReadDir("assets/srs")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		target := filepath.Join(dst, e.Name())
		if _, err := os.Stat(target); err == nil {
			continue // already extracted
		}
		data, err := efs.ReadFile("assets/srs/" + e.Name())
		if err != nil {
			return err
		}
		if len(data) == 0 {
			continue // skip stub/empty files
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
		log.Printf("singa: extracted srs/%s", e.Name())
	}
	return nil
}
