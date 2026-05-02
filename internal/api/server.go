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
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/singa/internal/builder"
	"github.com/singa/internal/config"
	"github.com/singa/internal/core"
	"github.com/singa/internal/ipfilter"
	"github.com/singa/internal/node"
	"github.com/singa/internal/singbox"
	"github.com/singa/internal/updater"
)

// errorOnlyFormatter is a gin LogFormatter that only prints 4xx/5xx responses.
var errorOnlyFormatter gin.LogFormatter = func(param gin.LogFormatterParams) string {
	if param.StatusCode < 400 {
		return ""
	}
	return fmt.Sprintf("[GIN] %s | %d | %s | %s | %s %s\n",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		param.StatusCode,
		param.Latency,
		param.ClientIP,
		param.Method,
		param.Path,
	)
}

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
		Formatter: errorOnlyFormatter,
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
		// Subscriptions
		a.GET("/subscriptions", s.listSubscriptions)
		a.POST("/subscriptions", s.addSubscription)
		a.DELETE("/subscriptions/:id", s.deleteSubscription)
		a.PATCH("/subscriptions/:id", s.updateSubscriptionMeta)
		a.POST("/subscriptions/:id/update", s.updateSubscription)
		a.GET("/subscriptions/:id/proxies", s.getSubscriptionProxies)
		a.DELETE("/subscriptions/:id/proxies/:idx", s.deleteSubscriptionProxy)
		// Profiles (independent of subscriptions)
		a.GET("/profiles", s.listProfiles)
		a.POST("/profiles", s.addProfile)
		a.PATCH("/profiles/:id", s.updateProfile)
		a.DELETE("/profiles/:id", s.deleteProfile)
		// Settings
		a.GET("/singbox/version", s.singboxVersion)
		a.POST("/singbox/install", s.singboxInstall)
		a.GET("/system-info", s.systemInfo)
		a.GET("/ip-filter", s.getIPFilter)
		a.POST("/ip-filter", s.saveIPFilter)
		a.GET("/proxy-settings", s.getProxySettings)
		a.POST("/proxy-settings", s.saveProxySettings)
		a.GET("/singa-settings", s.getSingaSettings)
		a.POST("/singa-settings", s.saveSingaSettings)
		// Rulesets
		a.GET("/rulesets", s.listRulesets)
		a.DELETE("/rulesets/:file", s.deleteRuleset)
		a.POST("/rulesets/download", s.downloadRuleset)
		a.GET("/rulesets/fetch-hub", s.fetchRulesetHub)
		// Config
		a.GET("/config/raw", s.rawConfig)
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
			// JS/CSS assets have content-hash in filename, cache forever
			if strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".css") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			}
			// Set explicit MIME types to prevent browser rejection of JS modules
			switch {
			case strings.HasSuffix(p, ".js"):
				c.Header("Content-Type", "application/javascript; charset=utf-8")
			case strings.HasSuffix(p, ".css"):
				c.Header("Content-Type", "text/css; charset=utf-8")
			case strings.HasSuffix(p, ".json"):
				c.Header("Content-Type", "application/json; charset=utf-8")
			case strings.HasSuffix(p, ".svg"):
				c.Header("Content-Type", "image/svg+xml")
			case strings.HasSuffix(p, ".png"):
				c.Header("Content-Type", "image/png")
			case strings.HasSuffix(p, ".ico"):
				c.Header("Content-Type", "image/x-icon")
			case strings.HasSuffix(p, ".webmanifest"):
				c.Header("Content-Type", "application/manifest+json")
			}
			http.ServeContent(c.Writer, c.Request, fi.Name(), fi.ModTime(), f.(io.ReadSeeker))
			return
		}
	}

	// For /assets/* requests that don't exist in the embed, return 404 instead
	// of falling back to index.html — this prevents the browser from receiving
	// HTML content when it expects a JavaScript module, which causes:
	// "Failed to fetch dynamically imported module"
	if strings.HasPrefix(p, "assets/") {
		c.Status(404)
		return
	}

	idx, err := dist.Open("index.html")
	if err != nil {
		c.Status(404)
		return
	}
	defer idx.Close()
	fi, _ := idx.Stat()
	// Never cache index.html so browsers always fetch fresh asset references
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Content-Type", "text/html; charset=utf-8")
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
	switch p.ConfigMode {
	case "upload", "node", "subnode", "subscription", "profile":
	default:
		c.JSON(400, gin.H{"error": fmt.Sprintf("unknown configMode %q", p.ConfigMode)})
		return
	}
	if p.ConfigMode == "node" || p.ConfigMode == "subnode" || p.ConfigMode == "subscription" {
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
		Proxy   string `json:"proxy"`
		Flavor  string `json:"flavor"`  // "official" or "ref1nd"
		Version string `json:"version"` // "latest" or e.g. "1.13.2"
	}
	_ = c.ShouldBindJSON(&req)

	flavor := singbox.FlavorOfficial
	if req.Flavor == "ref1nd" {
		flavor = singbox.FlavorReF1nd
	}
	if req.Version == "" {
		req.Version = "latest"
	}

	ver, err := singbox.Install(flavor, req.Proxy, req.Version)
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

