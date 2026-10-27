package agent

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "text/template"
    "time"
)

// WrapperGenerator handles generation of business customization wrappers
type WrapperGenerator struct {
    templateDir    string
    customTemplates map[string]template.Template
    pricingLogger   PricingLogger
    maxTemplates    int
    templateCache   map[string]template.Template
}

// BusinessWrapper represents a generated business customization wrapper
 type BusinessWrapper struct {
     ProjectName    string
     TemplateType   string
     Timestamp      time.Time
     Content        string
     FilePath       string
     Metadata       WrapperMetadata
     Pricing        WrapperPricing
 }

// WrapperMetadata contains metadata about the generated wrapper
 type WrapperMetadata struct {
     Version         string
     Author          string
     Description     string
     License        string
     Dependencies    []string
     IntegrationPoints []string
 }

// WrapperPricing contains pricing information for the wrapper
 type WrapperPricing struct {
     TokensUsed      int
     Cost            float64
     Currency        string
     CacheHit        bool
     Enterprise      bool
     Timestamp       time.Time
 }

// NewWrapperGenerator creates a new wrapper generator
 func NewWrapperGenerator(templateDir string, pricingLogger PricingLogger) *WrapperGenerator {
     if templateDir == "" {
         templateDir = "./templates"
     }

     os.MkdirAll(templateDir, 0755)

     return &WrapperGenerator{
         templateDir:    templateDir,
         customTemplates: make(map[string]template.Template),
         pricingLogger:   pricingLogger,
         maxTemplates:    50,
         templateCache:   make(map[string]template.Template),
     }
 }

// GenerateWrapper generates a business customization wrapper for a project
 func (wg *WrapperGenerator) GenerateWrapper(projectName string, templateType string, outputDir string, enterprise bool) (*BusinessWrapper, error) {
     // Load template
     tmpl, err := wg.loadTemplate(templateType)
     if err != nil {
         return nil, fmt.Errorf("failed to load template: %w", err)
     )

     // Create project info for template
     projectInfo := ProjectInfo{
         Name:        projectName,
         Type:        wg.inferProjectType(projectName),
         Language:    wg.inferLanguage(projectName),
         Timestamp:   time.Now(),
         Author:      "RAPID AI",
         Version:     "1.0.0",
     }

     // Create output directory
     if outputDir == "" {
         outputDir = "./generated_wrappers"
     }

     if err := os.MkdirAll(outputDir, 0755); err != nil {
         return nil, fmt.Errorf("failed to create output directory: %w", err)
     }

     // Execute template to generate content
     content, err := wg.executeTemplate(tmpl, projectInfo, templateType)
     if err != nil {
         return nil, fmt.Errorf("failed to execute template: %w", err)
     }

     // Determine wrapper filename
     wrapperFile := filepath.Join(outputDir, fmt.Sprintf("%s_%s_%s", projectName, templateType, time.Now().Format("20060102")))

     // Add file extension based on template type
     wrapperExt := ""
     switch templateType {
     case "go-wrapper":
         wrapperExt = ".go"
     case "python-wrapper":
         wrapperExt = ".py"
     case "node-wrapper":
         wrapperExt = ".js"
     case "terraform-wrapper":
         wrapperExt = ".tf"
     case "dockerfile-wrapper":
         wrapperExt = ".Dockerfile"
     default:
         wrapperExt = ".txt"
     }

     wrapperFile += wrapperExt

     // Write wrapper to file
     if err := os.WriteFile(wrapperFile, []byte(content), 0644); err != nil {
         return nil, fmt.Errorf("failed to write wrapper file: %w", err)
     }

     // Create metadata
     metadata := WrapperMetadata{
         Version:         projectInfo.Version,
         Author:          projectInfo.Author,
         Description:     wg.generateDescription(projectInfo),
         License:        "MIT",
         Dependencies:    wg.getDependencies(templateType),
         IntegrationPoints: wg.getIntegrationPoints(projectInfo),
     }

     // Calculate pricing
     estimatedTokens := len(content) / 4 // Rough estimate
     cost := wg.calculateWrapperCost(estimatedTokens, enterprise)

     pricing := WrapperPricing{
         TokensUsed:   estimatedTokens,
         Cost:         cost,
         Currency:     "USD",
         CacheHit:     false, // This would be determined by cache checking
         Enterprise:   enterprise,
         Timestamp:    time.Now(),
     }

     // Log pricing transaction
     wg.pricingLogger.LogWrapperTransaction(pricing)

     // Create metadata file
     metadataFile := wrapperFile + ".metadata.json"
     metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
     if err != nil {
         return nil, fmt.Errorf("failed to serialize metadata: %w", err)
     }

     if err := os.WriteFile(metadataFile, metadataJSON, 0644); err != nil {
         return nil, fmt.Errorf("failed to write metadata file: %w", err)
     }

     return &BusinessWrapper{
         ProjectName:    projectName,
         TemplateType:   templateType,
         Timestamp:      time.Now(),,
         Content:       content,
         FilePath:      wrapperFile,
         Metadata:      metadata,
         Pricing:       pricing,
     }, nil
 }

