package cli

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "text/tabwriter"
    "time"
)

// Command represents a CLI command
type Command struct {
    Name        string
    Description  string
    Usage        string
    Flags        []Flag
    Action       func(*CommandContext) error
}

// Flag represents a command line flag
type Flag struct {
    Name        string
    Short       string
    Description  string
    Default     string
    Required    bool
}

// CommandContext holds the context for command execution
type CommandContext struct {
    Args      []string
    Flags     map[string]string
    Writer    io.Writer
    ErrWriter io.Writer
    Config    *CLIConfig
}

// CLIConfig holds the CLI configuration
type CLIConfig struct {
    Version      string
    HomeDir      string
    ConfigFile   string
    Debug        bool
    LogLevel     string
}

// Commands is the master list of available commands
type Commands struct {
    Analyze      *Command
    Generate     *Command
    Complete     *Command
    Help         *Command
    Version      *Command
}

// NewCommands creates a new commands manager
func NewCommands(config *CLIConfig) *Commands {\n    return &	Commands{
        Analyze: createAnalyzeCommand(),
        Generate: createGenerateCommand(),
        Complete: createCompleteCommand(),
        Help: createHelpCommand(),
        Version: createVersionCommand(),
    }
}

// ExecuteCommand executes a command with the given context
func (cm *Commands) ExecuteCommand(cmdName string, ctx *CommandContext) error {
    switch strings.ToLower(cmdName) {\n    case "analyze":
        return cm.Analyze.Action(ctx)
    case "generate":
        return cm.Generate.Action(ctx)
    case "complete":
        return cm.Complete.Action(ctx)
    case "help":
        return cm.Help.Action(ctx)
    case "version":
        return cm.Version.Action(ctx)
    default:
        return fmt.Errorf("unknown command: %s", cmdName)
    }
}

// createAnalyzeCommand creates the analyze command
func createAnalyzeCommand() *Command {
    return & Command{
        Name:        "analyze",
        Description: "Analyze a project directory for AI opportunities",
        Usage:       "rapid-cli analyze <path> [flags]",
        Flags: []Flag{
            {Name: "output", Short: "o", Description: "Output format (json|text)", Default: "text", Required: false},
            {Name: "verbose", Short: "v", Description: "Enable verbose output", Default: "false", Required: false},
            {Name: "max-tokens", Description: "Maximum tokens for analysis", Default: "4000", Required: false},
        },
        Action: func(ctx *CommandContext) error {
            return handleAnalyzeCommand(ctx)
        },
    }
}

// createGenerateCommand creates the generate command
func createGenerateCommand() *Command {
    return & Command{
        Name:        "generate",
        Description: "Generate a business customization wrapper",
        Usage:       "rapid-cli generate <project-name> <template-type> [flags]",
        Flags: []Flag{
            {Name: "output-dir", Short: "o", Description: "Output directory for generated files", Default: "./generated_wrappers", Required: false},
            {Name: "template-type", Short: "t", Description: "Type of wrapper to generate", Default: "go-wrapper", Required: false},
            {Name: "enterprise", Short: "e", Description: "Generate with enterprise pricing", Default: "false", Required: false},
            {Name: "language", Short: "l", Description: "Target programming language", Default: "go", Required: false},
        },
        Action: func(ctx *CommandContext) error {
            return handleGenerateCommand(ctx)
        },
    }
}

// createCompleteCommand creates the complete command
func createCompleteCommand() *Command {
    return & Command{
        Name:        "complete",
        Description: "Complete a multi-step workflow",
        Usage:       "rapid-cli complete <workflow>",
        Flags: []Flag{
            {Name: "force", Short: "f", Description: "Force completion", Default: "false", Required: false},
        },
        Action: func(ctx *CommandContext) error {
            return handleCompleteCommand(ctx)
        },
    }
}

// createHelpCommand creates the help command
func createHelpCommand() *Command {
    return & Command{
        Name:        "help",
        Description: "Show help information",
        Usage:       "rapid-cli help [command]",
        Flags:       []Flag{},
        Action: func(ctx *CommandContext) error {
            return handleHelpCommand(ctx)
        },
    }
}

// createVersionCommand creates the version command
func createVersionCommand() *Command {
    return & Command{
        Name:        "version",
        Description: "Show version information",
        Usage:       "rapid-cli version",
        Flags:       []Flag{},
        Action: func(ctx *CommandContext) error {
            return handleVersionCommand(ctx)
        },
    }
}

