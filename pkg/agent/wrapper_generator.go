package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"rapid.ai/pkg/pricing"
)

// CostLogger is the slice of the pricing engine the generator needs. The real
// implementation is pricing.PriceLogger; a nil logger simply records nothing.
type CostLogger interface {
	LogTransaction(cost pricing.CostBreakdown)
}

// WrapperGenerator renders one business customization wrapper from a template
// and writes it, plus its metadata, into the output directory.
type WrapperGenerator struct {
	templateDir string
	logger      CostLogger
	calc        *pricing.PricingCalculator

	mu    sync.Mutex
	cache map[string]*template.Template
}

// BusinessWrapper is a generated wrapper and everything known about it.
type BusinessWrapper struct {
	ProjectName  string          `json:"project_name"`
	TemplateType string          `json:"template_type"`
	Timestamp    time.Time       `json:"timestamp"`
	Content      string          `json:"content"`
	FilePath     string          `json:"file_path"`
	Metadata     WrapperMetadata `json:"metadata"`
	Pricing      WrapperPricing  `json:"pricing"`
}

// WrapperMetadata describes the generated artefact.
type WrapperMetadata struct {
	Version           string   `json:"version"`
	Author            string   `json:"author"`
	Description       string   `json:"description"`
	License           string   `json:"license"`
	Dependencies      []string `json:"dependencies"`
	IntegrationPoints []string `json:"integration_points"`
}

