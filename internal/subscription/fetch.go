package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/singa/internal/node"
	"github.com/singa/internal/builder"
)

// Fetch downloads a subscription URL and returns the parsed outbound list.
// Supports:
//   - sing-box JSON ({"outbounds":[...]})
//   - raw base64-encoded share links (one per line after decode)
//   - plain share links (one per line, no base64)
func Fetch(url string) ([]map[string]any, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("fetch: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // 8 MB cap
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return parse(string(body))
}

// parse auto-detects format and returns a list of sing-box outbound objects.
func parse(body string) ([]map[string]any, error) {
	body = strings.TrimSpace(body)

	// 1. sing-box JSON {"outbounds":[...]}
	if strings.HasPrefix(body, "{") {
		var cfg struct {
			Outbounds []map[string]any `json:"outbounds"`
		}
		if err := json.Unmarshal([]byte(body), &cfg); err == nil && len(cfg.Outbounds) > 0 {
			return filterProxies(cfg.Outbounds), nil
		}
	}

	// 2. JSON array of outbounds directly
	if strings.HasPrefix(body, "[") {
		var arr []map[string]any
		if err := json.Unmarshal([]byte(body), &arr); err == nil && len(arr) > 0 {
			return filterProxies(arr), nil
		}
	}

	// 3. Base64-encoded share links
	if decoded, err := tryBase64(body); err == nil {
		return parseLinks(decoded)
	}

	// 4. Raw share links (one per line)
	return parseLinks(body)
}

// tryBase64 attempts to decode body as standard or URL-safe base64.
func tryBase64(s string) (string, error) {
	// Pad if needed
	pad := (4 - len(s)%4) % 4
	s = s + strings.Repeat("=", pad)
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		b, err = base64.URLEncoding.DecodeString(s)
	}
	if err != nil {
		return "", err
	}
	decoded := string(b)
	// Must look like share links or JSON to be valid
	if !strings.Contains(decoded, "://") && !strings.HasPrefix(strings.TrimSpace(decoded), "{") {
		return "", fmt.Errorf("not a share-link base64")
	}
	return decoded, nil
}

// parseLinks converts newline-separated share links into sing-box outbound objects.
func parseLinks(text string) ([]map[string]any, error) {
	nodes, errs := node.ParseLinks(text)
	if len(nodes) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("no valid nodes: %s", strings.Join(errs, "; "))
	}
	var out []map[string]any
	for _, n := range nodes {
		ob, err := builder.NodeToOutbound(n, n.Name)
		if err != nil {
			continue
		}
		raw := make(map[string]any)
		// round-trip through JSON to get map[string]any
		data, _ := json.Marshal(ob)
		_ = json.Unmarshal(data, &raw)
		out = append(out, raw)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no supported nodes found")
	}
	return out, nil
}

// filterProxies keeps only actual proxy outbounds (not selector/urltest/direct/block/dns).
var nonProxyTypes = map[string]bool{
	"direct": true, "block": true, "dns": true,
	"selector": true, "urltest": true, "loadbalance": true,
}

func filterProxies(obs []map[string]any) []map[string]any {
	var out []map[string]any
	for _, ob := range obs {
		t, _ := ob["type"].(string)
		if t == "" || nonProxyTypes[t] {
			continue
		}
		out = append(out, ob)
	}
	return out
}
