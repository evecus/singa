package singbox

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const installPath = "/usr/bin/sing-box"

// Flavor selects which build to download.
type Flavor string

const (
	FlavorOfficial Flavor = "official" // SagerNet/sing-box
	FlavorReF1nd   Flavor = "ref1nd"   // reF1nd/sing-box-releases
)

// Version returns the installed sing-box version string, or "" if not installed.
func Version() string {
	out, err := exec.Command(installPath, "version").Output()
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return strings.TrimSpace(string(out))
}

// SystemInfo holds detected OS/arch info.
type SystemInfo struct {
	Arch   string `json:"arch"`
	LibC   string `json:"libc"`
	OSName string `json:"osName"`
}

// DetectSystem detects the current system's arch and libc type.
func DetectSystem() SystemInfo {
	info := SystemInfo{
		Arch:   runtime.GOARCH,
		LibC:   "glibc",
		OSName: detectOSName(),
	}
	if _, err := os.Stat("/etc/alpine-release"); err == nil {
		info.LibC = "musl"
		return info
	}
	out, err := exec.Command("ldd", "--version").CombinedOutput()
	if err == nil && strings.Contains(strings.ToLower(string(out)), "musl") {
		info.LibC = "musl"
		return info
	}
	matches, _ := filepath.Glob("/lib/ld-musl*")
	if len(matches) > 0 {
		info.LibC = "musl"
	}
	return info
}

func detectOSName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "linux"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
	}
	return "linux"
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func latestRelease(repo string) (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch release info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch release info: HTTP %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("parse release info: %w", err)
	}
	return &rel, nil
}

// archKeywords returns the arch keyword candidates to look for in an asset filename.
// Ordered from most specific to least specific so the first match wins.
func archKeywords(arch string) []string {
	switch arch {
	case "arm64", "aarch64":
		return []string{"arm64", "aarch64"}
	case "arm":
		return []string{"armv7", "armv6", "arm"}
	case "amd64":
		return []string{"amd64", "x86_64"}
	case "386":
		return []string{"386", "i386", "x86"}
	case "mips64le":
		return []string{"mips64le"}
	case "mips64":
		return []string{"mips64"}
	case "mipsle":
		return []string{"mipsle", "mipsel"}
	case "mips":
		return []string{"mips"}
	default:
		return []string{arch}
	}
}

// pickAsset selects the best matching asset from a release for the given arch/libc.
//
// Strategy (no hardcoded filenames):
//  1. Keep only assets that are downloadable archives (.tar.gz / .zip) and
//     contain both "linux" and an arch keyword in the name.
//  2. Three-tier priority by libc:
//     glibc: prefer "glibc" tag → generic (no libc tag) → musl/purego static
//     musl:  prefer "musl"  tag → "purego" static        → generic/glibc dynamic
//  3. First asset in the winning tier is returned.
//
// This way the code never needs updating when the upstream author renames files.
func pickAsset(assets []struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}, arch, libc string) (name, url string) {

	archWords := archKeywords(arch)
	isMusl := libc == "musl"

	isArchive := func(n string) bool {
		n = strings.ToLower(n)
		return strings.HasSuffix(n, ".tar.gz") || strings.HasSuffix(n, ".zip")
	}
	containsArch := func(n string) bool {
		lower := strings.ToLower(n)
		for _, kw := range archWords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
		return false
	}
	isLinux := func(n string) bool {
		return strings.Contains(strings.ToLower(n), "linux")
	}
	type candidate struct{ name, url string }
	var candidates []candidate
	for _, a := range assets {
		if !isArchive(a.Name) || !isLinux(a.Name) || !containsArch(a.Name) {
			continue
		}
		candidates = append(candidates, candidate{a.Name, a.BrowserDownloadURL})
	}
	if len(candidates) == 0 {
		return "", ""
	}

	hasKeyword := func(n, kw string) bool {
		return strings.Contains(strings.ToLower(n), kw)
	}

	// Three tiers, first non-empty wins.
	// glibc: "glibc" tag → generic (no libc tag) → musl/purego static
	// musl:  "musl"  tag → "purego" static        → generic/glibc dynamic
	var tier1, tier2, tier3 []candidate
	for _, c := range candidates {
		hasGlibc := hasKeyword(c.name, "glibc")
		hasMusl := hasKeyword(c.name, "musl")
		hasPurego := hasKeyword(c.name, "purego")
		isGeneric := !hasGlibc && !hasMusl && !hasPurego

		if isMusl {
			switch {
			case hasMusl:
				tier1 = append(tier1, c)
			case hasPurego:
				tier2 = append(tier2, c)
			default: // generic or glibc
				tier3 = append(tier3, c)
			}
		} else {
			switch {
			case hasGlibc:
				tier1 = append(tier1, c)
			case isGeneric:
				tier2 = append(tier2, c)
			default: // musl/purego static
				tier3 = append(tier3, c)
			}
		}
	}
	for _, tier := range [][]candidate{tier1, tier2, tier3} {
		if len(tier) > 0 {
			return tier[0].name, tier[0].url
		}
	}
	return "", ""
}

