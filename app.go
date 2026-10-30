// Package rapid is the RAPID AI analysis application.
//
// RAPID does not simulate the Azzurro platform: it runs a real atp.Server and
// uses the real first-party modules it mounts. Project metadata, analyses and
// generated wrappers are pod records; generated wrappers and the dashboard are
// hosted by song; share links are shepherd capability tokens; and the vault,
// header policy and function registry come from atp itself.
package rapid

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"azzurrotech/atp"
	"azzurrotech/pod"
	"azzurrotech/shepherd/capability"

	"rapid.ai/pkg/agent"
	"rapid.ai/pkg/pricing"
)

// AIProject is a project RAPID knows about, whether discovered in the catalog
// or submitted for analysis.
type AIProject struct {
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	URL          string   `json:"url,omitempty"`
	Repository   string   `json:"repository,omitempty"`
	Language     string   `json:"language,omitempty"`
	License      string   `json:"license,omitempty"`
	Status       string   `json:"status,omitempty"`
	Path         string   `json:"path,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// AnalysisRecord is the persisted summary of one analysis run. The full result
// is kept as JSON in Result so the catalog can list runs cheaply.
type AnalysisRecord struct {
	ID          string    `json:"id"`
	ProjectName string    `json:"project_name"`
	ProjectPath string    `json:"project_path"`
	AnalyzedAt  time.Time `json:"analyzed_at"`
	TotalFiles  int       `json:"total_files"`
	TotalSize   int64     `json:"total_size"`
	Complexity  float64   `json:"complexity"`
	HasTests    bool      `json:"has_tests"`
	HasCI       bool      `json:"has_ci"`
	Tokens      int       `json:"tokens"`
	Cost        float64   `json:"cost"`
	Result      string    `json:"result"`
}

// WrapperRecord is the persisted summary of one generated wrapper.
type WrapperRecord struct {
	ID           string    `json:"id"`
	ProjectName  string    `json:"project_name"`
	TemplateType string    `json:"template_type"`
	FilePath     string    `json:"file_path"`
	Tokens       int       `json:"tokens"`
	Cost         float64   `json:"cost"`
	Currency     string    `json:"currency"`
	Enterprise   bool      `json:"enterprise"`
	CreatedAt    time.Time `json:"created_at"`
}

// RAPIDApplication owns the platform and the analysis services.
type RAPIDApplication struct {
	cfg       *RAPIDConfig
	platform  *atp.Server
	store     *pod.Store
	analyzer  *agent.ProjectAnalyzer
	generator *agent.WrapperGenerator
	pricing   *pricing.PriceLogger
	started   time.Time
}

// NewRAPIDApplication builds the real platform and the services on top of it.
func NewRAPIDApplication(cfg *RAPIDConfig) (*RAPIDApplication, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if err := cfg.Normalize(); err != nil {
		return nil, err
	}
	secret, err := cfg.ResolveSecret()
	if err != nil {
		return nil, err
	}

	platform, err := atp.New(atp.Options{
		Root:   cfg.DataDir,
		Addr:   cfg.Addr(),
		Secret: secret,
		Title:  "RAPID — AI analysis",
	})
	if err != nil {
		return nil, fmt.Errorf("rapid: open atp platform: %w", err)
	}

	logger, err := pricing.NewPriceLogger(1000, filepath.Join(cfg.DataDir, "pricing.log"), "json")
	if err != nil {
		return nil, fmt.Errorf("rapid: open pricing log: %w", err)
	}

	analyzer, err := agent.NewProjectAnalyzer(cfg.OllamaURL, cfg.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("rapid: analyzer: %w", err)
	}

	app := &RAPIDApplication{
		cfg:       cfg,
		platform:  platform,
		store:     platform.Pod(),
		analyzer:  analyzer,
		generator: agent.NewWrapperGenerator(cfg.TemplateDir, logger),
		pricing:   logger,
		started:   time.Now().UTC(),
	}
	if err := app.ensureSchema(); err != nil {
		return nil, err
	}
	if err := app.seedProjects(); err != nil {
		return nil, err
	}
	return app, nil
}

// ensureSchema creates or widens the pod tables RAPID owns. It is safe on every
// start because pod only adds columns.
func (a *RAPIDApplication) ensureSchema() error {
	for table, sample := range map[string]any{
		"projects": AIProject{},
		"analyses": AnalysisRecord{},
		"wrappers": WrapperRecord{},
	} {
		if err := a.store.EnsureStructSchema(table, sample); err != nil {
			return fmt.Errorf("rapid: schema %s: %w", table, err)
		}
	}
	return nil
}

// seedProjects writes the default catalog once, so a fresh install has
// something to show without pretending it discovered anything.
func (a *RAPIDApplication) seedProjects() error {
	count, err := a.projectCount()
	if err != nil || count > 0 {
		return err
	}
	defaults := []AIProject{
		{
			Name:        "Go",
			Description: "Open source programming language",
			URL:         "https://go.dev",
			Repository:  "https://github.com/golang/go",
			Language:    "Go",
			License:     "BSD-3-Clause",
			Status:      "active",
			Tags:        []string{"language", "programming"},
		},
		{
			Name:        "Docker",
			Description: "Containerization platform",
			URL:         "https://www.docker.com",
			Repository:  "https://github.com/moby/moby",
			Language:    "Go",
			License:     "Apache-2.0",
			Status:      "active",
			Tags:        []string{"containers", "infrastructure"},
		},
	}
	for _, p := range defaults {
		if err := a.upsertProject(p); err != nil {
			return err
		}
	}
	return nil
}

func (a *RAPIDApplication) projectCount() (int, error) {
	ids, err := a.store.IDs("projects")
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

func (a *RAPIDApplication) upsertProject(p AIProject) error {
	if p.ID == "" {
		p.ID = slug(p.Name)
	}
	if p.Status == "" {
		p.Status = "active"
	}
	_, _, err := a.store.UpsertStruct("projects", p.ID, p)
	if err != nil {
		return fmt.Errorf("rapid: store project %s: %w", p.ID, err)
	}
	return nil
}

// HandleGetProjects returns the project catalog from pod, ordered by name.
func (a *RAPIDApplication) HandleGetProjects() []AIProject {
	projects, err := pod.QueryStruct[AIProject](a.store, "projects", nil, nil)
	if err != nil {
		return []AIProject{}
	}
	out := make([]AIProject, 0, len(projects))
	for _, p := range projects {
		if p != nil {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// AnalyzeProject analyzes a directory. This is the CLI entry point.
func (a *RAPIDApplication) AnalyzeProject(projectPath string) (agent.AnalysisResult, error) {
	result, err := a.analyzer.AnalyzeProject(projectPath)
	if err != nil {
		return agent.AnalysisResult{}, err
	}
	if err := a.recordAnalysis(result); err != nil {
		return result, err
	}
	return result, nil
}

// AnalyzeAIProject analyzes the path carried by an API project description.
func (a *RAPIDApplication) AnalyzeAIProject(p AIProject) (agent.AnalysisResult, error) {
	path := strings.TrimSpace(p.Path)
	if path == "" {
		return agent.AnalysisResult{}, fmt.Errorf("rapid: project path is required")
	}
	result, err := a.AnalyzeProject(path)
	if err != nil {
		return agent.AnalysisResult{}, err
	}
	if p.Name != "" {
		result.ProjectName = p.Name
	}
	return result, nil
}

func (a *RAPIDApplication) recordAnalysis(result agent.AnalysisResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("rapid: encode analysis: %w", err)
	}
	rec := AnalysisRecord{
		ID:          a.store.NewID(),
		ProjectName: result.ProjectName,
		ProjectPath: result.ProjectPath,
		AnalyzedAt:  result.AnalysisDate,
		TotalFiles:  result.TotalFiles,
		TotalSize:   result.TotalSize,
		Complexity:  result.Complexity,
		HasTests:    result.HasTests,
		HasCI:       result.HasCI,
		Tokens:      result.Pricing.EstimatedTokens,
		Cost:        result.Pricing.EstimatedCost,
		Result:      string(payload),
	}
	if _, _, err := a.store.UpsertStruct("analyses", rec.ID, rec); err != nil {
		return fmt.Errorf("rapid: store analysis: %w", err)
	}
	return nil
}

// StoreProject adds or replaces a catalog entry.
func (a *RAPIDApplication) StoreProject(p AIProject) error {
	return a.upsertProject(p)
}

// GenerateWrapper analyzes a directory and generates the wrapper its dominant
// language calls for, persisting a wrapper record.
func (a *RAPIDApplication) GenerateWrapper(projectPath string) (*agent.BusinessWrapper, error) {
	return a.CreateWrapperWithOptions(projectPath, "", a.cfg.Enterprise)
}

// CreateWrapperWithOptions generates a wrapper with an explicit template type
// and pricing tier; an empty template type is chosen from the languages found.
func (a *RAPIDApplication) CreateWrapperWithOptions(projectPath, templateType string, enterprise bool) (*agent.BusinessWrapper, error) {
	result, err := a.AnalyzeProject(projectPath)
	if err != nil {
		return nil, err
	}
	if templateType == "" {
		templateType = templateTypeFor(result.Files)
	}
	wrapper, err := a.generator.GenerateWrapper(result.ProjectName, templateType, a.cfg.OutputDir, enterprise)
	if err != nil {
		return nil, err
	}

	rec := WrapperRecord{
		ID:           a.store.NewID(),
		ProjectName:  wrapper.ProjectName,
		TemplateType: wrapper.TemplateType,
		FilePath:     wrapper.FilePath,
		Tokens:       wrapper.Pricing.TokensUsed,
		Cost:         wrapper.Pricing.Cost,
		Currency:     wrapper.Pricing.Currency,
		Enterprise:   wrapper.Pricing.Enterprise,
		CreatedAt:    wrapper.Timestamp,
	}
	if _, _, err := a.store.UpsertStruct("wrappers", rec.ID, rec); err != nil {
		return nil, fmt.Errorf("rapid: store wrapper: %w", err)
	}
	return wrapper, nil
}

// GenerateNamedWrapper generates a wrapper from a project name alone, for the
// CLI's name-based workflow, and records it like any other wrapper.
func (a *RAPIDApplication) GenerateNamedWrapper(projectName, templateType, outputDir string, enterprise bool) (*agent.BusinessWrapper, error) {
	if outputDir == "" {
		outputDir = a.cfg.OutputDir
	}
	wrapper, err := a.generator.GenerateWrapper(projectName, templateType, outputDir, enterprise)
	if err != nil {
		return nil, err
	}
	rec := WrapperRecord{
		ID:           a.store.NewID(),
		ProjectName:  wrapper.ProjectName,
		TemplateType: wrapper.TemplateType,
		FilePath:     wrapper.FilePath,
		Tokens:       wrapper.Pricing.TokensUsed,
		Cost:         wrapper.Pricing.Cost,
		Currency:     wrapper.Pricing.Currency,
		Enterprise:   wrapper.Pricing.Enterprise,
		CreatedAt:    wrapper.Timestamp,
	}
	if _, _, err := a.store.UpsertStruct("wrappers", rec.ID, rec); err != nil {
		return nil, fmt.Errorf("rapid: store wrapper: %w", err)
	}
	return wrapper, nil
}

// CreateWrapper generates a wrapper and returns the file it wrote.
func (a *RAPIDApplication) CreateWrapper(projectPath string) (string, error) {
	wrapper, err := a.GenerateWrapper(projectPath)
	if err != nil {
		return "", err
	}
	return wrapper.FilePath, nil
}

// Analyses returns every recorded analysis, newest first.
func (a *RAPIDApplication) Analyses() []AnalysisRecord {
	records, err := pod.QueryStruct[AnalysisRecord](a.store, "analyses", nil,
		&pod.QueryOptions{OrderBy: "analyzed_at", Desc: true})
	if err != nil {
		return []AnalysisRecord{}
	}
	out := make([]AnalysisRecord, 0, len(records))
	for _, r := range records {
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}

// Wrappers returns every recorded wrapper, newest first.
func (a *RAPIDApplication) Wrappers() []WrapperRecord {
	records, err := pod.QueryStruct[WrapperRecord](a.store, "wrappers", nil,
		&pod.QueryOptions{OrderBy: "created_at", Desc: true})
	if err != nil {
		return []WrapperRecord{}
	}
	out := make([]WrapperRecord, 0, len(records))
	for _, r := range records {
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}

// Share issues a revocable shepherd capability token for one record. The token
// is the platform's, not a RAPID invention.
func (a *RAPIDApplication) Share(kind, id string, ttl time.Duration) (*capability.Share, error) {
	return capability.Issue(a.platform.Shepherd().Manager(), capability.IssueOptions{
		Target:  capability.Target{Kind: kind, ID: id},
		TTL:     ttl,
		Issuer:  "rapid",
		Subject: "share",
	})
}

// ValidateConfiguration checks the parts RAPID can check without a network.
func (a *RAPIDApplication) ValidateConfiguration() error {
	if a.cfg == nil {
		return fmt.Errorf("rapid: no configuration")
	}
	if err := a.cfg.Normalize(); err != nil {
		return err
	}
	if len(a.cfg.Secret) < 32 {
		return fmt.Errorf("rapid: secret is shorter than 32 bytes")
	}
	if a.platform == nil || a.store == nil || a.analyzer == nil || a.generator == nil || a.pricing == nil {
		return fmt.Errorf("rapid: application is not fully initialised")
	}
	return nil
}

// ValidateIntegrations exercises each module with a real operation.
func (a *RAPIDApplication) ValidateIntegrations() bool {
	return len(a.IntegrationFailures()) == 0
}

// PublishDashboard hosts the dashboard through song, in the "rapid" silo.
// atp serves it at /rapid/, like any other hosted application.
func (a *RAPIDApplication) PublishDashboard(indexHTML []byte) error {
	store := a.platform.Song().Store()
	if !store.HasSilo("rapid") {
		if err := store.CreateSilo("rapid"); err != nil {
			return fmt.Errorf("rapid: create song silo: %w", err)
		}
	}
	if _, err := store.Stat("rapid", "index.html"); err == nil {
		_, err = store.Update("rapid", "index.html", string(indexHTML))
		if err != nil {
			return fmt.Errorf("rapid: update dashboard: %w", err)
		}
		return nil
	}
	if _, err := store.Create("rapid", "index.html", string(indexHTML)); err != nil {
		return fmt.Errorf("rapid: publish dashboard: %w", err)
	}
	return nil
}

// Handler wraps a host handler with the real platform. atp keeps its own paths
// (/pod/, /secrets, /_song/, hosted apps) and delegates everything else, which
// is where RAPID's API and dashboard live.
func (a *RAPIDApplication) Handler(next http.Handler) http.Handler {
	return a.platform.Middleware(next)
}

// Health is the live status of the platform RAPID is running on. It delegates
// the module checks to atp, so every atp application reports the same shape.
func (a *RAPIDApplication) Health() map[string]any {
	if a.platform == nil {
		return map[string]any{"status": "degraded", "components": map[string]bool{"rapid": true}}
	}
	return a.healthReport(a.platform.Health())
}

// HealthCheck is Health with a real probe of every module. It is the same
// report with ?check=1, and is what /api/integrations uses.
func (a *RAPIDApplication) HealthCheck() map[string]any {
	if a.platform == nil {
		return map[string]any{"status": "degraded", "components": map[string]bool{"rapid": true}}
	}
	return a.healthReport(a.platform.Check())
}

// healthReport flattens atp's component list into the map the dashboard reads
// while keeping the detailed report available under "platform".
func (a *RAPIDApplication) healthReport(report atp.HealthReport) map[string]any {
	components := map[string]bool{"rapid": a.cfg != nil}
	for _, c := range report.Components {
		components[c.Name] = c.OK
	}
	status := report.Status
	if status == "" {
		status = "degraded"
	}
	return map[string]any{
		"status":        status,
		"ok":            report.OK(),
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"version":       a.cfg.Version,
		"platform":      "azzurro",
		"components":    components,
		"data_dir":      a.cfg.DataDir,
		"module_report": report,
	}
}

// IntegrationFailures returns one message per integration that did not answer.
// The probes are atp's, not RAPID's, so a new module is covered platform-wide.
func (a *RAPIDApplication) IntegrationFailures() []string {
	if a.platform == nil {
		return []string{"atp: platform is not open"}
	}
	return a.platform.Check().Failures
}

// Metrics is the observability surface the README promises: record counts,
// pricing statistics and uptime, all read from what the platform already
// stores rather than from a second counter that could drift.
func (a *RAPIDApplication) Metrics() map[string]any {
	out := map[string]any{
		"uptime_seconds": int64(0),
		"pricing":        pricing.PricingStatistics{},
		"counts":         map[string]int{},
	}
	if !a.started.IsZero() {
		out["uptime_seconds"] = int64(time.Since(a.started).Seconds())
	}
	if a.pricing != nil {
		out["pricing"] = a.pricing.GetStatistics()
	}
	if a.store != nil {
		counts := map[string]int{}
		for _, table := range []string{"projects", "analyses", "wrappers"} {
			if ids, err := a.store.IDs(table); err == nil {
				counts[table] = len(ids)
			}
		}
		out["counts"] = counts
	}
	return out
}

// ExportTables is the whitelist of record sets an operator may export, mapped
// to the field it is ordered by. It keeps an export from becoming an arbitrary
// table dump.
var ExportTables = map[string]string{
	"projects": "name",
	"analyses": "analyzed_at",
	"wrappers": "created_at",
}

// Export writes one of RAPID's record sets in pod's shared JSON/CSV shape.
// An empty columns list exports every field except the analyses' full result
// blob, which would dwarf the rest of the row; ask for it explicitly.
func (a *RAPIDApplication) Export(w io.Writer, table, format string, columns []string) error {
	orderBy, ok := ExportTables[table]
	if !ok {
		return fmt.Errorf("rapid: unknown export table %q", table)
	}
	records, err := a.store.Query(table, nil, &pod.QueryOptions{OrderBy: orderBy, Desc: table != "projects"})
	if err != nil {
		return fmt.Errorf("rapid: export %s: %w", table, err)
	}
	cols := make([]pod.ExportColumn, 0, len(columns))
	for _, name := range columns {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		cols = append(cols, pod.ExportColumn{Field: name})
	}
	if len(cols) == 0 && table == "analyses" {
		for _, f := range []string{"id", "project_name", "project_path", "analyzed_at", "total_files", "total_size", "complexity", "has_tests", "has_ci", "tokens", "cost"} {
			cols = append(cols, pod.ExportColumn{Field: f})
		}
	}
	return pod.ExportRecords(w, records, pod.ExportOptions{Format: format, Columns: cols})
}

// RevokeShare cancels a share token by its value. Shepherd owns the deny list,
// so the revocation is visible to every host reading the same process.
func (a *RAPIDApplication) RevokeShare(token string) (capability.Target, error) {
	if a.platform == nil || a.platform.Shepherd() == nil {
		return capability.Target{}, fmt.Errorf("rapid: shepherd is not mounted")
	}
	return capability.RevokeRaw(a.platform.Shepherd().Manager(), a.platform.Shepherd(), token)
}

// Config returns the effective configuration.
func (a *RAPIDApplication) Config() RAPIDConfig { return *a.cfg }

// Platform exposes the atp server for hosts that need to register functions.
func (a *RAPIDApplication) Platform() *atp.Server { return a.platform }

// Store exposes the pod store for hosts that need the same records.
func (a *RAPIDApplication) Store() *pod.Store { return a.store }

// Close releases the pricing log.
func (a *RAPIDApplication) Close() error {
	if a != nil && a.pricing != nil {
		return a.pricing.Close()
	}
	return nil
}

// templateTypeFor chooses the wrapper template from the languages present.
func templateTypeFor(files []agent.FileInfo) string {
	counts := map[string]int{}
	for _, f := range files {
		switch f.Extension {
		case ".go":
			counts["go-wrapper"]++
		case ".py":
			counts["python-wrapper"]++
		case ".js", ".ts", ".jsx", ".tsx":
			counts["node-wrapper"]++
		}
	}
	best, bestN := "generic-wrapper", 0
	for kind, n := range counts {
		if n > bestN || (n == bestN && kind < best) {
			best, bestN = kind, n
		}
	}
	return best
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteByte('-')
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "project"
	}
	return out
}