func (s *Server) getIPFilter(c *gin.Context) {
	cfg := s.manager.GetIPFilter()
	c.JSON(200, cfg)
}

func (s *Server) saveIPFilter(c *gin.Context) {
	var cfg ipfilter.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	switch cfg.Mode {
	case ipfilter.ModeOff, ipfilter.ModeBlacklist, ipfilter.ModeWhitelist:
	default:
		c.JSON(400, gin.H{"error": "invalid mode"})
		return
	}
	if err := s.manager.SaveIPFilter(cfg); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) getProxySettings(c *gin.Context) {
	c.JSON(200, s.manager.GetProxySettings())
}

func (s *Server) saveProxySettings(c *gin.Context) {
	var ps core.ProxySettings
	if err := c.ShouldBindJSON(&ps); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	switch ps.TCPMode {
	case config.TCPModeOff, config.TCPModeRedir, config.TCPModeTProxy, config.TCPModeTun:
	default:
		c.JSON(400, gin.H{"error": "invalid tcpMode"})
		return
	}
	switch ps.UDPMode {
	case config.UDPModeOff, config.UDPModeTProxy, config.UDPModeTun:
	default:
		c.JSON(400, gin.H{"error": "invalid udpMode"})
		return
	}
	if err := s.manager.SaveProxySettings(ps); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) getSingaSettings(c *gin.Context) {
	c.JSON(200, s.manager.GetSingaSettings())
}

