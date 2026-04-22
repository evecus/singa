package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/singa/internal/builder"
	"github.com/singa/internal/config"
	"github.com/singa/internal/core"
	"github.com/singa/internal/node"
	"github.com/singa/internal/singbox"
	"github.com/singa/internal/updater"
)

type Server struct {
	manager *core.Manager
	dataDir string
	srsDir  string
	webFS   embed.FS
}

func NewServer(m *core.Manager, dataDir string, srsDir string, webFS embed.FS) *Server {
	return &Server{manager: m, dataDir: dataDir, srsDir: srsDir, webFS: webFS}
}

func (s *Server) Run(addr string) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/api/status"},
	}), gin.Recovery(), cors.Default())

	a := r.Group("/api")
	{
		a.POST("/config", s.uploadConfig)
		a.GET("/config/info", s.configInfo)
		a.GET("/nodes", s.listNodes)
		a.POST("/nodes/import", s.importNodes)
		a.DELETE("/nodes/:id", s.deleteNode)
		a.POST("/start", s.start)
		a.POST("/stop", s.stop)
		a.GET("/status", s.status)
		a.GET("/logs", s.streamLogs)
		a.POST("/update-rules", s.updateRules)
		// Settings
		a.GET("/singbox/version", s.singboxVersion)
		a.POST("/singbox/install", s.singboxInstall)
		a.GET("/system-info", s.systemInfo)
	}

	dist, err := fs.Sub(s.webFS, "web/dist")
	if err != nil {
		return fmt.Errorf("embed web/dist: %w", err)
	}
	r.NoRoute(func(c *gin.Context) {
		serveDistFile(c, dist, c.Request.URL.Path)
	})

	return r.Run(addr)
}

func serveDistFile(c *gin.Context, dist fs.FS, path string) {
	p := path
	if len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	f, err := dist.Open(p)
	if err == nil {
		defer f.Close()
		fi, _ := f.Stat()
		if !fi.IsDir() {
			http.ServeContent(c.Writer, c.Request, fi.Name(), fi.ModTime(), f.(io.ReadSeeker))
			return
		}
	}
	idx, err := dist.Open("index.html")
	if err != nil {
		c.Status(404)
		return
	}
	defer idx.Close()
	fi, _ := idx.Stat()
	http.ServeContent(c.Writer, c.Request, "index.html", fi.ModTime(), idx.(io.ReadSeeker))
}

// ── Config upload ──────────────────────────────────────────────────────────

func (s *Server) uploadConfig(c *gin.Context) {
	file, err := c.FormFile("config")
	if err != nil {
		c.JSON(400, gin.H{"error": "no config file"})
		return
	}
	dst := s.manager.ConfigPath()
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	cfg, err := config.ParseConfig(dst)
	if err != nil {
		os.Remove(dst)
		c.JSON(400, gin.H{"error": "invalid config.json: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "inbounds": summarizeInbounds(cfg)})
}

func (s *Server) configInfo(c *gin.Context) {
	cfg, err := config.ParseConfig(s.manager.ConfigPath())
	if err != nil {
		c.JSON(404, gin.H{"error": "no valid config.json"})
		return
	}
	c.JSON(200, gin.H{"inbounds": summarizeInbounds(cfg)})
}

func summarizeInbounds(cfg *config.SingboxConfig) []gin.H {
	out := make([]gin.H, 0, len(cfg.Inbounds))
	for _, ib := range cfg.Inbounds {
		out = append(out, gin.H{"type": ib.Type, "tag": ib.Tag, "port": ib.ListenPort})
	}
	return out
}

// ── Nodes ──────────────────────────────────────────────────────────────────

func (s *Server) listNodes(c *gin.Context) {
	c.JSON(200, s.manager.GetNodes())
}

func (s *Server) importNodes(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Text == "" {
		c.JSON(400, gin.H{"error": "missing text"})
		return
	}
	nodes, errs := node.ParseLinks(req.Text)
	if len(nodes) > 0 {
		s.manager.AddNodes(nodes)
	}
	c.JSON(200, gin.H{"imported": len(nodes), "errors": errs, "nodes": nodes})
}

func (s *Server) deleteNode(c *gin.Context) {
	if !s.manager.DeleteNode(c.Param("id")) {
		c.JSON(404, gin.H{"error": "node not found"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// ── Start / Stop ───────────────────────────────────────────────────────────

func (s *Server) start(c *gin.Context) {
	var p core.StartParams
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	switch p.ProxyMode {
	case config.ModeTProxy, config.ModeRedirect, config.ModeTun, config.ModeSystemProxy:
	default:
		c.JSON(400, gin.H{"error": fmt.Sprintf("unknown proxyMode %q", p.ProxyMode)})
		return
	}
	switch p.ConfigMode {
	case "upload", "node":
	default:
		c.JSON(400, gin.H{"error": fmt.Sprintf("unknown configMode %q", p.ConfigMode)})
		return
	}
	if p.ConfigMode == "node" {
		switch p.RouteMode {
		case builder.RouteModeWhitelist, builder.RouteModeGFWList, builder.RouteModeGlobal:
		default:
			c.JSON(400, gin.H{"error": fmt.Sprintf("unknown routeMode %q", p.RouteMode)})
			return
		}
	}
	if err := s.manager.Start(p); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) stop(c *gin.Context) {
	s.manager.Stop()
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) status(c *gin.Context) {
	c.JSON(200, s.manager.Status())
}

// ── SSE ────────────────────────────────────────────────────────────────────

func (s *Server) streamLogs(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.Status(500)
		return
	}
	for _, line := range s.manager.RecentLogs(100) {
		fmt.Fprintf(c.Writer, "data: %s\n\n", sseEscape(line))
	}
	flusher.Flush()

	ch := s.manager.SubscribeLogs()
	defer s.manager.UnsubscribeLogs(ch)
	notify := c.Request.Context().Done()
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", sseEscape(line))
			flusher.Flush()
		case <-notify:
			return
		}
	}
}

func sseEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

var _ = time.Now

// ── Update rules ───────────────────────────────────────────────────────────

func (s *Server) updateRules(c *gin.Context) {
	var req struct {
		Proxy string `json:"proxy"`
	}
	// ignore parse error — proxy is optional
	_ = c.ShouldBindJSON(&req)

	results := updater.UpdateAll(s.srsDir, req.Proxy)

	failed := 0
	for _, r := range results {
		if r.Error != "" {
			failed++
		}
	}
	status := http.StatusOK
	if failed == len(results) {
		status = http.StatusBadGateway
	}
	c.JSON(status, gin.H{"results": results, "failed": failed, "total": len(results)})
}

// ── sing-box management ────────────────────────────────────────────────────

func (s *Server) singboxVersion(c *gin.Context) {
	ver := singbox.Version()
	sys := singbox.DetectSystem()
	c.JSON(200, gin.H{
		"version": ver,
		"arch":    sys.Arch,
		"libc":    sys.LibC,
		"osName":  sys.OSName,
	})
}

func (s *Server) singboxInstall(c *gin.Context) {
	var req struct {
		Proxy  string `json:"proxy"`
		Flavor string `json:"flavor"` // "official" or "ref1nd"
	}
	_ = c.ShouldBindJSON(&req)

	flavor := singbox.FlavorOfficial
	if req.Flavor == "ref1nd" {
		flavor = singbox.FlavorReF1nd
	}

	ver, err := singbox.Install(flavor, req.Proxy)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "version": ver})
}

func (s *Server) systemInfo(c *gin.Context) {
	sys := singbox.DetectSystem()
	c.JSON(200, sys)
}
