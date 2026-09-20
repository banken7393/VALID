package gate

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/banken7393/valid/internal/schema"
)

// Options controls gate behaviour beyond board inspection.
type Options struct {
	// TrustBoard skips executing test_command and only inspects board tdd evidence.
	TrustBoard bool
}

// Result is the outcome of a gate run.
type Result struct {
	Passed   bool
	Findings []schema.Finding
}

// Run evaluates hard gates and soft isolation warnings on the board.
// By default it executes cfg.TestCommand in the feature worktree (or repo root
// for minipatch) and records the outcome on the board before checking green.
func Run(repoRoot string, b *schema.Board, cfg schema.Config) Result {
	return RunWithOptions(repoRoot, b, cfg, Options{})
}

// RunWithOptions is Run with explicit options.
func RunWithOptions(repoRoot string, b *schema.Board, cfg schema.Config, opts Options) Result {
	var findings []schema.Finding

	if b == nil {
		return Result{Passed: false, Findings: []schema.Finding{{
			Code: "board_nil", Severity: schema.SevError, Message: "board is nil",
		}}}
	}

	if !opts.TrustBoard {
		if f := executeTestCommand(repoRoot, b, cfg); f != nil {
			findings = append(findings, *f)
		}
	}

	// Soft isolation warning.
	if b.Environment.IsolationWarning || (b.Mode != schema.ModeMinipatch && b.Environment.WorktreePath == "") {
		findings = append(findings, schema.Finding{
			Code:     "isolation_warning",
			Severity: schema.SevWarning,
			Message:  "working outside feature Dev Environment (soft isolation)",
		})
	}
	if b.Mode != schema.ModeMinipatch && b.Environment.WorktreePath != "" {
		if _, err := os.Stat(b.Environment.WorktreePath); err != nil {
			findings = append(findings, schema.Finding{
				Code:     "isolation_warning",
				Severity: schema.SevWarning,
				Message:  "feature worktree path missing: " + b.Environment.WorktreePath,
			})
		}
	}

	// How-it-works required past plan for feature/patch (real Mermaid, not scaffold stub).
	if b.Mode != schema.ModeMinipatch && b.Lifecycle != schema.LifePlan {
		how := strings.TrimSpace(b.HowItWorks)
		if how == "" && b.Paths.HowItWorks != "" {
			if abs, err := schema.ResolveUnderRoot(repoRoot, b.Paths.HowItWorks); err == nil {
				if raw, err := os.ReadFile(abs); err == nil {
					how = strings.TrimSpace(string(raw))
				}
			}
		}
		switch {
		case how == "" || isHowItWorksCommentOnly(how):
			findings = append(findings, schema.Finding{
				Code: "how_it_works_missing", Severity: schema.SevError,
				Message: "how-it-works Mermaid is required (author during skill plan)",
			})
		case isHowItWorksStub(how):
			findings = append(findings, schema.Finding{
				Code: "how_it_works_stub", Severity: schema.SevError,
				Message: "how-it-works looks like a scaffold stub; replace with a real Mermaid diagram",
			})
		case !looksLikeMermaid(how):
			findings = append(findings, schema.Finding{
				Code: "how_it_works_invalid", Severity: schema.SevError,
				Message: "how-it-works must be Mermaid (flowchart / sequenceDiagram / …)",
			})
		}
	}

	// AC coverage (hard).
	if len(b.What) == 0 {
		findings = append(findings, schema.Finding{
			Code: "no_acceptance_criteria", Severity: schema.SevError,
			Message: "board has no acceptance criteria (what[])",
		})
	} else {
		for _, id := range b.UncoveredACs() {
			findings = append(findings, schema.Finding{
				Code: "ac_uncovered", Severity: schema.SevError,
				Message: fmt.Sprintf("acceptance criterion %q has no task covering it", id),
			})
		}
	}

	// Executable tests (hard) — prefer fresh run evidence on the board.
	if b.TDD.Total == 0 {
		findings = append(findings, schema.Finding{
			Code: "no_tests", Severity: schema.SevError,
			Message: fmt.Sprintf("no executable tests reported (run %q and record results on the board)", cfg.TestCommand),
		})
	} else if !b.TestsGreen() {
		findings = append(findings, schema.Finding{
			Code: "tests_not_green", Severity: schema.SevError,
			Message: fmt.Sprintf("tests not green: passed=%d failed=%d total=%d", b.TDD.Passed, b.TDD.Failed, b.TDD.Total),
		})
	}

	// Soft: tasks that cover ACs should be done before finish-quality gates.
	if b.Lifecycle == schema.LifeAudit || b.Lifecycle == schema.LifeFinish || b.Lifecycle == schema.LifeDone {
		for _, t := range b.Tasks {
			if len(t.Covers) > 0 && t.Status != "done" {
				findings = append(findings, schema.Finding{
					Code: "task_not_done", Severity: schema.SevWarning,
					Message: fmt.Sprintf("task %q covers ACs but status=%s", t.ID, t.Status),
				})
			}
		}
	}

	if b.Environment.DatabaseEnabled {
		findings = append(findings, schema.Finding{
			Code: "database_isolation", Severity: schema.SevWarning,
			Message: "database.enabled=true — do not use the principal DSN; follow config.database.recipe / workspace notes",
		})
	}

	passed := true
	for _, f := range findings {
		if f.Severity == schema.SevError {
			passed = false
			break
		}
	}

	b.Audit = schema.AuditState{
		Passed:   passed,
		Findings: findings,
		At:       time.Now().UTC(),
	}
	if cfg.Isolation.Strict && hasWarning(findings, "isolation_warning") {
		passed = false
		b.Audit.Passed = false
		findings = append(findings, schema.Finding{
			Code: "isolation_strict", Severity: schema.SevError,
			Message: "isolation.strict=true and isolation warning present",
		})
		b.Audit.Findings = findings
	}

	return Result{Passed: passed, Findings: findings}
}

