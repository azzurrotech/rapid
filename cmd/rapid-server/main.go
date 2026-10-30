// Command rapid-server serves the RAPID dashboard and API on the real atp
// platform (pod, song and shepherd mounted in-process).
package main

import (
	"log"
	"os"
	"strconv"

	rapid "rapid.ai"
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

	app, err := rapid.NewRAPIDApplication(cfg)
	if err != nil {
		log.Fatalf("RAPID: %v", err)
	}
	defer app.Close()

	if err := web.NewWebServer(app, cfg.Host, strconv.Itoa(cfg.Port)).Start(); err != nil {
		log.Fatalf("RAPID server: %v", err)
	}
}
