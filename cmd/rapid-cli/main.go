// Command rapid-cli is the RAPID command-line front end. It runs the same
// application and the same real atp platform as the web server.
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	rapid "rapid.ai"
	"rapid.ai/pkg/cli"
	"rapid.ai/pkg/web"
)

func main() {
	cfg := rapid.DefaultConfig()
	if v := os.Getenv("RAPID_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("RAPID_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("RAPID_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Port = n
		}
	}

	args, flags := splitArgs(os.Args[1:])
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	applyFlags(cfg, flags)
	command := args[0]
	args = args[1:]

	app, err := rapid.NewRAPIDApplication(cfg)
	if err != nil {
		log.Fatalf("RAPID: %v", err)
	}
	defer app.Close()

	if command == "serve" {
		if err := web.NewWebServer(app, cfg.Host, strconv.Itoa(cfg.Port)).Start(); err != nil {
			log.Fatalf("RAPID server: %v", err)
		}
		return
	}

	cmds := cli.NewCommands(app)
	ctx := &cli.CommandContext{
		App:       app,
		Args:      args,
		Flags:     flags,
		Writer:    os.Stdout,
		ErrWriter: os.Stderr,
		Config:    &cli.CLIConfig{Version: cfg.Version, LogLevel: "info"},
	}

	switch command {
	case "validate":
		if err := app.ValidateConfiguration(); err != nil {
			fmt.Fprintln(os.Stderr, "configuration:", err)
			os.Exit(1)
		}
		fmt.Println("configuration: ok")
		if failures := app.IntegrationFailures(); len(failures) > 0 {
			fmt.Println("integrations: degraded")
			for _, f := range failures {
				fmt.Println("  -", f)
			}
			os.Exit(1)
		}
		fmt.Println("integrations: ok (atp, pod, shepherd, song)")
		return
	case "wrap":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "usage: rapid-cli wrap <path>")
			os.Exit(1)
		}
		path, err := app.CreateWrapper(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		fmt.Println("wrapper created at:", path)
		return
	}

	if err := cmds.ExecuteCommand(command, ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// splitArgs separates flags from positional arguments. Both "--name value" and
// "--name=value" are accepted, as is the short form.
func splitArgs(argv []string) ([]string, map[string]string) {
	args := make([]string, 0, len(argv))
	flags := map[string]string{}
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			args = append(args, a)
			continue
		}
		name := strings.TrimLeft(a, "-")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			flags[name[:eq]] = name[eq+1:]
			continue
		}
		if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
			flags[name] = argv[i+1]
			i++
			continue
		}
		flags[name] = "true"
	}
	return args, flags
}

func applyFlags(cfg *rapid.RAPIDConfig, flags map[string]string) {
	if v := flags["data"]; v != "" {
		cfg.DataDir = v
	}
	if v := flags["host"]; v != "" {
		cfg.Host = v
	}
	if v := flags["port"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Port = n
		}
	}
	if v := flags["output-dir"]; v != "" {
		cfg.OutputDir = v
	}
	if v := flags["template-dir"]; v != "" {
		cfg.TemplateDir = v
	}
	if v := flags["enterprise"]; v == "true" || v == "1" {
		cfg.Enterprise = true
	}
}

func usage() {
	fmt.Println("RAPID - Azzurro Technology AI Analysis and Wrapper Generation")
	fmt.Println()
	fmt.Println("Usage: rapid-cli <command> [arguments] [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  analyze <path>            Analyze a project directory")
	fmt.Println("  generate <name> <type>    Generate a wrapper")
	fmt.Println("  validate                  Validate configuration and integrations")
	fmt.Println("  wrap <path>               Analyze and generate in one step")
	fmt.Println("  serve                     Start the web dashboard and API")
	fmt.Println("  version                   Print the version")
	fmt.Println()
	fmt.Println("Flags: --data DIR --host HOST --port N --output-dir DIR --enterprise")
}
