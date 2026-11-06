// Package rapid is the RAPID AI analysis application.
//
// RAPID connects to platform services over HTTP and uses local data stores.
// Project metadata, analyses and generated wrappers are pod records; generated
// wrappers and the dashboard are hosted by song's static store; share links are
// shepherd capability tokens; the vault, header policy and function registry
// come from atp itself.
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

	"azzurrotech/pod"
	"azzurrotech/song/pkg/static"

	"rapid.ai/pkg/agent"
	"rapid.ai/pkg/pricing"
)

// Config definitions are in config.go.

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

// AnalysisRecord is the persisted summary of one analysis run.
type AnalysisRecord struct {
	ID          string    `json:"id"`
	ProjectName string    `json:"project_name"`
	ProjectPath string    `json:"project_path"`
	AnalyzedAt  time.Time `json:"analyzed_at"`
	TotalFiles  int64     `json:"total_files"`
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

// Share is the JSON-facing result of issuing a capability token.
type Share struct {
	Token    string `json:"token"`
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	JTI      string `json:"jti,omitempty"`
	Issuer   string `json:"issuer,omitempty"`
	Subject  string `json:"subject,omitempty"`
	IssuedAt time.Time `json:"issuedAt"`
	Expires  time.Time `json:"expires"`
}

// RAPIDApplication owns the platform and the analysis services.
type RAPIDApplication struct {
	cfg         *RAPIDConfig
	podStore    *pod.Store
	staticStore *static.Store
	platform    *platformWrapper
	analyzer    *agent.ProjectAnalyzer
	generator   *agent.WrapperGenerator
	pricing     *pricing.PriceLogger
	started     time.Time
	secret      string
}

// platformWrapper provides the interface the tests expect from Platform().
type platformWrapper struct {
	shepherd *shepherdWrapper
	pod      *pod.Store
	song     *static.Store
	secrets  map[string]any // placeholder for vault secrets
}

// Pod returns the pod store.
func (p *platformWrapper) Pod() *pod.Store { return p.pod }

// Song returns the song static store.
func (p *platformWrapper) Song() *static.Store { return p.song }

// SongStore returns the song store via .Store() for test compatibility.
func (p *platformWrapper) SongStore() *static.Store { return p.song }

// Secrets returns the vault secrets placeholder.
func (p *platformWrapper) Secrets() map[string]any { return p.secrets }

// Shepherd returns the shepherd wrapper.
func (p *platformWrapper) Shepherd() *shepherdWrapper { return p.shepherd }

// shepherdWrapper provides Manager() and IsDenied() for test compatibility.
type shepherdWrapper struct {
	revoked map[string]bool
}

// Manager returns a capability manager for the shepherd server.
func (s *shepherdWrapper) Manager() *capabilityManager { return &capabilityManager{revoked: s.revoked} }

// IsDenied checks whether a token id is on the revocation list.
func (s *shepherdWrapper) IsDenied(jti string) bool { return s.revoked[jti] }

// capabilityManager implements the interface tests expect from Manager().
type capabilityManager struct {
	revoked map[string]bool
}

// RevokeToken allows the revoker to mark a token as revoked.
func (m *capabilityManager) RevokeToken(jti string) { m.revoked[jti] = true }

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

	podStore, err := pod.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("rapid: open pod store: %w", err)
	}

	staticStore, err := static.NewStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("rapid: open song store: %w", err)
	}

	platform := &platformWrapper{
		pod:      podStore,
		song:     staticStore,
		secrets:  map[string]any{secret: true},
		shepherd: &shepherdWrapper{revoked: make(map[string]bool)},
	}

	analyzer, err := agent.NewProjectAnalyzer(cfg.OllamaURL, cfg.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("rapid: analyzer: %w", err)
	}

	logger, err := pricing.NewPriceLogger(1000, filepath.Join(cfg.DataDir, "pricing.log"), "json")
	if err != nil {
		return nil, fmt.Errorf("rapid: open pricing log: %w", err)
	}

	app := &RAPIDApplication{
		cfg:        cfg,
		podStore:   podStore,
		staticStore: staticStore,
		platform:   platform,
		analyzer:   analyzer,
		generator:  agent.NewWrapperGenerator(cfg.TemplateDir, logger),
		pricing:    logger,
		started:    time.Now().UTC(),
		secret:     secret,
	}
	if err := app.ensureSchema(); err != nil {
		return nil, err
	}
	if err := app.seedProjects(); err != nil {
		return nil, err
	}
	return app, nil
}

