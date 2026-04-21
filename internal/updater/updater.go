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
	"geosite-private.srs":         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geosite/private.srs",
	"geosite-gfw.srs":             "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geosite/gfw.srs",
	"geosite-geolocation-!cn.srs": "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geosite/geolocation-!cn.srs",
	"geoip-private.srs":           "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geoip/private.srs",
	"geoip-google.srs":            "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geoip/google.srs",
	"geoip-facebook.srs":          "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geoip/facebook.srs",
	"geoip-telegram.srs":          "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geoip/telegram.srs",
	"geoip-twitter.srs":           "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geoip/twitter.srs",
	"geoip-netflix.srs":           "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/refs/heads/sing/geo/geoip/netflix.srs",
	"ads.srs":                     "https://raw.githubusercontent.com/privacy-protection-tools/anti-ad.github.io/master/docs/anti-ad-sing-box.srs",
}

// mirrors are tried in order after a direct download fails.
// Each mirror is a URL prefix prepended to the raw GitHub URL.
var mirrors = []string{
	"https://ghfast.top/",
	"https://gh-proxy.com/",
	"https://gh.ddlc.top/",
	"https://ghproxy.it/",
}

// Result reports the outcome of a single file update.
type Result struct {
	File   string `json:"file"`
	Mirror string `json:"mirror"` // "direct" or the mirror prefix used
	Error  string `json:"error,omitempty"`
}

// UpdateAll downloads all rule set files into srsDir.
// Each file is tried directly first; on failure the mirrors are tried in order.
// Files are written atomically (temp file → rename).
func UpdateAll(srsDir string) []Result {
	results := make([]Result, 0, len(srsFiles))
	for filename, rawURL := range srsFiles {
		mirror, err := downloadFile(srsDir, filename, rawURL)
		r := Result{File: filename, Mirror: mirror}
		if err != nil {
			r.Error = err.Error()
		}
		results = append(results, r)
	}
	return results
}

// downloadFile tries direct then each mirror until one succeeds.
// Returns the mirror label used ("direct" or prefix) and any final error.
func downloadFile(srsDir, filename, rawURL string) (string, error) {
	candidates := []struct {
		label string
		url   string
	}{
		{"direct", rawURL},
	}
	for _, m := range mirrors {
		candidates = append(candidates, struct {
			label string
			url   string
		}{m, m + rawURL})
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

// fetchToFile downloads url into a temp file next to dst, then renames atomically.
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
