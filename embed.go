package milvuslite

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

const Version = "2.5.1"

var (
	extractOnce sync.Once
	extractedDir string
	extractErr   error
)

// platformLib is set by the platform-specific build tag file.
var platformLib embed.FS

// libDir extracts the embedded binary to a temporary directory (once)
// and returns the path to the lib directory.
func libDir() (string, error) {
	extractOnce.Do(func() {
		extractedDir, extractErr = extractEmbeddedLib()
	})
	return extractedDir, extractErr
}

func extractEmbeddedLib() (string, error) {
	dir, err := os.MkdirTemp("", "milvus-lite-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	err = fs.WalkDir(platformLib, "lib", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(dir, path)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		data, err := platformLib.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}

		if err := os.WriteFile(destPath, data, 0o755); err != nil {
			return fmt.Errorf("write %s: %w", destPath, err)
		}

		return nil
	})

	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("extract embedded lib: %w", err)
	}

	return filepath.Join(dir, "lib"), nil
}
