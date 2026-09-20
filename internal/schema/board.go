// Package schema defines the VALID board contract (v2) and project config.
// The LLM (via skills) owns board content edits; the CLI scaffolds boards and
// provides optional helpers plus atomic gate/promote/merge plumbing.
package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// Version is the board schema version.
	Version = "2"

	// Lifecycle values (feature board).
	LifePlan   = "plan"
	LifeSpec   = "spec"
	LifeBuild  = "build"
	LifeAudit  = "audit"
	LifeFinish = "finish"
	LifeDone   = "done"

	// TDD phase values during build.
	TDDRed      = "red"
	TDDGreen    = "green"
	TDDRefactor = "refactor"

	// Mode values.
	ModeFeature   = "feature"
	ModePatch     = "patch"
	ModeMinipatch = "minipatch"

	// Autonomy / loop modes (human presence during build).
	AutonomyInTheLoop    = "in_the_loop"
	AutonomyAboveTheLoop = "above_the_loop"

	// Test statuses.
	TestPass    = "pass"
	TestFail    = "fail"
	TestPending = "pending"

	// Gate finding severities.
	SevError   = "error"
	SevWarning = "warning"
	SevInfo    = "info"

	DataFileName   = "data.json"
	ValidDir       = ".valid"
	ConfigFileName = "config.json"
	FeaturesDir    = "features"
	KnowledgeDir   = "knowledge"
	WorktreesDir   = "worktrees"
)

// AcceptanceCriterion is one WHAT / AC entry.
type AcceptanceCriterion struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Phase is a delivery phase on the board.
type Phase struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Order int    `json:"order"`
}

// Task is a work item that may cover one or more ACs.
type Task struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"` // pending|doing|done
	Covers      []string `json:"covers"`
	Description string   `json:"description,omitempty"`
}

// Decision records a product/tech choice.
type Decision struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

// Assumption records an explicit assumption.
type Assumption struct {
	ID     string `json:"id"`
	Detail string `json:"detail"`
}

// TestCase is one named test result from valid report.
type TestCase struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Output string `json:"output,omitempty"`
}

// TDDState holds executable test evidence.
type TDDState struct {
	Phase    string     `json:"phase,omitempty"`
	Passed   int        `json:"passed"`
	Failed   int        `json:"failed"`
	Total    int        `json:"total"`
	Output   string     `json:"output,omitempty"`
	Coverage *float64   `json:"coverage,omitempty"`
	Cases    []TestCase `json:"cases,omitempty"`
}

// Finding is one gate/audit finding.
type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// AuditState stores the last gate run.
type AuditState struct {
	Passed   bool      `json:"passed"`
	Findings []Finding `json:"findings"`
	At       time.Time `json:"at,omitempty"`
}

// EnvironmentState tracks isolation / DC status.
type EnvironmentState struct {
	WorktreePath      string `json:"worktree_path,omitempty"`
	Branch            string `json:"branch,omitempty"`      // feature/<slug>
	BaseBranch        string `json:"base_branch,omitempty"` // principal branch the worktree was cut from
	DevcontainerPath  string `json:"devcontainer_path,omitempty"`
	IsolationWarning  bool   `json:"isolation_warning"`
	DatabaseEnabled   bool   `json:"database_enabled"`
}

// Promotion is a pending env/deps change.
type Promotion struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"` // dc_feature|forward_port|extension|dockerfile_line|manifest|dep
	Description string `json:"description"`
	Target      string `json:"target,omitempty"`
	Applied     bool   `json:"applied"`
}

// Paths points at feature-local files.
type Paths struct {
	BoardDir      string `json:"board_dir"`
	HowItWorks    string `json:"how_it_works,omitempty"`
	WorkspaceDir  string `json:"workspace_dir,omitempty"`
	DepsDelta     string `json:"deps_delta,omitempty"`
	Worktree      string `json:"worktree,omitempty"`
}

