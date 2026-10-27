package main

import (
    "context"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
)

// Handler functions for API endpoints
type ProjectAnalysisResult struct {
    Success bool `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string `json:"error,omitempty"`
}

// HandleAPIError writes an error response
type Handler func(http.ResponseWriter, *http.Request) ProjectAnalysisResult

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    result := h(w, r)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    if err := json.NewEncoder(w).Encode(result); err != nil {
        log.Printf("Error encoding response: %v", err)
    }
}

// APIHandler handles the main /api/projects endpoint
type APIProjectsHandler struct {
    app *RAPIDApplication
}

func NewAPIProjectsHandler(app *RAPIDApplication) *APIProjectsHandler {
    return &APIProjectsHandler{app: app}
}

func (h *APIProjectsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        h.handleGetProjects(w, r)
    case http.MethodPost:
        h.handleAnalyzeProject(w, r)
    case http.MethodOptions:
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        w.WriteHeader(http.StatusOK)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

func (h *APIProjectsHandler) handleGetProjects(w http.ResponseWriter, r *http.Request) {
    projects := []AIProject{
        {
            Name:        "Go",
            Description: "Open source programming language",
            URL:         "https://golang.org",
            Repository:  "https://github.com/golang/go",
            Lang:        "Go",
            Tags:        []string{"language", "programming"},
            License:     "BSD",
            Dependencies: []string{},
            Status:      "active",
        },
        {
            Name:        "Docker",
            Description: "Containerization platform",
            URL:         "https://docker.com",
            Repository:  "https://github.com/docker/docker",
            Lang:        "Go",
            Tags:        []string{"containers", "docker"},
            License:     "Apache 2.0",
            Dependencies: []string{},
            Status:      "active",
        },
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(projects)
}

func (h *APIProjectsHandler) handleAnalyzeProject(w http.ResponseWriter, r *http.Request) {
    var project AIProject
    if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    result := h.app.analyzeProject(project)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

// IntegrationAnalysisHandler handles integration analysis
type IntegrationAnalysisHandler struct {
    app *RAPIDApplication
}

func NewIntegrationAnalysisHandler(app *RAPIDApplication) *IntegrationAnalysisHandler {
    return &IntegrationAnalysisHandler{app: app}
}

func (h *IntegrationAnalysisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    testResults := h.app.runIntegrationTests()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(testResults)
}