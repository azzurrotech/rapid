package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "time"

    rapid "github.com/matthew/Projects/BLUE/rapid"
)

// RAPIDCLIOptions holds the configuration for the RAPID CLI
type RAPIDCLIOptions struct {
    Version       string        `json:"version"`
    Port          int           `json:"port"`
    Host          string        `json:"host"`
    MaxTokens     int           `json:"max_tokens"`
    MaxConcurrency int          `json:"max_concurrency"`
    CacheSizeMB   int           `json:"cache_size_mb"`
}

// RAPIDCLI holds the CLI application context
type RAPIDCLI struct {
    options *RAPIDCLIOptions
    app     *rapid.RAPIDApplication
}

// NewRAPIDCLI creates a new RAPID CLI instance
func NewRAPIDCLI(options *RAPIDCLIOptions) (*RAPIDCLI, error) {
    // Initialize RAPID application
    appOptions := &rapid.RAPIDConfig{
        Version:           options.Version,
        Port:              options.Port,
        Host:              options.Host,
        MaxConcurrentOps:  options.MaxConcurrency,
        CacheSizeMB:       options.CacheSizeMB,
        MaxTokens:         options.MaxTokens,
        SecretRotation:    "90d", // Default 90-day rotation
        IntegrationTimeout: time.Minute,
        AIEngineType:      "local-ollama",
    }

    app, err := rapid.NewRAPIDApplication(appOptions)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize RAPID application: %w", err)
    }

    return &RAPIDCLI{
        options: options,
        app:     app,
    }, nil
}

// AnalyzeProject analyzes a given directory for AI project opportunities
func (cli *RAPIDCLI) AnalyzeProject(path string, outputFormat string) error {
    // Get list of projects (demonstrates integration with existing data)
    result := cli.app.HandleGetProjects()

    // Create analysis output
    analysis := map[string]interface{}{
        "project_path":     path,
        "analysis_date":    time.Now().Format(time.RFC3339),
        "projects_found":   len(result.Projects),
        "recommendations":  cli.generateRecommendations(result.Projects),
        "integration_capabilities": map[string]bool{
            "POD":        true,
            "SHEPHERD":   true,
            "SONG":       true,
            "ATP":        true,
        },
    }

    // Output based on format
    switch outputFormat {
    case "json":
        data, err := json.MarshalIndent(analysis, "", "  ")
        if err != nil {
            return fmt.Errorf("failed to marshal analysis: %w", err)
        }
        fmt.Println(string(data))
    case "text":
        fmt.Printf("AI Project Analysis for: %s\n", path)
        fmt.Printf("Projects Found: %d\n", len(result.Projects))
        fmt.Println("\nRecommendations:")
        for _, rec := range analysis["recommendations"].([]string) {
            fmt.Printf("  - %s\n", rec)
        }
        fmt.Printf("\nIntegration Capabilities:\n")
        for module, compatible := range analysis["integration_capabilities"].(map[string]bool) {
            status := "No"
            if compatible {
                status = "Yes"
            }
            fmt.Printf("  - %s: %s\n", module, status)
        }
    default:
        return fmt.Errorf("unsupported output format: %s", outputFormat)
    }

    return nil
}

// GenerateWrapper generates a business customization wrapper for a project
func (cli *RAPIDCLI) GenerateWrapper(projectName string, templateType string, outputDir string) error {
    // Generate wrapper template based on type
    wrapper, err := cli.app.GenerateBusinessWrapper(projectName, templateType)
    if err != nil {
        return fmt.Errorf("failed to generate wrapper: %w", err)
    }

    // Create output directory
    if outputDir == "" {
        outputDir = "./generated_wrappers"
    }

    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %w", err)
    }

    // Write wrapper to file
    wrapperFile := filepath.Join(outputDir, fmt.Sprintf("%s_%s.go", projectName, templateType))
    if err := os.WriteFile(wrapperFile, []byte(wrapper), 0644); err != nil {
        return fmt.Errorf("failed to write wrapper file: %w", err)
    }

    // Create wrapper metadata
    metadata := map[string]interface{}{
        "project":        projectName,
        "template_type":  templateType,
        "generation_date": time.Now().Format(time.RFC3339),
        "wrapper_type":   "business_customization",
        "integration":    "azimuthro-platform",
        "license":        "MIT",
    }

    metadataFile := filepath.Join(outputDir, fmt.Sprintf("%s_%s.metadata.json", projectName, templateType))
    metadataData, err := json.MarshalIndent(metadata, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %w", err)
    }

    if err := os.WriteFile(metadataFile, metadataData, 0644); err != nil {
        return fmt.Errorf("failed to write metadata file: %w", err)
    }

    log.Printf("Wrapper generated successfully: %s", wrapperFile)
    log.Printf("Metadata file created: %s", metadataFile)

    return nil
}

// generateRecommendations generates recommendations based on analyzed projects
func (cli *RAPIDCLI) generateRecommendations(projects []rapid.AIProject) []string {
    recommendations := []string{}

    if len(projects) == 0 {
        return []string{"No projects available for analysis"}
    }

    for _, project := range projects {
        if project.Status == "active" && project.Lang == "Go" {
            recommendations = append(recommendations, fmt.Sprintf("Generate Go wrapper for %s - highly compatible with Azzurro platform", project.Name))
        } else if project.Status == "active" && project.Lang == "Python" {
            recommendations = append(recommendations, fmt.Sprintf("Generate Python wrapper for %s - good for data processing pipelines", project.Name))
        }
    }

    if len(recommendations) == 0 {
        recommendations = append(recommendations, "Consider focusing on Go-based projects for maximum integration benefits")
    }

    return recommendations
}

// PrintHelp displays help information for RAPID CLI commands
func (cli *RAPIDCLI) PrintHelp() {
    fmt.Println("RAPID CLI - Azzurro Technology AI Analysis and Wrapper Generation")
    fmt.Println("=================================================================")
    fmt.Println("")
    fmt.Println("USAGE:")
    fmt.Println("  rapid-cli <command> [options]")
    fmt.Println("")
    fmt.Println("COMMANDS:")
    fmt.Println("  analyze <path> [--output json|text]")
    fmt.Println("    Analyze a directory for AI project opportunities")
    fmt.Println("    Options:")
    fmt.Println("      --output json: Output detailed JSON analysis")
    fmt.Println("      --output text: Human-readable analysis (default)")
    fmt.Println("")
    fmt.Println("  generate <project-name> <template-type> [--output-dir <path>]")
    fmt.Println("    Generate a business customization wrapper for a project")
    fmt.Println("    Options:")
    fmt.Println("      --output-dir <path>: Output directory for generated files (default: ./generated_wrappers)")
    fmt.Println("")
    fmt.Println("EXAMPLES:")
    fmt.Println("  rapid-cli analyze ./my-project --output text")
    fmt.Println("  rapid-cli generate "Docker" "go-wrapper" --output-dir ./business-wrappers")
    fmt.Println("")
    fmt.Println("INTEGRATION WITH AZZURRO PLATFORM:")
    fmt.Println("  - Project analysis integrates with POD for metadata storage")
    fmt.Println("  - Wrapper generation compatible with SHEPHERD for security")
    fmt.Println("  - Web interface powered by SONG middleware")
    fmt.Println("  - Platform orchestration via ATP")
    fmt.Println("  - Pricing integration available via pkg/pricing engine")
    fmt.Println("")
    fmt.Println("For more information, visit: https://github.com/azzurrotech/rapid")
}