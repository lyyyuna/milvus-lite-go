package milvuslite

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	defaultPyPISimpleURL = "https://pypi.org/simple/milvus-lite/"
	Version              = "2.5.1"
)

// platformTag returns the wheel platform tag for the current OS/arch.
func platformTag() (string, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		return "macosx_11_0_arm64", nil
	case "darwin/amd64":
		return "macosx_10_9_x86_64", nil
	case "linux/amd64":
		return "manylinux2014_x86_64", nil
	case "linux/arm64":
		return "manylinux2014_aarch64", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

// wheelFileName returns the expected wheel filename for the given version and platform.
func wheelFileName(version, platform string) string {
	return fmt.Sprintf("milvus_lite-%s-py3-none-%s.whl", version, platform)
}

// cacheDir returns the cache directory for the given version and platform.
// Layout: ~/.cache/milvus-lite/{version}/{os}-{arch}/
func cacheDir(version string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".cache", "milvus-lite", version, runtime.GOOS+"-"+runtime.GOARCH), nil
}

// libDir returns the path to the lib directory containing the milvus binary.
// Returns empty string if not yet downloaded.
func libDir(version string) (string, error) {
	dir, err := cacheDir(version)
	if err != nil {
		return "", err
	}
	lib := filepath.Join(dir, "lib")
	bin := filepath.Join(lib, "milvus")
	if _, err := os.Stat(bin); err != nil {
		return "", nil
	}
	return lib, nil
}

// ensureBinary downloads the milvus-lite binary if not cached, returns the lib directory path.
func ensureBinary(version string) (string, error) {
	lib, err := libDir(version)
	if err != nil {
		return "", err
	}
	if lib != "" {
		return lib, nil
	}

	dir, err := cacheDir(version)
	if err != nil {
		return "", err
	}

	plat, err := platformTag()
	if err != nil {
		return "", err
	}

	whlName := wheelFileName(version, plat)
	baseURL := pypiSimpleURL()

	downloadURL, expectedHash, err := resolveWheelURL(baseURL, whlName)
	if err != nil {
		return "", fmt.Errorf("resolve wheel URL: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	whlPath := filepath.Join(dir, whlName)
	if err := downloadFile(downloadURL, whlPath, expectedHash); err != nil {
		return "", fmt.Errorf("download wheel: %w", err)
	}

	lib = filepath.Join(dir, "lib")
	if err := extractLib(whlPath, lib); err != nil {
		os.RemoveAll(lib)
		return "", fmt.Errorf("extract wheel: %w", err)
	}

	// make milvus binary executable
	if err := os.Chmod(filepath.Join(lib, "milvus"), 0o755); err != nil {
		return "", fmt.Errorf("chmod milvus: %w", err)
	}

	os.Remove(whlPath)
	return lib, nil
}

// pypiSimpleURL reads pip config to find mirror URL, falls back to official PyPI.
func pypiSimpleURL() string {
	for _, path := range pipConfigPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if url := parsePipIndexURL(string(data)); url != "" {
			// pip config uses /simple/pkgname/ format
			url = strings.TrimRight(url, "/")
			return url + "/milvus-lite/"
		}
	}
	return defaultPyPISimpleURL
}

// pipConfigPaths returns possible pip config file locations.
func pipConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	paths := []string{
		filepath.Join(home, ".pip", "pip.conf"),
		filepath.Join(home, ".config", "pip", "pip.conf"),
	}
	if runtime.GOOS == "windows" {
		appdata := os.Getenv("APPDATA")
		if appdata != "" {
			paths = append(paths, filepath.Join(appdata, "pip", "pip.ini"))
		}
	}
	return paths
}

// parsePipIndexURL extracts index-url from pip config content.
// Looks for: index-url = https://...
func parsePipIndexURL(content string) string {
	re := regexp.MustCompile(`(?m)^\s*index-url\s*=\s*(.+)$`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

// resolveWheelURL fetches the Simple API HTML page and finds the download URL for the target wheel.
// Returns the absolute download URL and sha256 hash.
func resolveWheelURL(simpleURL, targetFilename string) (downloadURL, sha256Hash string, err error) {
	resp, err := http.Get(simpleURL)
	if err != nil {
		return "", "", fmt.Errorf("fetch simple index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("simple index returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read simple index: %w", err)
	}

	// Parse HTML: <a href="../../packages/...whl#sha256=abc123">filename</a>
	// or absolute URLs from mirrors
	linkRe := regexp.MustCompile(`<a\s+href="([^"]+)"[^>]*>\s*` + regexp.QuoteMeta(targetFilename) + `\s*</a>`)
	matches := linkRe.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", "", fmt.Errorf("wheel %s not found in index", targetFilename)
	}

	href := matches[1]

	// Extract sha256 from fragment: ...#sha256=abc123
	parts := strings.SplitN(href, "#sha256=", 2)
	if len(parts) == 2 {
		sha256Hash = parts[1]
		href = parts[0]
	}

	// Resolve relative URL
	if strings.HasPrefix(href, "../../") || !strings.HasPrefix(href, "http") {
		// Relative to simple index base
		base := strings.TrimRight(simpleURL, "/")
		// Go up directory levels
		for strings.HasPrefix(href, "../") {
			href = strings.TrimPrefix(href, "../")
			base = base[:strings.LastIndex(base, "/")]
		}
		href = base + "/" + href
	}

	return href, sha256Hash, nil
}

// downloadFile downloads a URL to a local file, verifying sha256 if provided.
func downloadFile(url, dest, expectedSHA256 string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(f, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if expectedSHA256 != "" {
		actual := hex.EncodeToString(hasher.Sum(nil))
		if actual != expectedSHA256 {
			os.Remove(dest)
			return fmt.Errorf("sha256 mismatch: expected %s, got %s", expectedSHA256, actual)
		}
	}

	return nil
}

// extractLib extracts milvus_lite/lib/* from a wheel (zip) file into destDir.
func extractLib(whlPath, destDir string) error {
	r, err := zip.OpenReader(whlPath)
	if err != nil {
		return fmt.Errorf("open wheel: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	prefix := "milvus_lite/lib/"
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}

		relPath := strings.TrimPrefix(f.Name, prefix)
		if relPath == "" {
			continue
		}

		destPath := filepath.Join(destDir, relPath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0o755)
			continue
		}

		if err := extractZipFile(f, destPath); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