// loadTemplate loads a template from the template directory
 func (wg *WrapperGenerator) loadTemplate(templateType string) (template.Template, error) {
     // Check cache first
     if tmpl, exists := wg.templateCache[templateType]; exists {
         return tmpl, nil
     }

     // Load template file
     templatePath := filepath.Join(wg.templateDir, fmt.Sprintf("%s.template", templateType))

     // If template file doesn't exist, create a basic template
     if _, err := os.Stat(templatePath); os.IsNotExist(err) {
         return wg.createTemplate(templateType)
     }

     tmpl, err := template.ParseFiles(templatePath)
     if err != nil {
         return nil, fmt.Errorf("failed to parse template file: %w", err)
     }

     // Add to cache
     wg.templateCache[templateType] = *tmpl
     return *tmpl, nil
 }

// createTemplate creates a default template for a given type
 func (wg *WrapperGenerator) createTemplate(templateType string) (template.Template, error) {
     var templateStr string

     switch templateType {
     case "go-wrapper":
         templateStr = goWrapperTemplate
     case "python-wrapper":
         templateStr = pythonWrapperTemplate
     case "node-wrapper":
         templateStr = nodeWrapperTemplate
     case "terraform-wrapper":
         templateStr = terraformWrapperTemplate
     case "dockerfile-wrapper":
         templateStr = dockerfileWrapperTemplate
     default:
         templateStr = genericWrapperTemplate
     }

     tmpl, err := template.New(templateType).Parse(templateStr)
     if err != nil {
         return template.Template{}, fmt.Errorf("failed to parse default template: %w", err)
     }

     return *tmpl, nil
 }

// executeTemplate executes a template with the given project info
 func (wg *WrapperGenerator) executeTemplate(tmpl template.Template, info ProjectInfo, templateType string) (string, error) {
     var buf bytes.Buffer

     err := tmpl.Execute(&buf, info)
     if err != nil {
         return "", fmt.Errorf("failed to execute template: %w", err)
     }

     result := buf.String()

     return result, nil
 }

// inferProjectType infers project type from project name
 func (wg *WrapperGenerator) inferProjectType(projectName string) string {
     projectName = strings.ToLower(projectName)

     if strings.Contains(projectName, "docker") {
         return "docker"
     }
     if strings.Contains(projectName, "api") {
         return "api"
     }
     if strings.Contains(projectName, "web") {
         return "web"
     }
     if strings.Contains(projectName, "mobile") {
         return "mobile"
     }

     return "generic"
 }

// inferLanguage infers language from project name
 func (wg *WrapperGenerator) inferLanguage(projectName string) string {
     projectName = strings.ToLower(projectName)

     if strings.Contains(projectName, "python") || strings.Contains(projectName, "ml") || strings.Contains(projectName, "ai") {
         return "Python"
     }
     if strings.Contains(projectName, "javascript") || strings.Contains(projectName, "node") || strings.Contains(projectName, "ts") {
         return "JavaScript"
     }
     if strings.Contains(projectName, "go") {
         return "Go"
     }\n     if strings.Contains(projectName, "java") {
         return "Java"
     }

     return "Unknown"
 }

// generateDescription generates a description for the wrapper
 func (wg *WrapperGenerator) generateDescription(info ProjectInfo) string {
     return fmt.Sprintf("Auto-generated %s wrapper for %s project via RAPID AI analysis system.",
         strings.Title(info.Language), info.Name)
 }

