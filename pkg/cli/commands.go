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

	rapid "rapid.ai"
	"rapid.ai/pkg/agent"
)

// Command is one CLI verb.
type Command struct {
	Name        string
	Description string
	Usage       string
	Flags       []Flag
	Action      func(*CommandContext) error
}

// Flag is one command-line option.
type Flag struct {
	Name        string
	Short       string
	Description string
	Default     string
	Required    bool
}

// CommandContext carries the parsed invocation.
type CommandContext struct {
	App       *rapid.RAPIDApplication
	Args      []string
	Flags     map[string]string
	Writer    io.Writer
	ErrWriter io.Writer
	Config    *CLIConfig
}

// CLIConfig is the small set of options the command layer owns.
type CLIConfig struct {
	Version    string
	HomeDir    string
	ConfigFile string
	Debug      bool
	LogLevel   string
}

// Commands is the command table.
type Commands struct {
	app      *rapid.RAPIDApplication
	Analyze  *Command
	Generate *Command
	Complete *Command
	Help     *Command
	Version  *Command
}

// NewCommands builds the table around a running application.
func NewCommands(app *rapid.RAPIDApplication) *Commands {
	return &Commands{
		app:      app,
		Analyze:  createAnalyzeCommand(),
		Generate: createGenerateCommand(),
		Complete: createCompleteCommand(),
		Help:     createHelpCommand(),
		Version:  createVersionCommand(),
	}
}

// ExecuteCommand runs one command.
func (cm *Commands) ExecuteCommand(cmdName string, ctx *CommandContext) error {
	if ctx.App == nil {
		ctx.App = cm.app
	}
	switch strings.ToLower(cmdName) {
	case "analyze":
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

func createAnalyzeCommand() *Command {
	return &Command{
		Name:        "analyze",
		Description: "Analyze a project directory for AI opportunities",
		Usage:       "rapid-cli analyze <path> [flags]",
		Flags: []Flag{
			{Name: "output", Short: "o", Description: "Output format (json|text)", Default: "text"},
			{Name: "verbose", Short: "v", Description: "Enable verbose output", Default: "false"},
			{Name: "max-tokens", Description: "Maximum tokens for analysis", Default: "4000"},
		},
		Action: handleAnalyzeCommand,
	}
}

func createGenerateCommand() *Command {
	return &Command{
		Name:        "generate",
		Description: "Generate a business customization wrapper",
		Usage:       "rapid-cli generate <project-name> <template-type> [flags]",
		Flags: []Flag{
			{Name: "output-dir", Short: "o", Description: "Output directory for generated files", Default: "./generated_wrappers"},
			{Name: "template-type", Short: "t", Description: "Type of wrapper to generate", Default: "go-wrapper"},
			{Name: "enterprise", Short: "e", Description: "Generate with enterprise pricing", Default: "false"},
			{Name: "language", Short: "l", Description: "Target programming language", Default: "go"},
		},
		Action: handleGenerateCommand,
	}
}

func createCompleteCommand() *Command {
	return &Command{
		Name:        "complete",
		Description: "Complete a multi-step workflow",
		Usage:       "rapid-cli complete <workflow>",
		Flags: []Flag{
			{Name: "force", Short: "f", Description: "Force completion", Default: "false"},
		},
		Action: handleCompleteCommand,
	}
}

func createHelpCommand() *Command {
	return &Command{
		Name:        "help",
		Description: "Show help information",
		Usage:       "rapid-cli help [command]",
		Action:      handleHelpCommand,
	}
}

func createVersionCommand() *Command {
	return &Command{
		Name:        "version",
		Description: "Show version information",
		Usage:       "rapid-cli version",
		Action:      handleVersionCommand,
	}
}

func handleAnalyzeCommand(ctx *CommandContext) error {
	if ctx.App == nil {
		return fmt.Errorf("rapid: no application configured")
	}
	path, err := requirePath(ctx)
	if err != nil {
		return err
	}
	result, err := ctx.App.AnalyzeProject(path)
	if err != nil {
		return err
	}

	switch outputFormat(ctx) {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("encode analysis: %w", err)
		}
		fmt.Fprintln(ctx.Writer, string(data))
	case "text":
		printTextAnalysis(ctx.Writer, result)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat(ctx))
	}
	return nil
}

func handleGenerateCommand(ctx *CommandContext) error {
	if ctx.App == nil {
		return fmt.Errorf("rapid: no application configured")
	}
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: rapid-cli generate <project-name> <template-type>")
	}
	projectName := ctx.Args[0]
	templateType := ctx.Flags["template-type"]
	if templateType == "" {
		templateType = "go-wrapper"
	}
	if len(ctx.Args) > 1 {
		templateType = ctx.Args[1]
	}
	outputDir := ctx.Flags["output-dir"]
	if outputDir == "" {
		outputDir = "./generated_wrappers"
	}
	enterprise := truthy(ctx.Flags["enterprise"])

	wrapper, err := ctx.App.GenerateNamedWrapper(projectName, templateType, outputDir, enterprise)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Writer, "Wrapper generated successfully:\n")
	fmt.Fprintf(ctx.Writer, "  File: %s\n", wrapper.FilePath)
	fmt.Fprintf(ctx.Writer, "  Project: %s\n", wrapper.ProjectName)
	fmt.Fprintf(ctx.Writer, "  Template Type: %s\n", wrapper.TemplateType)
	fmt.Fprintf(ctx.Writer, "  Generated At: %s\n", wrapper.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(ctx.Writer, "  Cost: $%.4f\n", wrapper.Pricing.Cost)
	return nil
}