// WrapperPricing records what the generation cost.
type WrapperPricing struct {
	TokensUsed int       `json:"tokens_used"`
	Cost       float64   `json:"cost"`
	Currency   string    `json:"currency"`
	CacheHit   bool      `json:"cache_hit"`
	Enterprise bool      `json:"enterprise"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewWrapperGenerator builds a generator. A nil logger is allowed.
func NewWrapperGenerator(templateDir string, logger CostLogger) *WrapperGenerator {
	if templateDir == "" {
		templateDir = "./templates"
	}
	return &WrapperGenerator{
		templateDir: templateDir,
		logger:      logger,
		calc:        pricing.NewPricingCalculator(false),
		cache:       make(map[string]*template.Template),
	}
}

// GenerateWrapper renders the template and writes the source and metadata.
func (wg *WrapperGenerator) GenerateWrapper(projectName, templateType, outputDir string, enterprise bool) (*BusinessWrapper, error) {
	if strings.TrimSpace(projectName) == "" {
		return nil, fmt.Errorf("agent: project name is required")
	}
	if templateType == "" {
		templateType = "go-wrapper"
	}
	if outputDir == "" {
		outputDir = "./generated_wrappers"
	}

	tmpl, err := wg.loadTemplate(templateType)
	if err != nil {
		return nil, fmt.Errorf("load template: %w", err)
	}

	info := ProjectInfo{
		Name:      projectName,
		Type:      inferProjectType(projectName),
		Language:  inferLanguage(projectName),
		Timestamp: time.Now().UTC(),
		Author:    "RAPID AI",
		Version:   "1.0.0",
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, info); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	content := buf.String()

	base := fmt.Sprintf("%s_%s_%s", safeFileName(projectName), safeFileName(templateType), time.Now().Format("20060102"))
	filePath := filepath.Join(outputDir, base+wrapperExtension(templateType))
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("write wrapper: %w", err)
	}

	metadata := WrapperMetadata{
		Version:           info.Version,
		Author:            info.Author,
		Description:       wg.generateDescription(info),
		License:           "MIT",
		Dependencies:      wg.getDependencies(templateType),
		IntegrationPoints: wg.getIntegrationPoints(info),
	}
	metaJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode metadata: %w", err)
	}
	if err := os.WriteFile(filePath+".metadata.json", metaJSON, 0o644); err != nil {
		return nil, fmt.Errorf("write metadata: %w", err)
	}

	tokens := len(content) / 4
	breakdown := wg.calc.CalculateCost(tokens, false, "wrapper", enterprise, time.Now().UTC())
	if wg.logger != nil {
		wg.logger.LogTransaction(breakdown)
	}

	return &BusinessWrapper{
		ProjectName:  projectName,
		TemplateType: templateType,
		Timestamp:    info.Timestamp,
		Content:      content,
		FilePath:     filePath,
		Metadata:     metadata,
		Pricing: WrapperPricing{
			TokensUsed: breakdown.Tokens,
			Cost:       breakdown.TotalCost,
			Currency:   breakdown.Currency,
			CacheHit:   breakdown.CacheHit,
			Enterprise: breakdown.Enterprise,
			Timestamp:  breakdown.Timestamp,
		},
	}, nil
}

func safeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, name)
	return strings.Trim(name, "-")
}

func wrapperExtension(templateType string) string {
	switch templateType {
	case "go-wrapper":
		return ".go"
	case "python-wrapper":
		return ".py"
	case "node-wrapper":
		return ".js"
	case "terraform-wrapper":
		return ".tf"
	case "dockerfile-wrapper":
		return ".Dockerfile"
	default:
		return ".txt"
	}
}

// loadTemplate prefers a file in templateDir, and falls back to the built-in
// template for the type. Parsed templates are cached per type.
func (wg *WrapperGenerator) loadTemplate(templateType string) (*template.Template, error) {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	if tmpl, ok := wg.cache[templateType]; ok {
		return tmpl, nil
	}

	path := filepath.Join(wg.templateDir, templateType+".template")
	if data, err := os.ReadFile(path); err == nil {
		tmpl, err := parseTemplate(templateType, string(data))
		if err != nil {
			return nil, err
		}
		wg.cache[templateType] = tmpl
		return tmpl, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	tmpl, err := parseTemplate(templateType, builtinTemplate(templateType))
	if err != nil {
		return nil, err
	}
	wg.cache[templateType] = tmpl
	return tmpl, nil
}

func parseTemplate(name, body string) (*template.Template, error) {
	funcs := template.FuncMap{"lower": strings.ToLower, "upper": strings.ToUpper}
	return template.New(name).Funcs(funcs).Parse(body)
}

func builtinTemplate(templateType string) string {
	switch templateType {
	case "go-wrapper":
		return goWrapperTemplate
	case "python-wrapper":
		return pythonWrapperTemplate
	case "node-wrapper":
		return nodeWrapperTemplate
	case "terraform-wrapper":
		return terraformWrapperTemplate
	case "dockerfile-wrapper":
		return dockerfileWrapperTemplate
	default:
		return genericWrapperTemplate
	}
}

func inferProjectType(projectName string) string {
	n := strings.ToLower(projectName)
	switch {
	case strings.Contains(n, "docker"):
		return "docker"
	case strings.Contains(n, "api"):
		return "api"
	case strings.Contains(n, "web"):
		return "web"
	case strings.Contains(n, "mobile"):
		return "mobile"
	default:
		return "generic"
	}
}

func inferLanguage(projectName string) string {
	n := strings.ToLower(projectName)
	switch {
	case strings.Contains(n, "python"), strings.Contains(n, "ml"), strings.Contains(n, "ai"):
		return "Python"
	case strings.Contains(n, "javascript"), strings.Contains(n, "node"), strings.Contains(n, "ts"):
		return "JavaScript"
	case strings.Contains(n, "go"):
		return "Go"
	case strings.Contains(n, "java"):
		return "Java"
	default:
		return "Go"
	}
}

func (wg *WrapperGenerator) generateDescription(info ProjectInfo) string {
	return fmt.Sprintf("Auto-generated %s wrapper for %s via the RAPID analysis system.",
		info.Language, info.Name)
}

func (wg *WrapperGenerator) getDependencies(templateType string) []string {
	deps := []string{
		"azzurrotech/atp",
		"azzurrotech/pod",
		"azzurrotech/shepherd",
	}
	switch templateType {
	case "python-wrapper":
		deps = append(deps, "requests", "flask")
	case "node-wrapper":
		deps = append(deps, "express")
	case "terraform-wrapper":
		deps = append(deps, "hashicorp/aws")
	case "dockerfile-wrapper":
		deps = append(deps, "docker")
	}
	return deps
}

func (wg *WrapperGenerator) getIntegrationPoints(info ProjectInfo) []string {
	points := []string{
		"POD record storage",
		"SHEPHERD capability tokens",
		"SONG static hosting",
		"ATP function registration",
	}
	switch info.Language {
	case "Go":
		points = append(points, "Go module integration", "Container image support")
	case "Python":
		points = append(points, "Data-processing pipelines", "Model serving")
	case "JavaScript":
		points = append(points, "Web server integration", "API gateway support")
	}
	return points
}

// --- built-in templates ----------------------------------------------------

var goWrapperTemplate = `// Code generated by RAPID. DO NOT EDIT.
// Wrapper for {{ .Name }} ({{ .Language }}), generated {{ .Timestamp.Format "2006-01-02" }}.

package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type BusinessWrapper struct {
	ProjectName string
	CreatedAt   time.Time
	Version     string
}

func NewBusinessWrapper() *BusinessWrapper {
	return &BusinessWrapper{
		ProjectName: "{{ .Name }}",
		CreatedAt:   time.Now(),
		Version:     "{{ .Version }}",
	}
}

func (bw *BusinessWrapper) HandleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":      "success",
		"project":     bw.ProjectName,
		"platform":    "azzurro",
		"timestamp":   time.Now(),
		"version":     bw.Version,
		"integration": "atp,pod,shepherd,song",
	})
}