// Install downloads and installs sing-box to /usr/bin/sing-box.
// flavor selects official or reF1nd build.
// proxy is an optional GitHub proxy URL prefix.
func Install(flavor Flavor, proxy string) (string, error) {
	sys := DetectSystem()

	var repo string
	switch flavor {
	case FlavorReF1nd:
		repo = "reF1nd/sing-box-releases"
	default:
		repo = "SagerNet/sing-box"
	}

	rel, err := latestRelease(repo)
	if err != nil {
		return "", err
	}

	chosenAsset, downloadURL := pickAsset(rel.Assets, sys.Arch, sys.LibC)
	if downloadURL == "" {
		var allNames []string
		for _, a := range rel.Assets {
			allNames = append(allNames, a.Name)
		}
		return "", fmt.Errorf("no asset found for linux/%s (%s) version %s in %s\navailable: %v",
			sys.Arch, sys.LibC, rel.TagName, repo, allNames)
	}

	// Apply proxy prefix
	if proxy != "" {
		p := strings.TrimRight(proxy, "/") + "/"
		downloadURL = p + downloadURL
	}

	// Download to temp file
	tmp, err := os.CreateTemp("", "sing-box-*.tar.gz")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		tmp.Close()
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		tmp.Close()
		return "", fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	// Extract
	binPath, err := extractBinary(tmpPath, chosenAsset)
	if err != nil {
		return "", fmt.Errorf("extract: %w", err)
	}
	defer os.Remove(binPath)

	// Set executable permission
	if err := os.Chmod(binPath, 0755); err != nil {
		return "", err
	}

	// Atomic install
	if err := os.Rename(binPath, installPath); err != nil {
		if err2 := copyExec(binPath, installPath); err2 != nil {
			return "", fmt.Errorf("install: %w", err2)
		}
	}
	// Ensure executable bit after copy
	_ = os.Chmod(installPath, 0755)

	return rel.TagName, nil
}

func extractBinary(archivePath, assetName string) (string, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(archivePath)
	}
	return extractFromTarGz(archivePath)
}

func extractFromTarGz(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(hdr.Name) == "sing-box" && hdr.Typeflag == tar.TypeReg {
			tmp, err := os.CreateTemp("", "sing-box-bin-*")
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(tmp, tr); err != nil {
				tmp.Close()
				os.Remove(tmp.Name())
				return "", err
			}
			tmp.Close()
			return tmp.Name(), nil
		}
	}
	return "", fmt.Errorf("sing-box binary not found in archive")
}

func extractFromZip(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) == "sing-box" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			tmp, err := os.CreateTemp("", "sing-box-bin-*")
			if err != nil {
				rc.Close()
				return "", err
			}
			_, copyErr := io.Copy(tmp, rc)
			rc.Close()
			tmp.Close()
			if copyErr != nil {
				os.Remove(tmp.Name())
				return "", copyErr
			}
			return tmp.Name(), nil
		}
	}
	return "", fmt.Errorf("sing-box binary not found in zip")
}

func copyExec(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}