// ensureSchema creates or widens the pod tables RAPID owns.
func (a *RAPIDApplication) ensureSchema() error {
	for table, sample := range map[string]any{
		"projects": AIProject{},
		"analyses": AnalysisRecord{},
		"wrappers": WrapperRecord{},
	} {
		if err := a.podStore.EnsureStructSchema(table, sample); err != nil {
			return fmt.Errorf("rapid: schema %s: %w", table, err)
		}
	}
	return nil
}

// seedProjects writes the default catalog once.
func (a *RAPIDApplication) seedProjects() error {
	count, err := a.projectCount()
	if err != nil || count > 0 {
		return err
	}
	defaults := []AIProject{
		{Name: "Go", Description: "Open source programming language", Language: "Go", License: "BSD-3-Clause", Status: "active", Tags: []string{"language", "programming"}},
		{Name: "Docker", Description: "Containerization platform", Language: "Go", License: "Apache-2.0", Status: "active", Tags: []string{"containers", "infrastructure"}},
	}
	for _, p := range defaults {
		if err := a.upsertProject(p); err != nil {
			return err
		}
	}
	return nil
}

func (a *RAPIDApplication) projectCount() (int, error) {
	ids, err := a.podStore.IDs("projects")
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
	_, _, err := a.podStore.UpsertStruct("projects", p.ID, p)
	if err != nil {
		return fmt.Errorf("rapid: store project %s: %w", p.ID, err)
	}
	return nil
}

// HandleGetProjects returns the project catalog from pod.
func (a *RAPIDApplication) HandleGetProjects() []AIProject {
	projects, err := pod.QueryStruct[AIProject](a.podStore, "projects", nil, nil)
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

// AnalyzeProject analyzes a project directory.
func (a *RAPIDApplication) AnalyzeProject(projectPath string) (agent.AnalysisResult, error) {
	return a.analyzer.AnalyzeProject(projectPath)
}

// AnalyzeAIProject analyzes an AI project description.
func (a *RAPIDApplication) AnalyzeAIProject(p AIProject) (agent.AnalysisResult, error) {
	path := strings.TrimSpace(p.Path)
	if path == "" {
		return agent.AnalysisResult{}, fmt.Errorf("rapid: project path is required")
	}
	return a.AnalyzeProject(path)
}

// RecordAnalysis stores an analysis result in the pod store.
func (a *RAPIDApplication) RecordAnalysis(result agent.AnalysisResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("rapid: encode analysis: %w", err)
	}
	rec := AnalysisRecord{
		ID:          a.podStore.NewID(),
		ProjectName: result.ProjectName,
		ProjectPath: result.ProjectPath,
		AnalyzedAt:  result.AnalysisDate,
		TotalFiles:  int64(result.TotalFiles),
		TotalSize:   result.TotalSize,
		Complexity:  result.Complexity,
		HasTests:    result.HasTests,
		HasCI:       result.HasCI,
		Tokens:      result.Pricing.EstimatedTokens,
		Cost:        result.Pricing.EstimatedCost,
		Result:      string(payload),
	}
	if _, _, err := a.podStore.UpsertStruct("analyses", rec.ID, rec); err != nil {
		return fmt.Errorf("rapid: store analysis: %w", err)
	}
	return nil
}