// handleAnalyzeCommand handles the analyze command
func handleAnalyzeCommand(ctx *CommandContext) error {
    // Get output format
    outputFormat := ctx.Flags["output"]
    if outputFormat == "" {
        outputFormat = "text"
    }

    // Get path to analyze
    if len(ctx.Args) < 1 {
        return fmt.Errorf("path argument is required")
    }
    projectPath := ctx.Args[0]

    // Get max tokens
    maxTokensStr := ctx.Flags["max-tokens"]
    maxTokens := 4000
    if maxTokensStr != "" {
        fmt.Sscanf(maxTokensStr, "%d", & maxTokens)
    }

    // Import and use the actual analyzer
    analyzer, err := agent.NewProjectAnalyzer("ollama/plan-oss-64k:latest", maxTokens)
    if err != nil {
        return fmt.Errorf("failed to initialize analyzer: %w", err)
    }

    // Analyze project
    result, err := analyzer.AnalyzeProject(projectPath)
    if err != nil {
        return fmt.Errorf("failed to analyze project: %w", err)
    }

    // Output based on format
    switch outputFormat {
    case "json":
        jsonData, err := json.MarshalIndent(result, "", "  ")
        if err != nil {
            return fmt.Errorf("failed to marshal JSON: %w", err)
        }
        fmt.Fprint(ctx.Writer, string(jsonData))
    case "text":
        printTextAnalysis(ctx.Writer, result)
    default:
        return fmt.Errorf("unsupported output format: %s", outputFormat)
    }

    return nil
}

// handleGenerateCommand handles the generate command
func handleGenerateCommand(ctx *CommandContext) error {
    // Get required arguments
    if len(ctx.Args) < 2 {
        return fmt.Errorf("project-name and template-type arguments are required")
    }
    projectName := ctx.Args[0]
    templateType := ctx.Args 1]

    // Get output directory
    outputDir := ctx.Flags["output-dir"]
    if outputDir == "" {
        outputDir = "./generated_wrappers"
    }

    // Get template type
    if templateType == "" {
        templateType = ctx.Flags["template-type"]
        if templateType == "" {
            templateType = "go-wrapper"
        }
    }

    // Get enterprise flag
    enterprise := false
    if enterpriseStr := ctx.Flags["enterprise"]; enterpriseStr != "" {
        enterprise = strings.ToLower(enterpriseStr) == "true" || enterpriseStr == "1"
    }

    // Import and use the wrapper generator
    pricingLogger := pricing.NewPriceLogger(100, "./pricing.log", "text")
    generator := agent.NewWrapperGenerator("./templates", pricingLogger)

    // Generate wrapper
    wrapper, err := generator.GenerateWrapper(projectName, templateType, outputDir, enterprise)
    if err != nil {
        return fmt.Errorf("failed to generate wrapper: %w", err)\n    }

    // Output success message
    fmt.Fprintf(ctx.Writer, "Wrapper generated successfully:\n")
    fmt.Fprintf(ctx.Writer, "  File: %s\n", wrapper.FilePath)
    fmt.Fprintf(ctx.Writer, "  Project: %s\n", wrapper.ProjectName)
    fmt.Fprintf(ctx.Writer, "  Template Type: %s\n", wrapper.TemplateType)
    fmt.Fprintf(ctx.Writer, "  Generated At: %s\n", wrapper.Timestamp.Format(time.RFC3822))
    fmt.Fprintf(ctx.Writer, "  Cost: $%.4f\n", wrapper.Pricing.Cost)

    return nil
}

// handleCompleteCommand handles the complete command
func handleCompleteCommand(ctx *CommandContext) error {
    if len(ctx.Args) < 1 {
        return fmt.Errorf("workflow argument is required")
    }

    workflow := ctx.Args[0]

    // Import and use the workflow manager
    // For now, just print a message
    fmt.Fprintf(ctx.Writer, "Completing workflow: %s\n", workflow)

    return nil
}

// handleHelpCommand handles the help command
func handleHelpCommand(ctx *CommandContext) error {
    if len(ctx.Args) < 1 {
        printGeneralHelp(ctx.Writer)
    } else {
        command := ctx.Args[0]
        printCommandHelp(ctx.Writer, command)
    }

    return nil
}

