// Package handlers exposes RAPID's JSON API. Every handler reads and writes
// through RAPIDApplication, which in turn uses the real atp/pod/song/shepherd
// modules, so the API cannot drift from what the platform actually stores.
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"azzurrotech/pod"
	"azzurrotech/shepherd/capability"

	rapid "rapid.ai"
)

// API serves the RAPID HTTP surface.
type API struct {
	app *rapid.RAPIDApplication
}

// New builds an API around an application.
func New(app *rapid.RAPIDApplication) *API { return &API{app: app} }

// Register mounts every RAPID endpoint on mux.
func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", a.Health)
	mux.HandleFunc("/api/projects", a.Projects)
	mux.HandleFunc("/api/projects/analyze", a.Analyze)
	mux.HandleFunc("/api/wrappers", a.Wrappers)
	mux.HandleFunc("/api/analyses", a.Analyses)
	mux.HandleFunc("/api/integrations", a.Integrations)
	mux.HandleFunc("/api/share", a.Share)
	mux.HandleFunc("/api/share/revoke", a.RevokeShare)
	mux.HandleFunc("/api/export", a.Export)
	mux.HandleFunc("/api/metrics", a.Metrics)
}

// Health reports the live platform components. It is a cheap read-only check;
// ?check=1 exercises every module with a real operation first.
func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodGet) {
		return
	}
	if r.URL.Query().Get("check") == "1" || r.URL.Query().Get("probe") == "1" {
		writeJSON(w, http.StatusOK, a.app.HealthCheck())
		return
	}
	writeJSON(w, http.StatusOK, a.app.Health())
}

// Projects lists the catalog, or adds a project with POST.
func (a *API) Projects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.app.HandleGetProjects())
	case http.MethodPost:
		var p rapid.AIProject
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusBadRequest, "invalid project: "+err.Error())
			return
		}
		if p.Name == "" {
			writeError(w, http.StatusBadRequest, "project name is required")
			return
		}
		if p.Status == "" {
			p.Status = "active"
		}
		if err := a.app.StoreProject(p); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, p)
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Analyze runs an analysis for the path in the request body.
func (a *API) Analyze(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodPost) {
		return
	}
	var p rapid.AIProject
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid project: "+err.Error())
		return
	}
	result, err := a.app.AnalyzeAIProject(p)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Wrappers lists generated wrappers, or generates one with POST.
func (a *API) Wrappers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.app.Wrappers())
	case http.MethodPost:
		var body struct {
			Path         string `json:"path"`
			TemplateType string `json:"template_type"`
			Enterprise   bool   `json:"enterprise"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}
		if body.Path == "" {
			writeError(w, http.StatusBadRequest, "path is required")
			return
		}
		wrapper, err := a.app.CreateWrapperWithOptions(body.Path, body.TemplateType, body.Enterprise)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, wrapper)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Analyses lists recorded analyses, newest first.
func (a *API) Analyses(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, a.app.Analyses())
}

// Integrations reports each module's live state.
func (a *API) Integrations(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodGet) {
		return
	}
	failures := a.app.IntegrationFailures()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       len(failures) == 0,
		"failures": failures,
	})
}

// Share issues a revocable shepherd capability token.
func (a *API) Share(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodPost) {
		return
	}
	var body struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
		TTL  string `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if body.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	ttl := capability.ParseTTL(body.TTL, 7*24*time.Hour)
	share, err := a.app.Share(body.Kind, body.ID, ttl)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, share)
}

// RevokeShare cancels a share token. The token value is the credential, so a
// caller can cancel a link without having kept the issue-time response.
func (a *API) RevokeShare(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodPost) {
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	// Accept the token from the JSON body or from ?token=, so the operation
	// works from both a form and a script. An empty body is not an error.
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if body.Token == "" {
		body.Token = r.URL.Query().Get("token")
	}
	if body.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	target, err := a.app.RevokeShare(body.Token)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"revoked": true,
		"kind":    target.Kind,
		"id":      target.ID,
	})
}

// Export downloads one of RAPID's record sets as JSON or CSV. The export is
// rendered into a buffer first: an unknown table or an unwritable store must
// return an error, not a truncated file with a success status.
func (a *API) Export(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodGet) {
		return
	}
	q := r.URL.Query()
	table := q.Get("table")
	if table == "" {
		table = "projects"
	}
	format := q.Get("format")
	if format == "" {
		format = pod.FormatJSON
	}
	if format != pod.FormatJSON && format != pod.FormatCSV {
		writeError(w, http.StatusBadRequest, "format must be json or csv")
		return
	}
	var columns []string
	if raw := q.Get("columns"); raw != "" {
		for _, name := range strings.Split(raw, ",") {
			if name = strings.TrimSpace(name); name != "" {
				columns = append(columns, name)
			}
		}
	}

	var buf bytes.Buffer
	if err := a.app.Export(&buf, table, format, columns); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", pod.ContentType(format))
	w.Header().Set("Content-Disposition", "attachment; filename="+pod.ExportFilename("rapid-"+table, format))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// Metrics reports uptime, record counts and pricing statistics.
func (a *API) Metrics(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, a.app.Metrics())
}

func allow(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	if r.Method != method {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
