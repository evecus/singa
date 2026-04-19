package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/singa/internal/config"
	"github.com/singa/internal/core"
)

type Server struct {
	manager *core.Manager
	dataDir string
	webFS   embed.FS
}

func NewServer(m *core.Manager, dataDir, runDir string, webFS embed.FS) *Server {
	return &Server{manager: m, dataDir: dataDir, webFS: webFS}
}

func (s *Server) Run(addr string) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.Default())

	api := r.Group("/api")
	{
		api.POST("/config", s.uploadConfig)
		api.GET("/config/info", s.configInfo)
		api.POST("/start", s.start)
		api.POST("/stop", s.stop)
		api.GET("/status", s.status)
		api.GET("/logs", s.streamLogs)   // SSE
	}

	// Serve embedded Vue SPA
	dist, err := fs.Sub(s.webFS, "web/dist")
	if err != nil {
		return fmt.Errorf("embed web/dist: %w", err)
	}
	r.NoRoute(func(c *gin.Context) {
		// Try to serve from dist, fall back to index.html (SPA routing)
		p := c.Request.URL.Path
		f, err := dist.Open(p[1:]) // strip leading /
		if err == nil {
			defer f.Close()
			http.ServeContent(c.Writer, c.Request, filepath.Base(p), getModTime(f), f.(io.ReadSeeker))
			return
		}
		idx, err := dist.Open("index.html")
		if err != nil {
			c.Status(404)
			return
		}
		defer idx.Close()
		http.ServeContent(c.Writer, c.Request, "index.html", getModTime(idx), idx.(io.ReadSeeker))
	})

	return r.Run(addr)
}

// POST /api/config  — multipart upload of config.json
func (s *Server) uploadConfig(c *gin.Context) {
	file, err := c.FormFile("config")
	if err != nil {
		c.JSON(400, gin.H{"error": "no config file provided"})
		return
	}
	dst := filepath.Join(s.dataDir, "config.json")
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(500, gin.H{"error": "save config: " + err.Error()})
		return
	}
	// Validate it parses
	cfg, err := config.ParseConfig(dst)
	if err != nil {
		_ = os.Remove(dst)
		c.JSON(400, gin.H{"error": "invalid config.json: " + err.Error()})
		return
	}
	inbounds := make([]gin.H, 0, len(cfg.Inbounds))
	for _, ib := range cfg.Inbounds {
		inbounds = append(inbounds, gin.H{
			"type": ib.Type,
			"tag":  ib.Tag,
			"port": ib.ListenPort,
		})
	}
	c.JSON(200, gin.H{"ok": true, "inbounds": inbounds})
}

// GET /api/config/info  — parse existing config and return inbound summary
func (s *Server) configInfo(c *gin.Context) {
	path := filepath.Join(s.dataDir, "config.json")
	cfg, err := config.ParseConfig(path)
	if err != nil {
		c.JSON(404, gin.H{"error": "no valid config.json found"})
		return
	}
	inbounds := make([]gin.H, 0, len(cfg.Inbounds))
	for _, ib := range cfg.Inbounds {
		inbounds = append(inbounds, gin.H{
			"type": ib.Type,
			"tag":  ib.Tag,
			"port": ib.ListenPort,
		})
	}
	c.JSON(200, gin.H{"inbounds": inbounds})
}

type StartRequest struct {
	Mode     string `json:"mode"`
	LanProxy bool   `json:"lanProxy"`
}

// POST /api/start
func (s *Server) start(c *gin.Context) {
	var req StartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	mode := config.ProxyMode(req.Mode)
	switch mode {
	case config.ModeTProxy, config.ModeRedirect, config.ModeTun, config.ModeSystemProxy:
	default:
		c.JSON(400, gin.H{"error": fmt.Sprintf("unknown mode %q", req.Mode)})
		return
	}

	// Parse config to detect port
	cfgPath := filepath.Join(s.dataDir, "config.json")
	cfg, err := config.ParseConfig(cfgPath)
	if err != nil {
		c.JSON(400, gin.H{"error": "config.json missing or invalid — please upload first"})
		return
	}
	port, err := config.DetectPort(cfg, mode)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := s.manager.Start(mode, port, req.LanProxy); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// POST /api/stop
func (s *Server) stop(c *gin.Context) {
	s.manager.Stop()
	c.JSON(200, gin.H{"ok": true})
}

// GET /api/status
func (s *Server) status(c *gin.Context) {
	c.JSON(200, s.manager.Status())
}

// GET /api/logs  — Server-Sent Events
func (s *Server) streamLogs(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Connection", "keep-alive")

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.Status(500)
		return
	}

	// Send buffered history
	recent := s.manager.RecentLogs(100)
	for _, line := range recent {
		fmt.Fprintf(w, "data: %s\n\n", jsonEscape(line))
	}
	flusher.Flush()

	// Subscribe to new lines
	ch := s.manager.SubscribeLogs()
	defer s.manager.UnsubscribeLogs(ch)

	notify := c.Request.Context().Done()
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", jsonEscape(line))
			flusher.Flush()
		case <-notify:
			return
		}
	}
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1]) // strip surrounding quotes
}

func getModTime(f fs.File) time.Time {
	if fi, err := f.Stat(); err == nil {
		return fi.ModTime()
	}
	return time.Time{}
}