func main() {
	http.HandleFunc("/", NewBusinessWrapper().HandleRequest)
	http.ListenAndServe(":8080", nil)
}
`

var pythonWrapperTemplate = `#!/usr/bin/env python3
"""Wrapper for {{ .Name }} ({{ .Language }}), generated by RAPID on {{ .Timestamp.Format "2006-01-02" }}."""

from datetime import datetime

from flask import Flask, jsonify


class BusinessWrapper:
    def __init__(self):
        self.project_name = "{{ .Name }}"
        self.created_at = datetime.now()
        self.version = "{{ .Version }}"

    def handle_request(self):
        return {
            "status": "success",
            "project": self.project_name,
            "platform": "azzurro",
            "timestamp": datetime.now().isoformat(),
            "version": self.version,
            "integration": "atp,pod,shepherd,song",
        }


app = Flask(__name__)
wrapper = BusinessWrapper()


@app.route("/")
def index():
    return jsonify(wrapper.handle_request())


@app.route("/health")
def health():
    return jsonify({"status": "healthy", "timestamp": datetime.now().isoformat()})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
`

var nodeWrapperTemplate = `// Wrapper for {{ .Name }} ({{ .Language }}), generated by RAPID on {{ .Timestamp.Format "2006-01-02" }}.

const express = require("express");
const app = express();

class BusinessWrapper {
  constructor() {
    this.projectName = "{{ .Name }}";
    this.createdAt = new Date();
    this.version = "{{ .Version }}";
  }

  handleRequest() {
    return {
      status: "success",
      project: this.projectName,
      platform: "azzurro",
      timestamp: this.createdAt.toISOString(),
      version: this.version,
      integration: "atp,pod,shepherd,song",
    };
  }
}

const wrapper = new BusinessWrapper();
app.get("/", (req, res) => res.json(wrapper.handleRequest()));
app.get("/health", (req, res) => res.json({ status: "healthy", timestamp: new Date().toISOString() }));

const PORT = process.env.PORT || 8080;
app.listen(PORT, () => console.log("Business wrapper listening on " + PORT));

module.exports = { BusinessWrapper, app };
`

var terraformWrapperTemplate = `# Wrapper for {{ .Name }} ({{ .Language }}), generated by RAPID on {{ .Timestamp.Format "2006-01-02" }}.

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  description = "AWS region for deployment"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "{{ .Name }}"
}

resource "aws_security_group" "wrapper_sg" {
  name        = "${var.project_name}-sg"
  description = "Security group for the business wrapper"

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

output "project_info" {
  value = {
    name       = var.project_name
    created_by = "{{ .Author }}"
    created_at = "{{ .Timestamp.Format "2006-01-02" }}"
    version    = "{{ .Version }}"
  }
}
`

var dockerfileWrapperTemplate = `# Wrapper for {{ .Name }} ({{ .Language }}), generated by RAPID on {{ .Timestamp.Format "2006-01-02" }}.
FROM golang:1.22-alpine

WORKDIR /app/{{ lower .Name }}

RUN apk add --no-cache ca-certificates git

COPY . /app/{{ lower .Name }}

RUN go mod download && go build -o /app/{{ lower .Name }}/business-wrapper .

RUN adduser -D appuser
USER appuser

ENV APP_NAME="{{ .Name }}"
ENV APP_VERSION="{{ .Version }}"
ENV PLATFORM="azzurro"
ENV INTEGRATION="atp,pod,shepherd,song"

EXPOSE 8080

ENTRYPOINT ["/app/{{ lower .Name }}/business-wrapper"]
`

var genericWrapperTemplate = `# Wrapper for {{ .Name }} ({{ .Language }}), generated by RAPID on {{ .Timestamp.Format "2006-01-02 15:04:05" }}.

package main

import "fmt"

func main() {
	fmt.Println("Business wrapper for {{ .Name }} ({{ .Version }})")
	fmt.Println("Platform: azzurro")
}
`