func (s *Server) saveSingaSettings(c *gin.Context) {
	var ss core.SingaSettings
	if err := c.ShouldBindJSON(&ss); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := s.manager.SaveSingaSettings(ss); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// ── Subscriptions ──────────────────────────────────────────────────────────

func (s *Server) listSubscriptions(c *gin.Context) {
	c.JSON(200, s.manager.GetSubManager().List())
}

func (s *Server) addSubscription(c *gin.Context) {
	var req struct {
		Name         string          `json:"name"`
		URL          string          `json:"url"`
		WizardConfig json.RawMessage `json:"wizardConfig"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" {
		c.JSON(400, gin.H{"error": "name and url are required"})
		return
	}
	if req.Name == "" {
		req.Name = req.URL
	}
	sub, err := s.manager.GetSubManager().Add(req.Name, req.URL, req.WizardConfig)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, sub)
}

func (s *Server) deleteSubscription(c *gin.Context) {
	if err := s.manager.GetSubManager().Delete(c.Param("id")); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) updateSubscription(c *gin.Context) {
	sub, err := s.manager.GetSubManager().Update(c.Param("id"))
	if err != nil {
		c.JSON(502, gin.H{"error": err.Error(), "sub": sub})
		return
	}
	c.JSON(200, sub)
}

func (s *Server) getSubscriptionProxies(c *gin.Context) {
	proxies, err := s.manager.GetSubManager().GetProxies(c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, proxies)
}

func (s *Server) deleteSubscriptionProxy(c *gin.Context) {
	idx, err := strconv.Atoi(c.Param("idx"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid index"})
		return
	}
	if err := s.manager.GetSubManager().DeleteProxy(c.Param("id"), idx); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) updateSubscriptionMeta(c *gin.Context) {
	var req struct {
		Name         string          `json:"name"`
		URL          string          `json:"url"`
		WizardConfig json.RawMessage `json:"wizardConfig"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	sub, err := s.manager.GetSubManager().UpdateMeta(c.Param("id"), req.Name, req.URL, req.WizardConfig)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, sub)
}

// ── Profiles ────────────────────────────────────────────────────────────────

func (s *Server) listProfiles(c *gin.Context) {
	c.JSON(200, s.manager.GetProfileManager().List())
}

func (s *Server) addProfile(c *gin.Context) {
	var req struct {
		Name         string          `json:"name"`
		WizardConfig json.RawMessage `json:"wizardConfig"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(400, gin.H{"error": "name is required"})
		return
	}
	p, err := s.manager.GetProfileManager().Add(req.Name, "", req.WizardConfig)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, p)
}

func (s *Server) updateProfile(c *gin.Context) {
	var req struct {
		Name         string          `json:"name"`
		WizardConfig json.RawMessage `json:"wizardConfig"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	p, err := s.manager.GetProfileManager().Update(c.Param("id"), req.Name, "", req.WizardConfig)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, p)
}

func (s *Server) deleteProfile(c *gin.Context) {
	if err := s.manager.GetProfileManager().Delete(c.Param("id")); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// ── Rulesets ───────────────────────────────────────────────────────────────

func (s *Server) listRulesets(c *gin.Context) {
	type entry struct {
		File      string    `json:"file"`
		Format    string    `json:"format"`
		Size      int64     `json:"size"`
		UpdatedAt time.Time `json:"updatedAt"`
	}
	var items []entry
	dirEntries, err := os.ReadDir(s.srsDir)
	if err != nil {
		c.JSON(200, []entry{})
		return
	}
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		var format string
		if strings.HasSuffix(name, ".srs") {
			format = "binary"
		} else if strings.HasSuffix(name, ".json") {
			format = "source"
		} else {
			continue
		}
		fi, err := de.Info()
		if err != nil {
			continue
		}
		items = append(items, entry{
			File:      name,
			Format:    format,
			Size:      fi.Size(),
			UpdatedAt: fi.ModTime(),
		})
	}
	if items == nil {
		items = []entry{}
	}
	c.JSON(200, items)
}

func (s *Server) deleteRuleset(c *gin.Context) {
	name := c.Param("file")
	// sanitise: no path traversal
	if name == "" || name[0] == '.' || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		c.JSON(400, gin.H{"error": "invalid file name"})
		return
	}
	path := filepath.Join(s.srsDir, name)
	if err := os.Remove(path); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// ── Download a single ruleset from URL ────────────────────────────────────

func (s *Server) downloadRuleset(c *gin.Context) {
	var req struct {
		URL  string `json:"url"`
		File string `json:"file"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" || req.File == "" {
		c.JSON(400, gin.H{"error": "url and file are required"})
		return
	}
	// sanitise filename
	if strings.ContainsAny(req.File, "/\\..") && !strings.HasSuffix(req.File, ".srs") && !strings.HasSuffix(req.File, ".json") {
		c.JSON(400, gin.H{"error": "invalid file name"})
		return
	}
	dst := filepath.Join(s.srsDir, filepath.Base(req.File))
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(req.URL)
	if err != nil {
		c.JSON(502, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		c.JSON(502, gin.H{"error": fmt.Sprintf("upstream HTTP %d", resp.StatusCode)})
		return
	}
	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close(); os.Remove(tmp)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	f.Close()
	if err := os.Rename(tmp, dst); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "file": filepath.Base(dst)})
}

// fetchRulesetHub proxies the hub JSON fetch through the backend so the
// browser doesn't hit CORS / network restrictions.
func (s *Server) fetchRulesetHub(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(400, gin.H{"error": "url is required"})
		return
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		c.JSON(502, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		c.JSON(502, gin.H{"error": fmt.Sprintf("upstream HTTP %d", resp.StatusCode)})
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/json; charset=utf-8", body)
}

// ── Raw config endpoint ───────────────────────────────────────────────────

func (s *Server) rawConfig(c *gin.Context) {
	data, err := os.ReadFile(s.manager.ConfigPath())
	if err != nil {
		c.JSON(404, gin.H{"error": "no config"})
		return
	}
	c.Data(200, "application/json; charset=utf-8", data)
}