// getDependencies returns typical dependencies for a template type
 func (wg *WrapperGenerator) getDependencies(templateType string) []string {
     dependencies := []string{
         "github.com/azzurrotech/rapid",
         "github.com/azzurrotech/pod",
         "github.com/azzurrotech/shepherd",
     }

     switch templateType {
     case "go-wrapper":
         dependencies = append(dependencies, "github.com/gin-gonic/gin")
     case "python-wrapper":
         dependencies = append(dependencies, "requests", "flask")
     case "node-wrapper":
         dependencies = append(dependencies, "express", "axios")
     case "terraform-wrapper":
         dependencies = append(dependencies, "terraform-provider")
     case "dockerfile-wrapper":
         dependencies = append(dependencies, "docker")
     }

     return dependencies
 }

// getIntegrationPoints returns integration points for a project
 func (wg *WrapperGenerator) getIntegrationPoints(info ProjectInfo) []string {
     points := []string{
         "POD metadata storage",
         "SHEPHERD security audit",
         "SONG middleware integration",
         "ATP workflow orchestration",
     }

     switch info.Language {
     case "Go":
         points = append(points, "Go module integration", "Containerization support")
     case "Python":
         points = append(points, "Data processing pipelines", "ML model integration")
     case "JavaScript":
         points = append(points, "Web server integration", "API gateway support")
     }

     return points
 }

// calculateWrapperCost calculates the cost for a wrapper based on tokens
 func (wg *WrapperGenerator) calculateWrapperCost(tokens int, enterprise bool) float64 {
     // Base cost: $0.002 per 1K tokens
     cost := float64(tokens) * 0.002 / 1000.0

     // Enterprise discount
     if enterprise {
         cost *= 0.7 // 30% discount for enterprise
     }

     return cost
 }

// goWrapperTemplate is the template for Go wrappers
 var goWrapperTemplate = `package main

{{/*
 * Auto-generated Go wrapper for {{ .Name }}
 * Language: {{ .Language }}
 * Generated by: RAPID AI Analysis System
 * Author: {{ .Author }}
 * Version: {{ .Version }}
 */}}

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
    response := map[string]interface{}{
        "status":      "success",
        "project":     "{{ .Name }}",
        "platform":    "azimuthro",
        "timestamp":   time.Now(),
        "version":     bw.Version,
        "integration": "all_systems_go",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        NewBusinessWrapper().HandleRequest(w, r)
    })

    http.ListenAndServe(":8080", nil)
}
`

// pythonWrapperTemplate is the template for Python wrappers
 var pythonWrapperTemplate = `#!/usr/bin/env python3
"""
Auto-generated Python wrapper for {{ .Name }}
Language: {{ .Language }}
Generated by: RAPID AI Analysis System
Author: {{ .Author }}
Version: {{ .Version }}
"""

import json
import time
from datetime import datetime

class BusinessWrapper:
    def __init__(self):
        self.project_name = "{{ .Name }}"
        self.created_at = datetime.now()
        self.version = "{{ .Version }}"

    def handle_request(self, request_body=None):
        response = {
            "status": "success",
            "project": self.project_name,
            "platform": "azimuthro",
            "timestamp": datetime.now().isoformat(),
            "version": self.version,
            "integration": "all_systems_go"
        }

        return response

if __name__ == "__main__":
    wrapper = BusinessWrapper()
    print(json.dumps(wrapper.handle_request(), indent=2))

# API endpoint functionality
from flask import Flask, request, jsonify

app = Flask(__name__)
wrapper = BusinessWrapper()

@app.route('/')
def index():
    return jsonify(wrapper.handle_request())

@app.route('/health')
def health():
    return jsonify({"status": "healthy", "timestamp": datetime.now().isoformat()})

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=8080)
`

