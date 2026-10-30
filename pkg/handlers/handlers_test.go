package handlers_test

// These tests exercise the HTTP surface RAPID exposes: the routes are registered
// and the responses carry the platform's conventions (attachment filename,
// content type, JSON error bodies). They use the real application on a
// temporary data root, so nothing is mocked.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"rapid.ai/pkg/handlers"

	rapid "rapid.ai"
)

func newAPI(t *testing.T) (*rapid.RAPIDApplication, http.Handler) {
	t.Helper()
	root := t.TempDir()
	cfg := rapid.DefaultConfig()
	cfg.DataDir = filepath.Join(root, "data")
	cfg.OutputDir = filepath.Join(root, "wrappers")
	cfg.TemplateDir = filepath.Join(root, "templates")
	cfg.Secret = "0123456789abcdef0123456789abcdef"
	cfg.Port = 18090
	app, err := rapid.NewRAPIDApplication(cfg)
	if err != nil {
		t.Fatalf("NewRAPIDApplication: %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })
	mux := http.NewServeMux()
	handlers.New(app).Register(mux)
	return app, mux
}

func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealthAndMetricsRoutes(t *testing.T) {
	_, h := newAPI(t)

	rec := do(t, h, http.MethodGet, "/api/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d", rec.Code)
	}
	var health map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("health is not JSON: %v", err)
	}
	components, ok := health["components"].(map[string]any)
	if !ok {
		t.Fatalf("health components = %#v", health["components"])
	}
	for _, name := range []string{"pod", "song", "shepherd", "secrets", "registry"} {
		if _, ok := components[name]; !ok {
			t.Errorf("health is missing component %q", name)
		}
	}

	probe := do(t, h, http.MethodGet, "/api/health?check=1", "")
	if probe.Code != http.StatusOK || !strings.Contains(probe.Body.String(), `"module_report"`) {
		t.Errorf("probe health = %d %s", probe.Code, probe.Body.String())
	}

	metrics := do(t, h, http.MethodGet, "/api/metrics", "")
	if metrics.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", metrics.Code)
	}
	var m map[string]any
	if err := json.Unmarshal(metrics.Body.Bytes(), &m); err != nil {
		t.Fatalf("metrics is not JSON: %v", err)
	}
	if _, ok := m["uptime_seconds"]; !ok {
		t.Error("metrics is missing uptime_seconds")
	}
	if counts, ok := m["counts"].(map[string]any); !ok || counts["projects"] == nil {
		t.Errorf("metrics counts = %#v", m["counts"])
	}
}

func TestExportRouteDownloadsAFile(t *testing.T) {
	_, h := newAPI(t)

	rec := do(t, h, http.MethodGet, "/api/export?table=projects&format=csv", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("export status = %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("content type = %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "rapid-projects.csv") {
		t.Errorf("content disposition = %q", cd)
	}
	if !strings.HasPrefix(rec.Body.String(), "\ufeff") {
		t.Error("CSV download is missing pod's BOM")
	}

	bad := do(t, h, http.MethodGet, "/api/export?table=nope", "")
	if bad.Code != http.StatusBadRequest {
		t.Errorf("unknown table status = %d, want 400", bad.Code)
	}
	if bad := do(t, h, http.MethodGet, "/api/export?format=xml", ""); bad.Code != http.StatusBadRequest {
		t.Errorf("unknown format status = %d, want 400", bad.Code)
	}
}

func TestShareAndRevokeRoutes(t *testing.T) {
	app, h := newAPI(t)

	rec := do(t, h, http.MethodPost, "/api/share", `{"kind":"analysis","id":"a-1","ttl":"1h"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("share status = %d: %s", rec.Code, rec.Body.String())
	}
	var share struct {
		Token string `json:"token"`
		JTI   string `json:"jti"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &share); err != nil {
		t.Fatal(err)
	}
	if share.Token == "" || share.JTI == "" {
		t.Fatalf("share response = %s", rec.Body.String())
	}

	revoke := do(t, h, http.MethodPost, "/api/share/revoke", `{"token":"`+share.Token+`"}`)
	if revoke.Code != http.StatusOK {
		t.Fatalf("revoke status = %d: %s", revoke.Code, revoke.Body.String())
	}
	if !strings.Contains(revoke.Body.String(), `"revoked":true`) {
		t.Errorf("revoke body = %s", revoke.Body.String())
	}
	if !app.Platform().Shepherd().IsDenied(share.JTI) {
		t.Error("revoked token id is not denied")
	}

	if bad := do(t, h, http.MethodPost, "/api/share/revoke", `{"token":"not-a-token"}`); bad.Code != http.StatusBadRequest {
		t.Errorf("invalid token status = %d, want 400", bad.Code)
	}
}
