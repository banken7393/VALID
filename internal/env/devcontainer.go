package env

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultMainDevcontainer is a helper for tests only — VALID never writes the principal DC.
func DefaultMainDevcontainer(projectName string, dashboardPort int) map[string]interface{} {
	if projectName == "" {
		projectName = "app"
	}
	if dashboardPort <= 0 {
		dashboardPort = 7432
	}
	return map[string]interface{}{
		"name":       projectName,
		"image":      "mcr.microsoft.com/devcontainers/base:bookworm",
		"features":   map[string]interface{}{},
		"remoteUser": "vscode",
		"forwardPorts": []int{
			dashboardPort,
		},
		"customizations": map[string]interface{}{
			"vscode": map[string]interface{}{
				"extensions": []string{},
			},
		},
	}
}

// ResolveMainDevcontainer returns the absolute path to the project principal DC if it exists.
// VALID never creates the principal — only feature worktree DCs are generated for isolation.
func ResolveMainDevcontainer(repoRoot, relPath string) (string, bool) {
	if relPath == "" {
		relPath = ".devcontainer/devcontainer.json"
	}
	absPath := relPath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(repoRoot, relPath)
	}
	if _, err := os.Stat(absPath); err == nil {
		return absPath, true
	}
	return absPath, false
}

// LoadDevcontainerJSON reads a devcontainer.json into a generic map (comments not supported).
func LoadDevcontainerJSON(path string) (map[string]interface{}, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Strip // line comments for slightly more tolerant parsing of hand-edited files.
	cleaned := stripJSONLineComments(string(raw))
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	return m, nil
}

func stripJSONLineComments(s string) string {
	// Only drop full-line // comments. Never strip inline // — that corrupts
	// URLs like https://… inside JSON string values.
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "//") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// WriteDevcontainerJSON writes a devcontainer map via temp file + rename.
func WriteDevcontainerJSON(path string, m map[string]interface{}) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicWriteFile(path, raw)
}

func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// InjectDevcontainer writes an ephemeral DC into the worktree only (never the principal .devcontainer/).
// Prefer cloning the project principal DC (+ Dockerfile siblings) then remapping ports so host
// forwards do not collide with the principal. Otherwise write VALID quarantine.
func InjectDevcontainer(worktreePath, feature string, featureDashboardPort int, mainPath, repoRoot string) (string, error) {
	if featureDashboardPort <= 0 {
		featureDashboardPort = 7532
	}
	if mainPath != "" {
		if _, err := os.Stat(mainPath); err == nil {
			return CopyMainDevcontainerIntoWorktree(worktreePath, feature, mainPath, featureDashboardPort)
		}
	}
	base := DetectBaseImage(repoRoot)
	return WriteQuarantineDevcontainer(worktreePath, feature, featureDashboardPort, base)
}

// DevcontainerAvailable reports whether the Dev Containers CLI is on PATH.
func DevcontainerAvailable() bool {
	_, err := exec.LookPath("devcontainer")
	return err == nil
}

// Up starts the feature worktree Dev Container when the CLI is available.
func Up(worktreePath string) error {
	if !DevcontainerAvailable() {
		return fmt.Errorf("devcontainer CLI not found on PATH; install @devcontainers/cli or open the worktree in an editor with Dev Containers support")
	}
	cmd := exec.Command("devcontainer", "up", "--workspace-folder", worktreePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("devcontainer up: %w", err)
	}
	return nil
}

// Down stops the feature Dev Container when possible and does not remove the worktree.
// Prefers docker rm -f on the unique container name valid-<feature>; falls back to a soft no-op.
func Down(worktreePath, feature string) error {
	_ = worktreePath
	name := "valid-" + feature
	if feature == "" {
		return fmt.Errorf("feature slug required to tear down Dev Container")
	}
	if _, err := exec.LookPath("docker"); err == nil {
		cmd := exec.Command("docker", "rm", "-f", name)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Container may already be gone — soft.
			if !strings.Contains(strings.ToLower(string(out)), "no such container") &&
				!strings.Contains(strings.ToLower(string(out)), "not found") {
				return fmt.Errorf("docker rm -f %s: %v\n%s", name, err, strings.TrimSpace(string(out)))
			}
		}
		return nil
	}
	if DevcontainerAvailable() {
		// Best-effort: older CLIs have no reliable "down"; try stopping via dockerless path.
		return fmt.Errorf("docker CLI not found; cannot tear down container %q (install docker or stop it manually)", name)
	}
	return fmt.Errorf("neither docker nor devcontainer CLI available to tear down %q", name)
}
