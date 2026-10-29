package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "time"

    rapid "rapid.ai"
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
        MaxConcurrentOps:  10,
        CacheSizeMB:       options.CacheSizeMB,
        MaxTokens:         options.MaxTokens,
        SecretRotation:    "hourly",
        IntegrationTimeout: 30 * time.Second,
    }

    app, err := rapid.NewRAPIDApplication(appOptions)
    if err != nil {
        return nil, fmt.Errorf("failed to create RAPID application: %w", err)
    }

    return &RAPIDCLI{
        options: options,
        app:     app,
    }, nil
}

// Run executes the RAPID CLI
func (cli *RAPIDCLI) Run(command string, args []string) error {
    switch command {
    case "analyze":
        return cli.analyzeCommand(args)
    case "generate":
        return cli.generateCommand(args)
    case "validate":
        return cli.validateCommand(args)
    case "wrap":
        return cli.wrapCommand(args)
    default:
        return fmt.Errorf("unknown command: %s", command)
    }
}

// analyzeCommand handles the analyze command
func (cli *RAPIDCLI) analyzeCommand(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: rapid-cli analyze [path]")
    }

    targetPath := args[0]
    if targetPath <= "." {
        targetPath = "."
    }

    absPath, err := filepath.Abs(targetPath)
    if err != nil {
        return fmt.Errorf("failed to get absolute path: %w", err)
    }

    fmt.Printf("Analyzing %s...\n", absPath)

    // Perform the analysis
    result, err := cli.app.AnalyzeProject(absPath)
    if err != nil {
        return fmt.Errorf("analysis failed: %w", err)
    }

    // Print results
    output, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(output))

    return nil
}

// generateCommand handles the generate command
func (cli *RAPIDCLI) generateCommand(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: rapid-cli generate [path]")
    }

    targetPath := args[0]
    absPath, err := filepath.Abs(targetPath)
    if err != nil {
        return fmt.Errorf("failed to get absolute path: %w", err)
    }

    fmt.Printf("Generating wrapper for %s...\n", absPath)

    // Extract wrapper definition
    wrapperDef, err := cli.app.GenerateWrapper(absPath)
    if err != nil {
        return fmt.Errorf("wrapper generation failed: %w", err)
    }

    // Print results
    output, _ := json.MarshalIndent(wrapperDef, "", "  ")
    fmt.Println(string(output))

    return nil
}

// validateCommand handles the validate command
func (cli *RAPIDCLI) validateCommand(args []string) error {
    fmt.Println("Validating RAPID configuration...")

    if err := cli.app.ValidateConfiguration(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    fmt.Println("Configuration is valid!")

    // Validate integrations
    integrationsValid := cli.app.ValidateIntegrations()
    if integrationsValid {
        fmt.Println("Integrations are valid!")
    } else {
        fmt.Println("Warning: Some integrations may not be working correctly")
    }

    return nil
}

// wrapCommand handles the wrap command
func (cli *RAPIDCLI) wrapCommand(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: rapid-cli wrap [path]")
    }

    targetPath := args[0]
    absPath, err := filepath.Abs(targetPath)
    if err != nil {
        return fmt.Errorf("failed to get absolute path: %w", err)
    }

    fmt.Printf("Creating wrapper for %s...\n", absPath)

    // Create wrapper (full generation + packaging)
    wrapperPath, err := cli.app.CreateWrapper(absPath)
    if err != nil {
        return fmt.Errorf("wrapper creation failed: %w", err)
    }

    fmt.Printf("Wrapper created at: %s\n", wrapperPath)

    return nil
}

func main() {
    // Default options
    options := &RAPIDCLIOptions{
        Version:       "1.0.0",
        Port:          8080,
        Host:          "localhost",
        MaxTokens:     4096,
        MaxConcurrency: 10,
        CacheSizeMB:   512,
    }

    cli, err := NewRAPIDCLI(options)
    if err != nil {
        log.Fatalf("Failed to initialize CLI: %v", err)
    }

    // Determine command and args from os.Args
    if len(os.Args) < 2 {
        fmt.Println("Usage: rapid-cli [command] [args]")
        fmt.Println("Available commands:")
        fmt.Println("  analyze [path] - Analyze a project for AI opportunities")
        fmt.Println("  generate [path] - Generate wrapper definition")
        fmt.Println("  validate - Validate RAPID configuration")
        fmt.Println("  wrap [path] - Create complete wrapper")
        os.Exit(1)
    }

    command := os.Args[1]
    args := os.Args[2:]

    // Execute command
    if err := cli.Run(command, args); err != nil {
        log.Fatalf("Error: %v", err)
    }
}
