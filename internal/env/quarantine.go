package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// QuarantineDockerfile is the VALID feature isolation Dockerfile template.
// Language-agnostic: change BASE_IMAGE (go, node, python, …); keep UID 1000 + sudo + git.
const QuarantineDockerfile = `# VALID feature quarantine — isolate deps from the host.
# Override BASE_IMAGE for the consumer stack (do not hardcode a language in VALID).
ARG BASE_IMAGE=debian:bookworm
FROM ${BASE_IMAGE}

RUN apt-get update && apt-get install -y --no-install-recommends \
        git \
        openssh-client \
        sudo \
        ca-certificates \
        curl \
    && rm -rf /var/lib/apt/lists/*

# Non-root user with UID 1000 (matches typical host / Dev Containers vscode user)
RUN id -u 1000 >/dev/null 2>&1 || useradd -m -u 1000 -s /bin/bash vscode \
    && mkdir -p /home/vscode \
    && chown -R 1000:1000 /home/vscode \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

USER vscode
WORKDIR /workspaces
`

// QuarantineDevcontainerJSON is the feature worktree DC (build + DooD). Placeholders:
// {{NAME}} {{DASHBOARD_PORT}}
const QuarantineDevcontainerJSON = `{
  "name": "{{NAME}}",
  "build": {
    "dockerfile": "Dockerfile",
    "args": {
      "BASE_IMAGE": "{{BASE_IMAGE}}"
    }
  },
  "features": {
    "ghcr.io/devcontainers/features/docker-outside-of-docker:1": {
      "moby": true,
      "dockerDashComposeVersion": "v2"
    }
  },
  "remoteUser": "vscode",
  "runArgs": ["--name", "{{NAME}}"],
  "forwardPorts": [
    {{DASHBOARD_PORT}}
  ],
  "portsAttributes": {
    "{{DASHBOARD_PORT}}": {
      "label": "VALID dashboard",
      "onAutoForward": "notify"
    }
  },
  "customizations": {
    "vscode": {
      "extensions": [],
      "settings": {
        "files.eol": "\n"
      }
    }
  }
}
`

// DetectBaseImage picks a sensible BASE_IMAGE from repo signals (still not VALID-hardcoded to one language).
func DetectBaseImage(repoRoot string) string {
	checks := []struct {
		file  string
		image string
	}{
		{"go.mod", "golang:1.22-bookworm"},
		{"package.json", "node:22-bookworm"},
		{"pyproject.toml", "python:3.12-bookworm"},
		{"requirements.txt", "python:3.12-bookworm"},
		{"Cargo.toml", "rust:1-bookworm"},
		{"pom.xml", "maven:3-eclipse-temurin-21"},
		{"build.gradle", "gradle:8-jdk21"},
		{"Gemfile", "ruby:3.3-bookworm"},
		{"composer.json", "php:8.3-bookworm"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(repoRoot, c.file)); err == nil {
			return c.image
		}
	}
	return "debian:bookworm"
}

// WriteQuarantineDevcontainer writes Dockerfile + devcontainer.json into worktree/.devcontainer/.
func WriteQuarantineDevcontainer(worktreePath, feature string, dashboardPort int, baseImage string) (string, error) {
	if dashboardPort <= 0 {
		dashboardPort = 7532
	}
	if baseImage == "" {
		baseImage = "debian:bookworm"
	}
	dir := filepath.Join(worktreePath, ".devcontainer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create .devcontainer: %w", err)
	}

	dfPath := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(dfPath, []byte(QuarantineDockerfile), 0o644); err != nil {
		return "", fmt.Errorf("write Dockerfile: %w", err)
	}

	name := "valid-" + feature
	body := QuarantineDevcontainerJSON
	body = strings.ReplaceAll(body, "{{NAME}}", name)
	body = strings.ReplaceAll(body, "{{BASE_IMAGE}}", baseImage)
	body = strings.ReplaceAll(body, "{{DASHBOARD_PORT}}", fmt.Sprintf("%d", dashboardPort))

	jsonPath := filepath.Join(dir, "devcontainer.json")
	if err := os.WriteFile(jsonPath, []byte(body), 0o644); err != nil {
		return "", fmt.Errorf("write devcontainer.json: %w", err)
	}
	return jsonPath, nil
}

// CopyMainDevcontainerIntoWorktree clones the project principal DC (json + sibling Dockerfile if any)
// into the feature worktree only. It never writes the principal path.
//
// Host port collision avoidance: principal forwardPorts / portsAttributes are NOT copied.
// The feature DC forwards only featureDashboardPort (from config), under a unique container name.
func CopyMainDevcontainerIntoWorktree(worktreePath, feature, mainJSONPath string, featureDashboardPort int) (string, error) {
	if featureDashboardPort <= 0 {
		featureDashboardPort = 7532
	}
	dir := filepath.Join(worktreePath, ".devcontainer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	m, err := LoadDevcontainerJSON(mainJSONPath)
	if err != nil {
		return "", fmt.Errorf("clone main DC: %w", err)
	}

	containerName := "valid-" + feature
	m["name"] = containerName
	// Do not steal principal host ports (app servers, DB, principal dashboard/MCP).
	m["forwardPorts"] = []interface{}{featureDashboardPort}
	m["portsAttributes"] = map[string]interface{}{
		fmt.Sprintf("%d", featureDashboardPort): map[string]interface{}{
			"label":         "VALID feature dashboard",
			"onAutoForward": "notify",
		},
	}
	// Unique docker container name when the CLI supports runArgs.
	m["runArgs"] = mergeRunArgsName(m["runArgs"], containerName)

	dstJSON := filepath.Join(dir, "devcontainer.json")
	if err := WriteDevcontainerJSON(dstJSON, m); err != nil {
		return "", err
	}

	// Copy sibling Dockerfile / compose files from main .devcontainer when present.
	mainDir := filepath.Dir(mainJSONPath)
	entries, _ := os.ReadDir(mainDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if lower == "devcontainer.json" {
			continue
		}
		if strings.HasPrefix(lower, "dockerfile") || lower == "compose.yaml" || lower == "compose.yml" || lower == "docker-compose.yml" {
			src := filepath.Join(mainDir, name)
			dst := filepath.Join(dir, name)
			raw, err := os.ReadFile(src)
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(dst, raw, 0o644); err != nil {
				return "", err
			}
		}
	}
	return dstJSON, nil
}

func mergeRunArgsName(existing interface{}, containerName string) []interface{} {
	out := []interface{}{}
	seenName := false
	if arr, ok := existing.([]interface{}); ok {
		for i := 0; i < len(arr); i++ {
			s, _ := arr[i].(string)
			if s == "--name" && i+1 < len(arr) {
				out = append(out, "--name", containerName)
				i++
				seenName = true
				continue
			}
			out = append(out, arr[i])
		}
	}
	if !seenName {
		out = append(out, "--name", containerName)
	}
	return out
}
