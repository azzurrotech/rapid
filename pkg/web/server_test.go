package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rapid "rapid.ai"
)

func testServer(t *testing.T) (http.Handler, *rapid.RAPIDApplication, string) {
	t.Helper()
	root := t.TempDir()
	cfg := rapid.DefaultConfig()
	cfg.DataDir = filepath.Join(root, "data")
	cfg.OutputDir = filepath.Join(root, "wrappers")
	cfg.TemplateDir = filepath.Join(root, "templates")
	cfg.Secret = "0123456789abcdef0123456789abcdef"
	cfg.Port = 18082

	app, err := rapid.NewRAPIDApplication(cfg)
	if err != nil {
		t.Fatalf("NewRAPIDApplication: %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })

	h, err := NewWebServer(app, "localhost", "18082").Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	src := filepath.Join(root, "project")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return h, app, src
}

func request(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestDashboardIsServedBySong(t *testing.T) {
	h, _, _ := testServer(t)

	rec := request(t, h, http.MethodGet, "/rapid/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /rapid/ = %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "html") {
		t.Errorf("dashboard content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "RAPID") {
		t.Error("dashboard does not identify itself")
	}

	// "/" points at the one hosted copy rather than a second document.
	rec = request(t, h, http.MethodGet, "/", "")
	if rec.Code != http.StatusFound {
		t.Fatalf("GET / = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/rapid/" {
		t.Errorf("Location = %q", loc)
	}
}

func TestHealthReportsTheRealComponents(t *testing.T) {
	h, _, _ := testServer(t)

	rec := request(t, h, http.MethodGet, "/api/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health = %d", rec.Code)
	}
	var health struct {
		Status     string          `json:"status"`
		Components map[string]bool `json:"components"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("health is not JSON: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("status = %q", health.Status)
	}
	for _, component := range []string{"atp", "pod", "shepherd", "song"} {
		if !health.Components[component] {
			t.Errorf("component %s is down: %+v", component, health.Components)
		}
	}
}

func TestProjectsAndAnalysisThroughHTTP(t *testing.T) {
	h, app, src := testServer(t)

	rec := request(t, h, http.MethodGet, "/api/projects", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("projects = %d", rec.Code)
	}
	var projects []rapid.AIProject
	if err := json.Unmarshal(rec.Body.Bytes(), &projects); err != nil {
		t.Fatal(err)
	}
	if len(projects) == 0 {
		t.Fatal("catalog is empty")
	}

	rec = request(t, h, http.MethodPost, "/api/projects/analyze", `{"path":`+quote(src)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("analyze = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"total_files"`) {
		t.Errorf("analysis response: %s", rec.Body.String())
	}

	// The analysis really landed in pod, not just in the response.
	if got := len(app.Analyses()); got != 1 {
		t.Errorf("recorded analyses = %d", got)
	}
}

func TestWrapperAndShareEndpoints(t *testing.T) {
	h, app, src := testServer(t)

	rec := request(t, h, http.MethodPost, "/api/wrappers", `{"path":`+quote(src)+`,"template_type":"go-wrapper"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate = %d: %s", rec.Code, rec.Body.String())
	}
	if got := len(app.Wrappers()); got != 1 {
		t.Errorf("recorded wrappers = %d", got)
	}

	rec = request(t, h, http.MethodPost, "/api/share", `{"kind":"wrapper","id":"abc","ttl":"1h"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("share = %d: %s", rec.Code, rec.Body.String())
	}
	var share struct {
		Token string `json:"token"`
		Kind  string `json:"kind"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &share); err != nil {
		t.Fatal(err)
	}
	if share.Token == "" || share.Kind != "wrapper" {
		t.Errorf("share = %+v", share)
	}
	if _, err := app.Platform().Shepherd().Manager().Verify(share.Token); err != nil {
		t.Errorf("token does not verify: %v", err)
	}
}

func TestAPINeverFallsThroughToTheDashboard(t *testing.T) {
	h, _, _ := testServer(t)

	rec := request(t, h, http.MethodGet, "/api/nonsense", "")
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "<html") {
		t.Fatal("unknown API path was answered with the dashboard")
	}
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestDashboardOffersMetricsExportsAndRevoke(t *testing.T) {
	h, _, _ := testServer(t)

	rec := request(t, h, http.MethodGet, "/rapid/", "")
	body := rec.Body.String()
	for _, want := range []string{"Metrics", "Exports", "Shares", "/api/export?table=projects&amp;format=csv", "/api/metrics", "/api/share/revoke"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard is missing %q", want)
		}
	}
	if !strings.Contains(body, "/_song/viewkit.js") {
		t.Error("dashboard does not load the shared viewkit")
	}
}

func TestExportAndRevokeThroughThePlatform(t *testing.T) {
	h, app, _ := testServer(t)

	rec := request(t, h, http.MethodGet, "/api/export?table=projects&format=json", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("export = %d: %s", rec.Code, rec.Body.String())
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "rapid-projects.json") {
		t.Errorf("Content-Disposition = %q", cd)
	}

	rec = request(t, h, http.MethodPost, "/api/share", `{"kind":"project","id":"x","ttl":"1h"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("share = %d", rec.Code)
	}
	var share struct {
		Token string `json:"token"`
		JTI   string `json:"jti"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &share); err != nil {
		t.Fatal(err)
	}
	rec = request(t, h, http.MethodPost, "/api/share/revoke", `{"token":`+quote(share.Token)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke = %d: %s", rec.Code, rec.Body.String())
	}
	if !app.Platform().Shepherd().IsDenied(share.JTI) {
		t.Error("revoked token id is not denied by shepherd")
	}

	rec = request(t, h, http.MethodGet, "/api/metrics", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "uptime_seconds") {
		t.Errorf("metrics = %d: %s", rec.Code, rec.Body.String())
	}
}
