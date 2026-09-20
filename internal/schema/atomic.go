package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AtomicWrite writes data to path via a unique temporary file and os.Rename.
// Unique temps avoid concurrent writers clobbering a shared path.tmp sibling.
func AtomicWrite(path string, data []byte) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("atomic rename: %w", err)
	}
	return nil
}

// ResolveUnderRoot joins repoRoot with a relative path and ensures the result
// stays under repoRoot (rejects absolute paths and .. traversal).
func ResolveUnderRoot(repoRoot, rel string) (string, error) {
	if repoRoot == "" {
		return "", fmt.Errorf("repo root is required")
	}
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("empty relative path")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute paths are not allowed: %s", rel)
	}
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes repo root: %s", rel)
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(absRoot, clean)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	sep := string(os.PathSeparator)
	if absJoined != absRoot && !strings.HasPrefix(absJoined, absRoot+sep) {
		return "", fmt.Errorf("path escapes repo root: %s", rel)
	}
	return absJoined, nil
}