// handleVersionCommand handles the version command
func handleVersionCommand(ctx *CommandContext) error {
    config := "1.0.0"
    fmt.Fprintf(ctx.Writer, "RAPID CLI v%s\n", config)
    fmt.Fprintf(ctx.Writer, "Azzurro Technology AI Analysis and Wrapper Generation\n")
    fmt.Fprintf(ctx.Writer, "\n")
    fmt.Fprintf(ctx.Writer, "Built with Go
")

    return nil
}

// printTextAnalysis prints analysis results in text format
func printTextAnalysis(w io.Writer, result agent.AnalysisResult) {
    tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

    fmt.Fprintf(tw, "Project Analysis Results:
")
    fmt.Fprintf(tw, "=======================\n\n")
    fmt.Fprintf(tw,"Project: %s (%s)\n", result.Project.Name, result.Project.Lang)
    fmt.Fprintf(tw,"Type: %s\n", result.Project.Type)
    fmt.Fprintf(tw,"Size: %d bytes\n", result.Project.Size)
    fmt.Fprintf(tw,"Complexity: %.2f\n", result.Project.Complexity)
    fmt.Fprintf(tw,"Has Tests: %v\n", result.Project.HasTests)
    fmt.Fprintf(tw,"Has CI: %v\n\n", result.Project.HasCI)

    fmt.Fprintf(tw, "Recommendations:\n")
    for i, rec := range result.Recommendations {
        fmt.Fprintf(tw, "%d. [%s] %s - %s\n", i+1, rec.Priority, rec.Title, rec.Description)
    }

    fmt.Fprintf(tw, \n)
    fmt.Fprintf(tw, "Integration Compatibility:\n")
    for platform, compat := range result.IntegrationCompatibility {
        fmt.Fprintf(tw, "  %s: POD=%v SHEPHERD=%v SONG=%v ATP=%v (%.0f%%)\n",
            platform, compat.POD, compat.SHEPHERD, compat.SONG, compat.ATP, compat.Confidence*100)
    }

    fmt.Fprintf(tw, \n)
    fmt.Fprintf(tw, "Generated Code Files: %d\n", len(result.GeneratedCode))
    for _, code := range result.GeneratedCode {
        fmt.Fprintf(tw, "  - %s (%s): %s\n", code.File, code.Language, code.GeneratedAt.Format(time.RFC822))
    }

    fmt.Fprintf(tw, \n)
    fmt.Fprintf(tw, "Pricing Analysis:\n")
    fmt.Fprintf(tw, "  Estimated Tokens: %d\n", result.Pricing.EstimatedTokens)
    fmt.Fprintf(tw, "  Estimated Cost: $%.4f\n", result.Pricing.EstimatedCost)
    fmt.Fprintf(tw, "  Cache Efficiency: %.0f%%\n", result.Pricing.CacheEfficiency*100)
    fmt.Fprintf(tw, "  Currency: %s\n\n", result.Pricing.Currency)

    tw.Flush()
}

// printGeneralHelp prints general help information
func printGeneralHelp(w io.Writer) {
    fmt.Fprintf(w, "RAPID CLI - Azzurro Technology AI Analysis and Wrapper Generation\n")
    fmt.Fprintf(w, "===================================================================\n\n")\n    fmt.Fprintf(w, "Usage:\n")
    fmt.Fprintf(w, "  rapid-cli <command> [options]\n\n")

    fmt.Fprintf(w, "Commands:\n")
    fmt.Fprintf(w, "  analyze <path> [--output json|text] [--verbose] [--max-tokens n]\n")
    fmt.Fprintf(w, "    Analyze a project directory for AI opportunities\n\n")

    fmt.Fprintf(w, "  generate <project-name> <template-type> [--output-dir dir] [--enterprise] [--language lang]\n")
    fmt.Fprintf(w, "    Generate a business customization wrapper\n\n")

    fmt.Fprintf(w, "  complete <workflow> [--force]\n")
    fmt.Fprintf(w, "    Complete a multi-step workflow\n\n")

    fmt.Fprintf(w, "  help [command]\n")
    fmt.Fprintf(w, "    Show help information\n\n")

    fmt.Fprintf(w, "  version\n")
    fmt.Fprintf(w, "    Show version information\n\n")

    fmt.Fprintf(w, "For more information, visit: https://github.com/azzurrotech/rapid\n")
}

// printCommandHelp prints help information for a specific command
func printCommandHelp(w io.Writer, command string) {
    commands := []Command{
        createAnalyzeCommand(),
        createGenerateCommand(),
        createCompleteCommand(),
        createHelpCommand(),
        createVersionCommand(),
    }

    for _, cmd := range commands {
        if cmd.Name == command {
            fmt.Fprintf(w, "%s - %s\n", cmd.Usage, cmd.Description)
            fmt.Fprintf(w, "Flags:\n")
            for _, flag := range cmd.Flags {
                fmt.Fprintf(w, "  --%s%s: %s (default: %s)\n",
                    flag.Name, flag.getShort(), flag.Description, flag.Default)
            }
            return
        }
    }

    fmt.Fprintf(w, "Unknown command: %s\n", command)
}

// Helper function to get short flag
func (f *Flag) getShort() string {
    if f.Short != "" {
        return fmt.Sprintf(" (-%s)", f.Short)
    }
    return ""
}