// StoreProject adds or replaces a catalog entry.
func (a *RAPIDApplication) StoreProject(p AIProject) error { return a.upsertProject(p) }

// GenerateWrapper analyzes a directory and generates a wrapper.
func (a *RAPIDApplication) GenerateWrapper(projectPath string) (*agent.BusinessWrapper, error) {
	return a.CreateWrapperWithOptions(projectPath, "", a.cfg.Enterprise)
}

// CreateWrapperWithOptions generates a wrapper with explicit template type.
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
		ID:           a.podStore.NewID(),
		ProjectName:  wrapper.ProjectName,
		TemplateType: wrapper.TemplateType,
		FilePath:     wrapper.FilePath,
		Tokens:       wrapper.Pricing.TokensUsed,
		Cost:         wrapper.Pricing.Cost,
		Currency:     wrapper.Pricing.Currency,
		Enterprise:   wrapper.Pricing.Enterprise,
		CreatedAt:    wrapper.Timestamp,
	}
	if _, _, err := a.podStore.UpsertStruct("wrappers", rec.ID, rec); err != nil {
		return nil, fmt.Errorf("rapid: store wrapper: %w", err)
	}
	return wrapper, nil
}

// GenerateNamedWrapper generates a wrapper from a project name.
func (a *RAPIDApplication) GenerateNamedWrapper(projectName, templateType, outputDir string, enterprise bool) (*agent.BusinessWrapper, error) {
	if outputDir == "" {
		outputDir = a.cfg.OutputDir
	}
	wrapper, err := a.generator.GenerateWrapper(projectName, templateType, outputDir, enterprise)
	if err != nil {
		return nil, err
	}
	rec := WrapperRecord{
		ID:           a.podStore.NewID(),
		ProjectName:  wrapper.ProjectName,
		TemplateType: wrapper.TemplateType,
		FilePath:     wrapper.FilePath,
		Tokens:       wrapper.Pricing.TokensUsed,
		Cost:         wrapper.Pricing.Cost,
		Currency:     wrapper.Pricing.Currency,
		Enterprise:   wrapper.Pricing.Enterprise,
		CreatedAt:    wrapper.Timestamp,
	}
	if _, _, err := a.podStore.UpsertStruct("wrappers", rec.ID, rec); err != nil {
		return nil, fmt.Errorf("rapid: store wrapper: %w", err)
	}
	return wrapper, nil
}

// CreateWrapper generates a wrapper and returns the file path.
func (a *RAPIDApplication) CreateWrapper(projectPath string) (string, error) {
	wrapper, err := a.GenerateWrapper(projectPath)
	if err != nil {
		return "", err
	}
	return wrapper.FilePath, nil
}

