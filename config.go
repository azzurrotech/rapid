package rapid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RAPIDConfig configures a RAPID application. The zero value is usable:
// defaults fill every unset field.
type RAPIDConfig struct {
	Version            string        `json:"version"`
	Host               string        `json:"host"`
	Port               int           `json:"port"`
	MaxConcurrentOps   int           `json:"max_concurrent_ops"`
	CacheSizeMB        int           `json:"cache_size_mb"`
	MaxTokens          int           `json:"max_tokens"`
	SecretRotation     string        `json:"secret_rotation"`
	IntegrationTimeout time.Duration `json:"integration_timeout"`
	RotationInterval   Duration      `json:"-"` // set via SecretRotation or env

	// DataDir is the atp root: it holds the pod store, song's hosted apps,
	// the secrets vault and the platform state.
	DataDir string `json:"data_dir"`
	// Secret is the atp master secret (at least 32 bytes). When empty, a
	// persistent secret is generated under DataDir and reused.
	Secret string `json:"-"`
	// OllamaURL points at the local model runtime.
	OllamaURL string `json:"ollama_url"`
	// TemplateDir and OutputDir configure wrapper generation.
	TemplateDir string `json:"template_dir"`
	OutputDir   string `json:"output_dir"`
	// Enterprise selects enterprise pricing.
	Enterprise bool `json:"enterprise"`
}

// Duration is a duration of time, wrapping time.Duration for config use.
type Duration time.Duration

// RotationPolicy defines when and how secrets should be rotated.
type RotationPolicy struct {
	// Interval is the maximum age of a secret before rotation.
	// Empty means no automatic rotation.
	Interval Duration `json:"interval"`
	// MaxAge is the absolute maximum age of a secret.
	MaxAge Duration `json:"max_age"`
	// AllowWithoutSecret allows operation without a secret (for testing).
	AllowWithoutSecret bool `json:"allow_without_secret"`
}

// DefaultConfig returns the configuration the CLI and server start from.
func DefaultConfig() *RAPIDConfig {
	return &RAPIDConfig{
		Version:            "1.0.0",
		Host:               "localhost",
		Port:               8080,
		MaxConcurrentOps:   10,
		CacheSizeMB:        512,
		MaxTokens:          4096,
		SecretRotation:     "hourly",
		IntegrationTimeout: 30 * time.Second,
		DataDir:            "./data",
		OllamaURL:          "http://localhost:11434",
		TemplateDir:        "./templates",
		OutputDir:          "./generated_wrappers",
	}
}

// Normalize fills defaults and validates the result.
func (c *RAPIDConfig) Normalize() error {
	if c == nil {
		return fmt.Errorf("rapid: nil config")
	}
	if c.Version == "" {
		c.Version = "1.0.0"
	}
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.MaxConcurrentOps <= 0 {
		c.MaxConcurrentOps = 10
	}
	if c.CacheSizeMB <= 0 {
		c.CacheSizeMB = 512
	}
	if c.MaxTokens <= 0 {
		c.MaxTokens = 4096
	}
	if c.IntegrationTimeout <= 0 {
		c.IntegrationTimeout = 30 * time.Second
	}
	if c.DataDir == "" {
		c.DataDir = "./data"
	}
	if c.OllamaURL == "" {
		c.OllamaURL = "http://localhost:11434"
	}
	if c.TemplateDir == "" {
		c.TemplateDir = "./templates"
	}
	if c.OutputDir == "" {
		c.OutputDir = "./generated_wrappers"
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("rapid: port %d is out of range", c.Port)
	}
	// Parse the SecretRotation field: accept "hourly", "daily", or a Go
	// duration string (e.g. "72h", "1h30m"). Empty or "never" disables rotation.
	if c.SecretRotation != "" && c.SecretRotation != "never" {
		if dur, ok := parseDurationEnv(c.SecretRotation); ok && dur > 0 {
			c.RotationInterval = Duration(dur)
		}
	}
	return nil
}

// parseDurationEnv parses a duration string environment variable. It returns
// the duration and whether the parsing succeeded. Accepted formats include
// "hourly", "daily", and Go duration strings like "72h", "1h30m".
func parseDurationEnv(s string) (time.Duration, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "hourly":
		return time.Hour, true
	case "daily":
		return 24 * time.Hour, true
	case "never":
		return 0, true
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, false
	}
	return d, true
}

// Addr is the host:port the web server listens on.
func (c *RAPIDConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ResolveSecret returns the master secret, generating and persisting one under
// DataDir when the caller did not supply one. The file is created 0600, because
// the same secret derives the vault key, the pod encryption key and shepherd's
// signing key.
func (c *RAPIDConfig) ResolveSecret() (string, error) {
	if c.Secret != "" {
		if len(c.Secret) < 32 {
			return "", fmt.Errorf("rapid: secret must be at least 32 bytes")
		}
		return c.Secret, nil
	}
	if env := os.Getenv("RAPID_SECRET"); env != "" {
		if len(env) < 32 {
			return "", fmt.Errorf("rapid: RAPID_SECRET must be at least 32 bytes")
		}
		c.Secret = env
		return env, nil
	}

	if err := os.MkdirAll(c.DataDir, 0o700); err != nil {
		return "", fmt.Errorf("rapid: create data dir: %w", err)
	}
	path := filepath.Join(c.DataDir, ".rapid-secret")
	if data, err := os.ReadFile(path); err == nil && len(data) >= 32 {
		c.Secret = string(data)
		return c.Secret, nil
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("rapid: generate secret: %w", err)
	}
	secret := hex.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
		return "", fmt.Errorf("rapid: persist secret: %w", err)
	}
	c.Secret = secret
	return secret, nil
}
