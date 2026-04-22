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

// officialAssetNames returns candidate asset names for the official SagerNet build.
// Format: sing-box-{ver}-linux-{arch}.tar.gz
// For musl systems, try -musl suffix first then fall back to standard.
func officialAssetNames(version, arch, libc string) []string {
	ver := strings.TrimPrefix(version, "v")
	goArch := arch
	if arch == "arm" {
		goArch = "armv7"
	}
	base := fmt.Sprintf("sing-box-%s-linux-%s", ver, goArch)
	if libc == "musl" {
		return []string{base + "-musl.tar.gz", base + ".tar.gz"}
	}
	return []string{base + ".tar.gz"}
}

// ref1ndAssetNames returns candidate asset names for reF1nd build.
// Format: sing-box-{ver}-reF1nd-linux-{arch}.tar.gz
// For musl/OpenWrt systems, try -purego suffix (static/CGO-free build).
func ref1ndAssetNames(version, arch, libc string) []string {
	ver := strings.TrimPrefix(version, "v")
	goArch := arch
	switch arch {
	case "arm":
		goArch = "armv7"
	case "arm64":
		goArch = "arm64"
	}
	base := fmt.Sprintf("sing-box-%s-reF1nd-linux-%s", ver, goArch)
	if libc == "musl" {
		// purego = no CGO, works on musl/OpenWrt
		return []string{base + "-purego.tar.gz", base + ".tar.gz"}
	}
	return []string{base + ".tar.gz", base + "-purego.tar.gz"}
}

// Install downloads and installs sing-box to /usr/bin/sing-box.
// flavor selects official or reF1nd build.
// proxy is an optional GitHub proxy URL prefix.
func Install(flavor Flavor, proxy string) (string, error) {
	sys := DetectSystem()

	var repo string
	var assetNamesFn func(version, arch, libc string) []string

	switch flavor {
	case FlavorReF1nd:
		repo = "reF1nd/sing-box-releases"
		assetNamesFn = ref1ndAssetNames
	default:
		repo = "SagerNet/sing-box"
		assetNamesFn = officialAssetNames
	}

	rel, err := latestRelease(repo)
	if err != nil {
		return "", err
	}

	candidates := assetNamesFn(rel.TagName, sys.Arch, sys.LibC)

	var downloadURL, chosenAsset string
	for _, name := range candidates {
		for _, asset := range rel.Assets {
			if asset.Name == name {
				downloadURL = asset.BrowserDownloadURL
				chosenAsset = name
				break
			}
		}
		if downloadURL != "" {
			break
		}
	}
	if downloadURL == "" {
		return "", fmt.Errorf("no asset found for linux/%s (%s) version %s in %s\ntried: %v",
			sys.Arch, sys.LibC, rel.TagName, repo, candidates)
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