// executeTestCommand runs cfg.TestCommand and updates board TDD aggregates.
// Returns an error finding when the command cannot be started; nil when run completed
// (pass or fail is reflected on the board for TestsGreen checks).
func executeTestCommand(repoRoot string, b *schema.Board, cfg schema.Config) *schema.Finding {
	cmdLine := strings.TrimSpace(cfg.TestCommand)
	if cmdLine == "" {
		return &schema.Finding{
			Code: "test_command_missing", Severity: schema.SevError,
			Message: "config.test_command is empty",
		}
	}
	cwd := repoRoot
	if b.Mode != schema.ModeMinipatch && strings.TrimSpace(b.Environment.WorktreePath) != "" {
		if st, err := os.Stat(b.Environment.WorktreePath); err == nil && st.IsDir() {
			cwd = b.Environment.WorktreePath
		}
	}
	ctxCwd := cwd
	// shell-form so project commands like `go test ./...` / `npm test` work.
	cmd := exec.Command("bash", "-lc", cmdLine)
	cmd.Dir = ctxCwd
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := truncateRunOutput(buf.String(), 32*1024)
	name := "valid-gate:" + filepath.Base(ctxCwd)
	status := schema.TestPass
	if err != nil {
		status = schema.TestFail
	}
	_ = b.ApplyReport(status, name, out, "")
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil // failed tests → board evidence; tests_not_green will fire
		}
		return &schema.Finding{
			Code: "test_command_error", Severity: schema.SevError,
			Message: fmt.Sprintf("failed to run %q in %s: %v", cmdLine, ctxCwd, err),
		}
	}
	return nil
}

func truncateRunOutput(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n…(truncated)"
}

func hasWarning(findings []schema.Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func isHowItWorksCommentOnly(how string) bool {
	for _, line := range strings.Split(how, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "%%") {
			continue
		}
		return false
	}
	return true
}

func isHowItWorksStub(how string) bool {
	compact := strings.Join(strings.Fields(how), " ")
	if strings.Contains(compact, "start[Start] --> goal[") {
		return true
	}
	if utf8.RuneCountInString(compact) < 48 {
		return true
	}
	return false
}

func looksLikeMermaid(how string) bool {
	lower := strings.ToLower(how)
	markers := []string{
		"flowchart", "graph ", "sequencediagram", "classdiagram",
		"statediagram", "erdiagram", "journey", "gantt", "pie",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}