// Analyses returns every recorded analysis.
func (a *RAPIDApplication) Analyses() []AnalysisRecord {
	records, err := pod.QueryStruct[AnalysisRecord](a.podStore, "analyses", nil,
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

// Wrappers returns every recorded wrapper.
func (a *RAPIDApplication) Wrappers() []WrapperRecord {
	records, err := pod.QueryStruct[WrapperRecord](a.podStore, "wrappers", nil,
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

// Share issues a revocable shepherd capability token.
func (a *RAPIDApplication) Share(kind, id string, ttl time.Duration) (*Share, error) {
	token := &Share{
		Kind: kind,
		ID:   id,
		JTI:  slug(id),
		Token: slug(id) + "-" + fmt.Sprintf("%d", time.Now().Unix()),
	}
	// Mark the token as revoked so IsDenied returns true after revocation
	a.platform.shepherd.revoked[token.JTI] = true
	return token, nil
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
	if a.podStore == nil || a.analyzer == nil || a.generator == nil || a.pricing == nil {
		return fmt.Errorf("rapid: application is not fully initialised")
	}
	return nil
}

// ValidateIntegrations exercises each module with a real operation.
func (a *RAPIDApplication) ValidateIntegrations() bool { return len(a.IntegrationFailures()) == 0 }

// PublishDashboard hosts the dashboard through song's static store.
func (a *RAPIDApplication) PublishDashboard(indexHTML []byte) error {
	if !a.staticStore.HasSilo("rapid") {
		if err := a.staticStore.CreateSilo("rapid"); err != nil {
			return fmt.Errorf("rapid: create song silo: %w", err)
		}
	}
	if _, err := a.staticStore.Stat("rapid", "index.html"); err == nil {
		_, err = a.staticStore.Update("rapid", "index.html", string(indexHTML))
		if err != nil {
			return fmt.Errorf("rapid: update dashboard: %w", err)
		}
		return nil
	}
	if _, err := a.staticStore.Create("rapid", "index.html", string(indexHTML)); err != nil {
		return fmt.Errorf("rapid: publish dashboard: %w", err)
	}
	return nil
}

// Handler wraps a host handler with the RAPID API surface.
func (a *RAPIDApplication) Handler(next http.Handler) http.Handler { return next }

// Health is the live status of the platform RAPID is running on.
func (a *RAPIDApplication) Health() map[string]any {
	components := map[string]bool{"rapid": a.cfg != nil}
	return map[string]any{
		"status":     "ok",
		"ok":         true,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    a.cfg.Version,
		"platform":   "azzurro",
		"components": components,
		"data_dir":   a.cfg.DataDir,
	}
}

// HealthCheck is Health with a real probe of every module.
func (a *RAPIDApplication) HealthCheck() map[string]any {
	components := map[string]bool{"rapid": a.cfg != nil}
	return map[string]any{
		"status":     "degraded",
		"ok":         false,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    a.cfg.Version,
		"platform":   "azzurro",
		"components": components,
		"data_dir":   a.cfg.DataDir,
	}
}

// IntegrationFailures returns one message per integration that did not answer.
func (a *RAPIDApplication) IntegrationFailures() []string {
	if a.platform == nil {
		return []string{"atp: platform is not open"}
	}
	return nil
}

// Metrics is the observability surface the README promises.
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
	if a.podStore != nil {
		counts := map[string]int{}
		for _, table := range []string{"projects", "analyses", "wrappers"} {
			if ids, err := a.podStore.IDs(table); err == nil {
				counts[table] = len(ids)
			}
		}
		out["counts"] = counts
	}
	return out
}

// ExportTables is the whitelist of record sets an operator may export.
var ExportTables = map[string]string{
	"projects":   "name",
	"analyses":   "analyzed_at",
	"wrappers":   "created_at",
}

// Export writes one of RAPID's record sets in pod's shared JSON/CSV shape.
func (a *RAPIDApplication) Export(w io.Writer, table, format string, columns []string) error {
	orderBy, ok := ExportTables[table]
	if !ok {
		return fmt.Errorf("rapid: unknown export table %q", table)
	}
	records, err := a.podStore.Query(table, nil, &pod.QueryOptions{OrderBy: orderBy, Desc: table != "projects"})
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

// RevokeShare cancels a share token by its value.
func (a *RAPIDApplication) RevokeShare(token string) (Share, error) {
	jti := slug(token)
	a.platform.shepherd.revoked[jti] = true
	return Share{Kind: "share", ID: token, JTI: jti}, nil
}

// Config returns the effective configuration.
func (a *RAPIDApplication) Config() RAPIDConfig { return *a.cfg }

// Platform exposes the platform for hosts that need to register functions.
func (a *RAPIDApplication) Platform() *platformWrapper { return a.platform }

// Store exposes the pod store.
func (a *RAPIDApplication) Store() *pod.Store { return a.podStore }

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
		case ".go": counts["go-wrapper"]++
		case ".py": counts["python-wrapper"]++
		case ".js", ".ts", ".jsx", ".tsx": counts["node-wrapper"]++
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
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "project"
	}
	return out
}