// nodeWrapperTemplate is the template for Node.js wrappers
 var nodeWrapperTemplate = `// Auto-generated Node.js wrapper for {{ .Name }}
// Language: {{ .Language }}
// Generated by: RAPID AI Analysis System
// Author: {{ .Author }}
// Version: {{ .Version }}

const express = require('express');
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
            platform: "azimuthro",
            timestamp: this.createdAt.toISOString(),
            version: this.version,
            integration: "all_systems_go"
        };
    }
}

const wrapper = new BusinessWrapper();

app.get('/', (req, res) => {
    res.json(wrapper.handleRequest());
});

app.get('/health', (req, res) => {
    res.json({
        status: "healthy",
        timestamp: new Date().toISOString()
    });
});

const PORT = process.env.PORT || 8080;
app.listen(PORT, () => {
    console.log(`Business wrapper server running on port ${PORT}`);
});

module.exports = {
    BusinessWrapper,
    app
};
`

// terraformWrapperTemplate is the template for Terraform wrappers
 var terraformWrapperTemplate = `# Auto-generated Terraform wrapper for {{ .Name }}
 # Language: {{ .Language }}
 # Generated by: RAPID AI Analysis System
 # Author: {{ .Author }}
 # Version: {{ .Version }}
 #
 # This Terraform configuration provides infrastructure as code for the business wrapper
 # Integration with Azzurro's platform orchestration

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
        from_port   = 80
        to_port     = 80
        protocol    = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

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

resource "aws_instance" "wrapper_server" {
    ami           = "ami-0c55b159cbfafe1f0"  # Amazon Linux 2
    instance_type = "t2.micro"
    security_groups = [aws_security_group.wrapper_sg.name]

    tags = {
        Name        = "${var.project_name}"
        Environment = "production"
        ManagedBy   = "RAPID"
        Author      = "{{ .Author }}"
        Version     = "{{ .Version }}"
    }
}

output "wrapper_endpoint" {
    value = "http://${aws_instance.wrapper_server.public_ip}"
}

output "project_info" {
    value = {
        name        = var.project_name
        created_by  = "{{ .Author }}"
        created_at  = "{{ .CreatedAt.Format \"2006-01-02\" }}"
        version     = "{{ .Version }}"
    }
}
`

// dockerfileWrapperTemplate is the template for Dockerfile wrappers
 var dockerfileWrapperTemplate = `# Auto-generated Dockerfile wrapper for {{ .Name }}
# Language: {{ .Language }}
# Generated by: RAPID AI Analysis System
# Author: {{ .Author }}
# Version: {{ .Version }}
#
# Business Docker image for the project with full Azzurro platform integration

FROM golang:1.21-alpine

WORKDIR /app

# Install dependencies
RUN apk add --no-cache curl git

# Create application directory
RUN mkdir -p /app/{{ lower .Name }}

WORKDIR /app/{{ lower .Name }}

# Copy source code
COPY . /app/{{ lower .Name }}

# Build the application
RUN go mod download
RUN go build -o main .

# Install runtime dependencies
RUN curl -sfL https://dl.getdocker.com | sh

# Add business wrapper scripts
COPY wrapper.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/wrapper.sh

# Create runtime user
RUN adduser -D appuser
USER appuser

# Application configuration
ENV APP_NAME="{{ .Name }}"
ENV APP_VERSION="{{ .Version }}"
ENV PLATFORM="azimuthro"
ENV INTEGRATION="full"

# Health check endpoint
COPY healthcheck.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/healthcheck.sh
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD /usr/local/bin/healthcheck.sh

# Expose ports
EXPOSE 8080 9090

# Application entry point
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["/app/{{ lower .Name }}/main"]\n
# Labels for metadata
LABEL \\
    maintainer="RAPID AI" \\
    version="{{ .Version }}" \\
    author="{{ .Author }}" \\
    platform="azimuthro" \\
    integration="pod,shepherd,song,atp" \\
    description="Business wrapper for {{ .Name }} project"

# Cleanup
RUN rm -rf /root/.cache/go-build
`

// genericWrapperTemplate is the default template
 var genericWrapperTemplate = `# Auto-generated wrapper for {{ .Name }}
# Language: {{ .Language }}

package main

import "fmt"

func main() {
    fmt.Printf("Business wrapper for {{ .Name }} (%s)\n", "{{ .Version }}")
    fmt.Printf("Generated by RAPID AI on {{ .CreatedAt.Format \"2006-01-02 15:04:05\" }}\n")
    fmt.Printf("Author: {{ .Author }}\n")
    fmt.Printf("Platform: azimuthro\n")
}