func handleCompleteCommand(ctx *CommandContext) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: rapid-cli complete <workflow>")
	}
	fmt.Fprintf(ctx.Writer, "Completing workflow: %s\n", ctx.Args[0])
	return nil
}

func handleHelpCommand(ctx *CommandContext) error {
	if len(ctx.Args) < 1 {
		printGeneralHelp(ctx.Writer)
		return nil
	}
	printCommandHelp(ctx.Writer, ctx.Args[0])
	return nil
}

func handleVersionCommand(ctx *CommandContext) error {
	fmt.Fprintf(ctx.Writer, "RAPID CLI v1.0.0\n")
	fmt.Fprintf(ctx.Writer, "Azzurro Technology AI Analysis and Wrapper Generation\n\n")
	fmt.Fprintf(ctx.Writer, "Built with Go\n")
	return nil
}

func requirePath(ctx *CommandContext) (string, error) {
	if len(ctx.Args) < 1 {
		return "", fmt.Errorf("path argument is required")
	}
	abs, err := filepath.Abs(ctx.Args[0])
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("path %s: %w", abs, err)
	}
	return abs, nil
}

func outputFormat(ctx *CommandContext) string {
	if f := ctx.Flags["output"]; f != "" {
		return f
	}
	return "text"
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// printTextAnalysis renders an analysis for a terminal.
func printTextAnalysis(w io.Writer, result agent.AnalysisResult) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Project Analysis Results")
	fmt.Fprintln(tw, "=======================")
	fmt.Fprintln(tw)
	fmt.Fprintf(tw, "Project:\t%s\n", result.ProjectName)
	fmt.Fprintf(tw, "Path:\t%s\n", result.ProjectPath)
	fmt.Fprintf(tw, "Files:\t%d\n", result.TotalFiles)
	fmt.Fprintf(tw, "Size:\t%d bytes\n", result.TotalSize)
	fmt.Fprintf(tw, "Complexity:\t%.2f\n", result.Complexity)
	fmt.Fprintf(tw, "Has Tests:\t%v\n", result.HasTests)
	fmt.Fprintf(tw, "Has CI:\t%v\n", result.HasCI)
	fmt.Fprintln(tw)

	fmt.Fprintln(tw, "Recommendations:")
	for i, rec := range result.Recommendations {
		fmt.Fprintf(tw, "%d.\t[%s] %s - %s\n", i+1, rec.Priority, rec.Title, rec.Description)
	}
	fmt.Fprintln(tw)

	fmt.Fprintln(tw, "Integration Projects:")
	for _, p := range result.IntegrationProjects {
		fmt.Fprintf(tw, "  %s (%s): %.0f%%\n", p.Name, p.Platform, p.Compatibility*100)
	}
	fmt.Fprintln(tw)

	fmt.Fprintf(tw, "Generated Code Files: %d\n", len(result.GeneratedCode))
	for _, code := range result.GeneratedCode {
		fmt.Fprintf(tw, "  - %s (%s)\n", code.File, code.Language)
	}
	fmt.Fprintln(tw)

	fmt.Fprintln(tw, "Pricing Analysis:")
	fmt.Fprintf(tw, "  Estimated Tokens:\t%d\n", result.Pricing.EstimatedTokens)
	fmt.Fprintf(tw, "  Estimated Cost:\t$%.4f\n", result.Pricing.EstimatedCost)
	fmt.Fprintf(tw, "  Cache Efficiency:\t%.0f%%\n", result.Pricing.CacheEfficiency*100)
	fmt.Fprintf(tw, "  Currency:\t%s\n", result.Pricing.Currency)
	_ = tw.Flush()
}

func printGeneralHelp(w io.Writer) {
	fmt.Fprintln(w, "RAPID CLI - Azzurro Technology AI Analysis and Wrapper Generation")
	fmt.Fprintln(w, "===================================================================")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  rapid-cli <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  analyze <path> [--output json|text] [--max-tokens n]")
	fmt.Fprintln(w, "    Analyze a project directory for AI opportunities")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  generate <project-name> <template-type> [--output-dir dir] [--enterprise]")
	fmt.Fprintln(w, "    Generate a business customization wrapper")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  complete <workflow> [--force]")
	fmt.Fprintln(w, "    Complete a multi-step workflow")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  help [command]")
	fmt.Fprintln(w, "    Show help information")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  version")
	fmt.Fprintln(w, "    Show version information")
}

func printCommandHelp(w io.Writer, command string) {
	for _, cmd := range []*Command{
		createAnalyzeCommand(), createGenerateCommand(), createCompleteCommand(),
		createHelpCommand(), createVersionCommand(),
	} {
		if cmd.Name != command {
			continue
		}
		fmt.Fprintf(w, "%s - %s\n", cmd.Usage, cmd.Description)
		fmt.Fprintln(w, "Flags:")
		for _, flag := range cmd.Flags {
			fmt.Fprintf(w, "  --%s%s: %s (default: %s)\n", flag.Name, flag.short(), flag.Description, flag.Default)
		}
		return
	}
	fmt.Fprintf(w, "Unknown command: %s\n", command)
}

func (f *Flag) short() string {
	if f.Short != "" {
		return fmt.Sprintf(" (-%s)", f.Short)
	}
	return ""
}
