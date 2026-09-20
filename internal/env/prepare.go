package env

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PrepareWorktree ensures the worktree has a minimal .valid/config copy (boards live in main repo).
func (m *Manager) PrepareWorktree(feature, worktreePath string) error {
	_ = feature
	dstValid := filepath.Join(worktreePath, ValidDir)
	if err := os.MkdirAll(dstValid, 0o755); err != nil {
		return fmt.Errorf("create worktree .valid: %w", err)
	}
	srcCfg := filepath.Join(m.RepoRoot, ValidDir, "config.json")
	if _, err := os.Stat(srcCfg); err == nil {
		if err := copyFile(srcCfg, filepath.Join(dstValid, "config.json")); err != nil {
			return fmt.Errorf("copy config.json: %w", err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
