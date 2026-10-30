package rapid

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testApp builds an application on a temporary data root and a temporary
// output directory, so a test never touches the developer's real platform.
func testApp(t *testing.T) *RAPIDApplication {
	t.Helper()
	root := t.TempDir()
	cfg := DefaultConfig()
	cfg.DataDir = filepath.Join(root, "data")
	cfg.OutputDir = filepath.Join(root, "wrappers")
	cfg.TemplateDir = filepath.Join(root, "templates")
	cfg.Secret = "0123456789abcdef0123456789abcdef"
	cfg.Port = 18080
	app, err := NewRAPIDApplication(cfg)
	if err != nil {
		t.Fatalf("NewRAPIDApplication: %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })
	return app
}

// testProject writes a small but realistic Go project.
func testProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/project\n\ngo 1.22\n")
	write("main.go", "package main\n\nfunc main() {}\n")
	write("main_test.go", "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n")
	write("README.md", "# example\n\nA small project.\n")
	write(".github/workflows/ci.yml", "name: ci\non: [push]\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: go test ./...\n")
	return dir
}

func TestPlatformIsTheRealATP(t *testing.T) {
	app := testApp(t)

	if app.Platform() == nil {
		t.Fatal("no atp platform")
	}
	if app.Platform().Pod() == nil || app.Store() != app.Platform().Pod() {
		t.Error("RAPID is not using atp's mounted pod store")
	}
	if app.Platform().Song() == nil || app.Platform().Song().Store() == nil {
		t.Error("RAPID is not using atp's mounted song host")
	}
	if app.Platform().Shepherd() == nil || app.Platform().Shepherd().Manager() == nil {
		t.Error("RAPID is not using atp's mounted shepherd")
	}
	if app.Platform().Secrets() == nil {
		t.Error("atp vault is not mounted")
	}
	if failures := app.IntegrationFailures(); len(failures) != 0 {
		t.Fatalf("integrations failed: %v", failures)
	}
	if !app.ValidateIntegrations() {
		t.Error("ValidateIntegrations = false")
	}
	if err := app.ValidateConfiguration(); err != nil {
		t.Errorf("ValidateConfiguration: %v", err)
	}
}

func TestProjectsAreStoredInPod(t *testing.T) {
	app := testApp(t)

	projects := app.HandleGetProjects()
	if len(projects) < 2 {
		t.Fatalf("seeded catalog has %d projects, want at least 2", len(projects))
	}
	if err := app.StoreProject(AIProject{
		Name:        "Example",
		Description: "an example",
		Language:    "Go",
		Status:      "active",
		Tags:        []string{"test"},
	}); err != nil {
		t.Fatalf("StoreProject: %v", err)
	}

	// The same record must be readable through pod directly, not from a cache.
	var stored AIProject
	if _, err := app.Store().GetStruct("projects", "example", &stored); err != nil {
		t.Fatalf("pod GetStruct: %v", err)
	}
	if stored.Name != "Example" || stored.Language != "Go" {
		t.Errorf("stored project = %+v", stored)
	}
	if len(stored.Tags) != 1 || stored.Tags[0] != "test" {
		t.Errorf("stored tags = %v", stored.Tags)
	}
}

func TestAnalyzeProjectPersistsAnAnalysisRecord(t *testing.T) {
	app := testApp(t)
	dir := testProject(t)

	result, err := app.AnalyzeProject(dir)
	if err != nil {
		t.Fatalf("AnalyzeProject: %v", err)
	}
	if result.TotalFiles == 0 {
		t.Fatal("analysis scanned no files")
	}
	if !result.HasTests {
		t.Error("analysis did not notice the test file")
	}
	if !result.HasCI {
		t.Error("analysis did not notice the CI workflow")
	}
	if result.ProjectName != filepath.Base(dir) {
		t.Errorf("project name = %q", result.ProjectName)
	}
	if len(result.Recommendations) == 0 {
		t.Error("no recommendations")
	}
	if result.Pricing.EstimatedTokens == 0 {
		t.Error("no pricing estimate")
	}

	analyses := app.Analyses()
	if len(analyses) != 1 {
		t.Fatalf("recorded %d analyses, want 1", len(analyses))
	}
	if analyses[0].ProjectPath != dir {
		t.Errorf("recorded path = %q", analyses[0].ProjectPath)
	}
	if !strings.Contains(analyses[0].Result, `"project_name"`) {
		t.Error("recorded analysis does not carry the full result")
	}
}

func TestGenerateWrapperWritesAndRecords(t *testing.T) {
	app := testApp(t)
	dir := testProject(t)

	wrapper, err := app.GenerateWrapper(dir)
	if err != nil {
		t.Fatalf("GenerateWrapper: %v", err)
	}
	if wrapper.TemplateType != "go-wrapper" {
		t.Errorf("template = %q, want go-wrapper", wrapper.TemplateType)
	}
	data, err := os.ReadFile(wrapper.FilePath)
	if err != nil {
		t.Fatalf("read wrapper: %v", err)
	}
	if !strings.Contains(string(data), "package main") {
		t.Error("generated Go wrapper has no package clause")
	}
	if _, err := os.Stat(wrapper.FilePath + ".metadata.json"); err != nil {
		t.Errorf("metadata file missing: %v", err)
	}

	wrappers := app.Wrappers()
	if len(wrappers) != 1 {
		t.Fatalf("recorded %d wrappers, want 1", len(wrappers))
	}
	if wrappers[0].FilePath != wrapper.FilePath {
		t.Errorf("recorded wrapper path = %q", wrappers[0].FilePath)
	}
}

func TestShareIsAShepherdCapability(t *testing.T) {
	app := testApp(t)

	share, err := app.Share("analysis", "abc123", time.Hour)
	if err != nil {
		t.Fatalf("Share: %v", err)
	}
	if share.Token == "" {
		t.Fatal("share has no token")
	}
	claims, err := app.Platform().Shepherd().Manager().Verify(share.Token)
	if err != nil {
		t.Fatalf("shepherd rejected its own token: %v", err)
	}
	if claims.Meta["kind"] != "analysis" || claims.Meta["target"] != "abc123" {
		t.Errorf("token metadata = %v", claims.Meta)
	}
}

func TestDashboardIsHostedBySong(t *testing.T) {
	app := testApp(t)
	body := []byte("<!doctype html><title>RAPID test</title>")

	if err := app.PublishDashboard(body); err != nil {
		t.Fatalf("PublishDashboard: %v", err)
	}
	entry, err := app.Platform().Song().Store().Read("rapid", "index.html", true)
	if err != nil {
		t.Fatalf("read hosted dashboard: %v", err)
	}
	if string(entry.Content) != string(body) {
		t.Errorf("hosted dashboard = %q", entry.Content)
	}

	// Publishing twice updates in place rather than failing.
	if err := app.PublishDashboard([]byte("second")); err != nil {
		t.Fatalf("republish: %v", err)
	}
	entry, err = app.Platform().Song().Store().Read("rapid", "index.html", true)
	if err != nil {
		t.Fatalf("read republished dashboard: %v", err)
	}
	if string(entry.Content) != "second" {
		t.Errorf("republished dashboard = %q", entry.Content)
	}
}

func TestConfigRequiresARealSecret(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DataDir = t.TempDir()
	cfg.Secret = "short"
	cfg.Port = 18081
	if _, err := NewRAPIDApplication(cfg); err == nil {
		t.Fatal("a short secret was accepted")
	}

	cfg.Secret = ""
	app, err := NewRAPIDApplication(cfg)
	if err != nil {
		t.Fatalf("generated secret was not accepted: %v", err)
	}
	defer app.Close()
	if len(app.Config().Secret) < 32 {
		t.Errorf("generated secret is %d bytes", len(app.Config().Secret))
	}
}

func TestExportWritesPodJSONAndCSV(t *testing.T) {
	app := testApp(t)
	if err := app.StoreProject(AIProject{Name: "Exported", Description: "a project", Language: "Go", Status: "active"}); err != nil {
		t.Fatal(err)
	}

	var jsonOut strings.Builder
	if err := app.Export(&jsonOut, "projects", "json", []string{"name", "language"}); err != nil {
		t.Fatalf("export json: %v", err)
	}
	if !strings.Contains(jsonOut.String(), `"name":"Exported"`) {
		t.Errorf("JSON export = %s", jsonOut.String())
	}

	var csvOut strings.Builder
	if err := app.Export(&csvOut, "projects", "csv", []string{"name", "language"}); err != nil {
		t.Fatalf("export csv: %v", err)
	}
	if !strings.HasPrefix(csvOut.String(), "\ufeff") {
		t.Error("CSV export is missing pod's UTF-8 BOM")
	}
	if !strings.Contains(csvOut.String(), "name,language") {
		t.Errorf("CSV header = %q", csvOut.String())
	}

	if err := app.Export(&strings.Builder{}, "secrets", "json", nil); err == nil {
		t.Error("export accepted a table outside the whitelist")
	}
}

func TestRevokeShareCancelsTheToken(t *testing.T) {
	app := testApp(t)
	share, err := app.Share("analysis", "abc123", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if share.JTI == "" {
		t.Fatal("share has no token id, so it cannot be revoked")
	}
	target, err := app.RevokeShare(share.Token)
	if err != nil {
		t.Fatalf("RevokeShare: %v", err)
	}
	if target.Kind != "analysis" || target.ID != "abc123" {
		t.Errorf("revoked target = %#v", target)
	}
	// shepherd's deny list must now refuse the token it minted.
	if !app.Platform().Shepherd().IsDenied(share.JTI) {
		t.Error("the revoked token id is not in shepherd's deny list")
	}
}

func TestMetricsCountsAndUptime(t *testing.T) {
	app := testApp(t)
	m := app.Metrics()
	if n, ok := m["uptime_seconds"].(int64); !ok || n < 0 {
		t.Errorf("uptime_seconds = %v", m["uptime_seconds"])
	}
	counts, ok := m["counts"].(map[string]int)
	if !ok {
		t.Fatalf("counts = %#v", m["counts"])
	}
	if counts["projects"] < 2 {
		t.Errorf("projects count = %d, want the seeded catalog", counts["projects"])
	}
	for _, table := range []string{"analyses", "wrappers"} {
		if _, ok := counts[table]; !ok {
			t.Errorf("metrics is missing the %s count", table)
		}
	}
}
