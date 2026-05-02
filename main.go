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
)

//go:embed web/dist
var webFS embed.FS

//go:embed assets/srs
var srsFS embed.FS

//go:embed assets/cn-bypass.nft
var cnBypassNft []byte

func main() {
	var (
		dirFlag  string
		portFlag int
	)
	flag.StringVar(&dirFlag, "dir", "", "data directory (default: <exe-dir>/data)")
	flag.IntVar(&portFlag, "port", 0, "web UI port (default: 7777)")
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

	listen := ":7777"
	if portFlag != 0 {
		listen = fmt.Sprintf(":%d", portFlag)
	}

	runDir := filepath.Join(dataDir, "run")
	srsDir := filepath.Join(dataDir, "srs")
	configsDir := filepath.Join(dataDir, "configs")
	nodesDir := filepath.Join(dataDir, "nodes")
	profilesDir := filepath.Join(dataDir, "profiles")

	for _, d := range []string{dataDir, runDir, srsDir, configsDir, nodesDir, profilesDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			log.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Extract embedded .srs files to data/srs/ (skips if already present)
	if err := extractSRS(srsFS, srsDir); err != nil {
		log.Printf("warn: extract srs: %v", err)
	}

	// Extract cn-bypass.nft to dataDir (skips if already present)
	cnNftDst := filepath.Join(dataDir, "cn-bypass.nft")
	if _, statErr := os.Stat(cnNftDst); os.IsNotExist(statErr) {
		if err := os.WriteFile(cnNftDst, cnBypassNft, 0644); err != nil {
			log.Printf("warn: extract cn-bypass.nft: %v", err)
		} else {
			log.Printf("singa: extracted cn-bypass.nft -> %s", cnNftDst)
		}
	}

	manager := core.NewManager(dataDir, runDir, srsDir)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		sig := <-sigCh
		log.Printf("singa: signal %v — shutting down", sig)
		manager.Stop()
		os.Exit(0)
	}()

	manager.RecoverState()
	manager.AutoStart()

	srv := api.NewServer(manager, dataDir, srsDir, webFS)
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