// Board is the per-feature contract under .valid/features/<slug>/data.json.
type Board struct {
	Version           string                `json:"version"`
	Feature           string                `json:"feature"`
	Mode              string                `json:"mode"`
	Lifecycle         string                `json:"lifecycle"`
	Autonomy          string                `json:"autonomy,omitempty"` // in_the_loop (default) | above_the_loop
	NorthStar         string                `json:"north_star,omitempty"`
	What              []AcceptanceCriterion `json:"what"`
	Phases            []Phase               `json:"phases"`
	Tasks             []Task                `json:"tasks"`
	Decisions         []Decision            `json:"decisions"`
	Assumptions       []Assumption          `json:"assumptions"`
	HowItWorks        string                `json:"how_it_works,omitempty"`
	TDD               TDDState              `json:"tdd"`
	Audit             AuditState            `json:"audit"`
	Environment       EnvironmentState      `json:"environment"`
	PendingPromotions []Promotion           `json:"pending_promotions"`
	Paths             Paths                 `json:"paths"`
	PromotionSummary  string                `json:"promotion_summary,omitempty"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

// IsolationConfig controls soft vs hard isolation.
type IsolationConfig struct {
	Strict bool `json:"strict"`
}

// DatabaseConfig controls optional feature DB quarantine.
type DatabaseConfig struct {
	Enabled bool   `json:"enabled"`
	Recipe  string `json:"recipe"`
}

// MCPConfig points at the knowledge corpus.
type MCPConfig struct {
	KnowledgeRoot string `json:"knowledge_root"`
}

// Config is .valid/config.json (project-level).
type Config struct {
	Version              int             `json:"version"`
	TestCommand          string          `json:"test_command"`
	DashboardPort        int             `json:"dashboard_port"`         // principal / default dashboard
	MCPHTTPPort          int             `json:"mcp_http_port"`          // valid mcp --http
	FeatureDashboardPort int             `json:"feature_dashboard_port"` // 0 = auto from offset + slug
	FeaturePortOffset    int             `json:"feature_port_offset"`    // added to dashboard_port when auto
	Isolation            IsolationConfig `json:"isolation"`
	Database             DatabaseConfig  `json:"database"`
	MainDevcontainer     string          `json:"main_devcontainer"`
	ScriptsRoot          string          `json:"scripts_root"` // project MCP script nodes
	MCP                  MCPConfig       `json:"mcp"`
	Language             string          `json:"language"`
}

// DefaultConfig returns stock project config (database off, soft isolation).
func DefaultConfig() Config {
	return Config{
		Version:              2,
		TestCommand:          "go test ./...",
		DashboardPort:        7432,
		MCPHTTPPort:          7433,
		FeatureDashboardPort: 0,
		FeaturePortOffset:    100,
		Isolation:            IsolationConfig{Strict: false},
		Database:             DatabaseConfig{Enabled: false, Recipe: ""},
		MainDevcontainer:     ".devcontainer/devcontainer.json",
		ScriptsRoot:          ".valid/scripts",
		MCP:                  MCPConfig{KnowledgeRoot: ".valid/knowledge"},
		Language:             "repo",
	}
}

// ResolveFeatureDashboardPort picks the host port forwarded by a feature Dev Container.
// Always salts by slug so concurrent features never share one host port.
// FeatureDashboardPort, when set, is the base (instead of dashboard_port + offset).
func (c Config) ResolveFeatureDashboardPort(slug string) int {
	h := 0
	for _, r := range slug {
		h = (h*31 + int(r)) % 200
	}
	if h < 0 {
		h = -h
	}
	if c.FeatureDashboardPort > 0 {
		return c.FeatureDashboardPort + h
	}
	base := c.DashboardPort
	if base <= 0 {
		base = 7432
	}
	offset := c.FeaturePortOffset
	if offset <= 0 {
		offset = 100
	}
	return base + offset + h
}

// ResolveMCPHTTPAddr returns the listen address for valid mcp --http (loopback).
func (c Config) ResolveMCPHTTPAddr() string {
	p := c.MCPHTTPPort
	if p <= 0 {
		p = 7433
	}
	return fmt.Sprintf("127.0.0.1:%d", p)
}

var validLifecycle = map[string]struct{}{
	LifePlan: {}, LifeSpec: {}, LifeBuild: {}, LifeAudit: {}, LifeFinish: {}, LifeDone: {},
}

var validModes = map[string]struct{}{
	ModeFeature: {}, ModePatch: {}, ModeMinipatch: {},
}

var validAutonomy = map[string]struct{}{
	"": {}, AutonomyInTheLoop: {}, AutonomyAboveTheLoop: {},
}

var validTDD = map[string]struct{}{
	"": {}, TDDRed: {}, TDDGreen: {}, TDDRefactor: {},
}

var validTestStatuses = map[string]struct{}{
	TestPass: {}, TestFail: {}, TestPending: {},
}

var validTaskStatus = map[string]struct{}{
	"pending": {}, "doing": {}, "done": {},
}

// NewBoard creates a fresh board for a feature/patch/minipatch.
func NewBoard(feature, mode string) *Board {
	if mode == "" {
		mode = ModeFeature
	}
	return &Board{
		Version:   Version,
		Feature:   feature,
		Mode:      mode,
		Lifecycle: LifePlan,
		Autonomy:  AutonomyInTheLoop,
		What:      []AcceptanceCriterion{},
		Phases:    []Phase{},
		Tasks:     []Task{},
		Decisions: []Decision{},
		Assumptions: []Assumption{},
		TDD: TDDState{
			Cases: []TestCase{},
		},
		Audit: AuditState{
			Findings: []Finding{},
		},
		PendingPromotions: []Promotion{},
		UpdatedAt:         time.Now().UTC(),
	}
}

// Validate checks board invariants.
func (b *Board) Validate() error {
	if b == nil {
		return errors.New("board is nil")
	}
	if b.Version != Version {
		return fmt.Errorf("unsupported version %q (want %q)", b.Version, Version)
	}
	if strings.TrimSpace(b.Feature) == "" {
		return errors.New("feature is required")
	}
	if _, ok := validModes[b.Mode]; !ok {
		return fmt.Errorf("invalid mode %q", b.Mode)
	}
	if _, ok := validLifecycle[b.Lifecycle]; !ok {
		return fmt.Errorf("invalid lifecycle %q", b.Lifecycle)
	}
	if b.Autonomy != "" {
		if _, ok := validAutonomy[b.Autonomy]; !ok {
			return fmt.Errorf("invalid autonomy %q (want in_the_loop|above_the_loop)", b.Autonomy)
		}
	}
	if _, ok := validTDD[b.TDD.Phase]; !ok {
		return fmt.Errorf("invalid tdd phase %q", b.TDD.Phase)
	}
	ids := map[string]struct{}{}
	for i, ac := range b.What {
		if strings.TrimSpace(ac.ID) == "" {
			return fmt.Errorf("what[%d].id is required", i)
		}
		ids[ac.ID] = struct{}{}
	}
	for i, t := range b.Tasks {
		if strings.TrimSpace(t.ID) == "" {
			return fmt.Errorf("tasks[%d].id is required", i)
		}
		if _, ok := validTaskStatus[t.Status]; !ok && t.Status != "" {
			return fmt.Errorf("tasks[%d].status invalid %q", i, t.Status)
		}
		for _, c := range t.Covers {
			if _, ok := ids[c]; !ok && len(b.What) > 0 {
				// Allow covers before ACs are fully synced only when what is empty;
				// when what exists, covers must reference known ids.
				return fmt.Errorf("tasks[%d].covers references unknown AC %q", i, c)
			}
		}
	}
	for i, c := range b.TDD.Cases {
		if c.Name == "" {
			return fmt.Errorf("tdd.cases[%d].name is required", i)
		}
		if _, ok := validTestStatuses[c.Status]; !ok {
			return fmt.Errorf("tdd.cases[%d].status invalid %q", i, c.Status)
		}
	}
	if b.TDD.Passed < 0 || b.TDD.Failed < 0 || b.TDD.Total < 0 {
		return errors.New("tdd counts must be non-negative")
	}
	return nil
}

// Touch updates UpdatedAt.
func (b *Board) Touch() {
	b.UpdatedAt = time.Now().UTC()
}

// RecalculateTDD recomputes aggregates from Cases.
// Preserves an explicit refactor phase when tests are green; otherwise derives red/green.
func (b *Board) RecalculateTDD() {
	prev := b.TDD.Phase
	var passed, failed int
	var outputs []string
	for _, c := range b.TDD.Cases {
		switch c.Status {
		case TestPass:
			passed++
		case TestFail:
			failed++
		}
		if strings.TrimSpace(c.Output) != "" {
			outputs = append(outputs, fmt.Sprintf("[%s] %s", c.Name, strings.TrimSpace(c.Output)))
		}
	}
	b.TDD.Passed = passed
	b.TDD.Failed = failed
	b.TDD.Total = len(b.TDD.Cases)
	if len(outputs) > 0 {
		b.TDD.Output = strings.Join(outputs, "\n")
	}
	if failed > 0 || hasPending(b.TDD.Cases) {
		b.TDD.Phase = TDDRed
	} else if prev == TDDRefactor && passed > 0 {
		b.TDD.Phase = TDDRefactor
	} else if passed > 0 && failed == 0 && !hasPending(b.TDD.Cases) {
		b.TDD.Phase = TDDGreen
	}
}

func hasPending(cases []TestCase) bool {
	for _, c := range cases {
		if c.Status == TestPending {
			return true
		}
	}
	return false
}

// UncoveredACs returns AC ids not referenced by any task.covers.
func (b *Board) UncoveredACs() []string {
	covered := map[string]struct{}{}
	for _, t := range b.Tasks {
		for _, c := range t.Covers {
			covered[c] = struct{}{}
		}
	}
	var missing []string
	for _, ac := range b.What {
		if _, ok := covered[ac.ID]; !ok {
			missing = append(missing, ac.ID)
		}
	}
	return missing
}

// TestsGreen reports whether executable tests evidence is green.
func (b *Board) TestsGreen() bool {
	return b.TDD.Total > 0 && b.TDD.Failed == 0 && !hasPending(b.TDD.Cases) && b.TDD.Passed == b.TDD.Total
}

// ApplyReport upserts a single test result (TDD only).
func (b *Board) ApplyReport(status, testName, output, phase string) error {
	if b == nil {
		return errors.New("board is nil")
	}
	if phase != "" {
		phase = strings.ToLower(strings.TrimSpace(phase))
		if _, ok := validTDD[phase]; !ok || phase == "" {
			return fmt.Errorf("tdd phase must be red|green|refactor (got %q)", phase)
		}
		b.TDD.Phase = phase
	}
	if testName == "" {
		return errors.New("--test-name is required")
	}
	st := strings.ToLower(strings.TrimSpace(status))
	if st == "" {
		st = TestPending
	}
	if _, ok := validTestStatuses[st]; !ok {
		return fmt.Errorf("status must be pass|fail|pending (got %q)", status)
	}
	updated := false
	for i := range b.TDD.Cases {
		if b.TDD.Cases[i].Name == testName {
			b.TDD.Cases[i].Status = st
			if output != "" {
				b.TDD.Cases[i].Output = output
			}
			updated = true
			break
		}
	}
	if !updated {
		b.TDD.Cases = append(b.TDD.Cases, TestCase{Name: testName, Status: st, Output: output})
	}
	if output != "" {
		b.TDD.Output = output
	}
	b.RecalculateTDD()
	b.Touch()
	return b.Validate()
}

// UpsertTask adds or updates a task.
func (b *Board) UpsertTask(id, title, status, description string, covers []string) error {
	if b == nil {
		return errors.New("board is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("task id is required")
	}
	if status == "" {
		status = "pending"
	}
	if _, ok := validTaskStatus[status]; !ok {
		return fmt.Errorf("status must be pending|doing|done (got %q)", status)
	}
	if covers == nil {
		covers = []string{}
	}
	for i := range b.Tasks {
		if b.Tasks[i].ID == id {
			if title != "" {
				b.Tasks[i].Title = title
			}
			b.Tasks[i].Status = status
			if description != "" {
				b.Tasks[i].Description = description
			}
			if len(covers) > 0 {
				b.Tasks[i].Covers = covers
			}
			b.Touch()
			return b.Validate()
		}
	}
	if title == "" {
		title = id
	}
	b.Tasks = append(b.Tasks, Task{
		ID: id, Title: title, Status: status, Covers: covers, Description: description,
	})
	b.Touch()
	return b.Validate()
}

// SetWhat replaces acceptance criteria.
func (b *Board) SetWhat(items []AcceptanceCriterion) error {
	if b == nil {
		return errors.New("board is nil")
	}
	if items == nil {
		items = []AcceptanceCriterion{}
	}
	b.What = items
	if b.Lifecycle == LifePlan {
		b.Lifecycle = LifeSpec
	}
	b.Touch()
	return b.Validate()
}

// SetLifecycle sets the board lifecycle stage.
func (b *Board) SetLifecycle(life string) error {
	life = strings.ToLower(strings.TrimSpace(life))
	if _, ok := validLifecycle[life]; !ok {
		return fmt.Errorf("lifecycle must be plan|spec|build|audit|finish|done (got %q)", life)
	}
	b.Lifecycle = life
	b.Touch()
	return b.Validate()
}

// SetAutonomy sets human-presence mode for build: in_the_loop | above_the_loop.
func (b *Board) SetAutonomy(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = AutonomyInTheLoop
	}
	if _, ok := validAutonomy[mode]; !ok {
		return fmt.Errorf("autonomy must be in_the_loop|above_the_loop (got %q)", mode)
	}
	b.Autonomy = mode
	b.Touch()
	return b.Validate()
}

// LoadBoard reads and validates a board JSON file.
func LoadBoard(path string) (*Board, error) {
	if path == "" {
		return nil, errors.New("path is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read board: %w", err)
	}
	var b Board
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, fmt.Errorf("parse board: %w", err)
	}
	if b.Autonomy == "" {
		b.Autonomy = AutonomyInTheLoop
	}
	if err := b.Validate(); err != nil {
		return nil, fmt.Errorf("invalid board: %w", err)
	}
	return &b, nil
}

// SaveBoard validates and atomically writes a board.
func SaveBoard(path string, b *Board) error {
	if path == "" {
		return errors.New("path is required")
	}
	if b == nil {
		return errors.New("board is nil")
	}
	if b.Autonomy == "" {
		b.Autonomy = AutonomyInTheLoop
	}
	b.Touch()
	if err := b.Validate(); err != nil {
		return fmt.Errorf("invalid board: %w", err)
	}
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal board: %w", err)
	}
	raw = append(raw, '\n')
	return AtomicWrite(path, raw)
}

// FeatureDir returns .valid/features/<slug> under repo root.
func FeatureDir(root, slug string) string {
	return filepath.Join(root, ValidDir, FeaturesDir, slug)
}

// BoardPath returns .valid/features/<slug>/data.json.
func BoardPath(root, slug string) string {
	return filepath.Join(FeatureDir(root, slug), DataFileName)
}

// ConfigPath returns .valid/config.json.
func ConfigPath(root string) string {
	return filepath.Join(root, ValidDir, ConfigFileName)
}

// KnowledgePath returns the knowledge root from config (or default).
func KnowledgePath(root string, cfg Config) string {
	rel := cfg.MCP.KnowledgeRoot
	if rel == "" {
		rel = filepath.Join(ValidDir, KnowledgeDir)
	}
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(root, rel)
}

// ScriptsPath returns the project scripts root (MCP script tool nodes).
func ScriptsPath(root string, cfg Config) string {
	rel := cfg.ScriptsRoot
	if strings.TrimSpace(rel) == "" {
		rel = filepath.Join(ValidDir, "scripts")
	}
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(root, rel)
}

// LoadConfig reads config or returns defaults.
func LoadConfig(root string) (Config, error) {
	path := ConfigPath(root)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("read config.json: %w", err)
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config.json: %w", err)
	}
	if cfg.Version == 0 {
		cfg.Version = 2
	}
	if cfg.DashboardPort <= 0 {
		cfg.DashboardPort = 7432
	}
	if cfg.MCPHTTPPort <= 0 {
		cfg.MCPHTTPPort = 7433
	}
	if cfg.FeaturePortOffset <= 0 {
		cfg.FeaturePortOffset = 100
	}
	if strings.TrimSpace(cfg.TestCommand) == "" {
		cfg.TestCommand = "go test ./..."
	}
	if strings.TrimSpace(cfg.MainDevcontainer) == "" {
		cfg.MainDevcontainer = ".devcontainer/devcontainer.json"
	}
	if strings.TrimSpace(cfg.MCP.KnowledgeRoot) == "" {
		cfg.MCP.KnowledgeRoot = ".valid/knowledge"
	}
	if strings.TrimSpace(cfg.ScriptsRoot) == "" {
		cfg.ScriptsRoot = ".valid/scripts"
	}
	return cfg, nil
}

// SaveConfig atomically writes config.json.
func SaveConfig(root string, cfg Config) error {
	if cfg.Version == 0 {
		cfg.Version = 2
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config.json: %w", err)
	}
	raw = append(raw, '\n')
	return AtomicWrite(ConfigPath(root), raw)
}

// ListFeatureSlugs lists feature directories under .valid/features/.
func ListFeatureSlugs(root string) ([]string, error) {
	dir := filepath.Join(root, ValidDir, FeaturesDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			if _, err := os.Stat(BoardPath(root, e.Name())); err == nil {
				out = append(out, e.Name())
			}
		}
	}
	return out, nil
}
