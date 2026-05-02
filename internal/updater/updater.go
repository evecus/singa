package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// srsFiles maps local filename → raw GitHub URL (no proxy prefix).
var srsFiles = map[string]string{
	"geoip-cn.srs":                "https://raw.githubusercontent.com/1715173329/IPCIDR-CHINA/refs/heads/rule-set/cn.srs",
	"geosite-cn.srs":              "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geosite/cn.srs",
	"geosite-gfw.srs":             "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geosite/gfw.srs",
	"geosite-geolocation-!cn.srs": "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geosite/geolocation-!cn.srs",
	"geoip-telegram.srs":          "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geoip/telegram.srs",
	"ads.srs":                     "https://raw.githubusercontent.com/privacy-protection-tools/anti-ad.github.io/master/docs/anti-ad-sing-box.srs",
}

// BuiltinMirrors are tried in order after a direct download fails.
var BuiltinMirrors = []string{
	"https://ghfast.top/",
	"https://gh-proxy.com/",
	"https://gh.ddlc.top/",
	"https://ghproxy.it/",
}

// Result reports the outcome of a single file update.
type Result struct {
	File   string `json:"file"`
	Mirror string `json:"mirror"`
	Error  string `json:"error,omitempty"`
}

// UpdateAll downloads all rule set files into srsDir.
// proxy is an optional custom GitHub proxy prefix (e.g. "https://mymirror.com/").
// If empty, only direct + builtin mirrors are tried.
func UpdateAll(srsDir string, proxy string) []Result {
	results := make([]Result, 0, len(srsFiles))
	for filename, rawURL := range srsFiles {
		mirror, err := downloadFile(srsDir, filename, rawURL, proxy)
		r := Result{File: filename, Mirror: mirror}
		if err != nil {
			r.Error = err.Error()
		}
		results = append(results, r)
	}
	return results
}

func downloadFile(srsDir, filename, rawURL, customProxy string) (string, error) {
	type candidate struct {
		label string
		url   string
	}
	candidates := []candidate{{"direct", rawURL}}

	// Custom proxy first (user-specified takes priority over builtins)
	if customProxy != "" {
		p := customProxy
		if p[len(p)-1] != '/' {
			p += "/"
		}
		candidates = append(candidates, candidate{p, p + rawURL})
	}
	for _, m := range BuiltinMirrors {
		candidates = append(candidates, candidate{m, m + rawURL})
	}

	var lastErr error
	for _, c := range candidates {
		if err := fetchToFile(c.url, filepath.Join(srsDir, filename)); err != nil {
			lastErr = fmt.Errorf("%s: %w", c.label, err)
			continue
		}
		return c.label, nil
	}
	return "", lastErr
}

func fetchToFile(url, dst string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()
	return os.Rename(tmp, dst)
}
