package shepherdclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ParseTTL parses a TTL string, accepting Go duration syntax or a numeric seconds value.
func ParseTTL(ttl string, defaultTTL time.Duration) time.Duration {
	if ttl == "" {
		return defaultTTL
	}
	// Try as duration string (e.g. "72h", "7*24h")
	d, err := time.ParseDuration(ttl)
	if err == nil {
		return d
	}
	// Try as plain seconds
	if n, err := StringToInt(ttl); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return defaultTTL
}

// StringToInt converts a string to int.
func StringToInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// ShareOptions options for issuing a shepherd share token.
type ShareOptions struct {
	Kind   string   `json:"kind"`
	ID     string   `json:"id"`
	TTL    string   `json:"ttl"`
	Roles  []string `json:"roles,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
	Meta   map[string]string `json:"meta,omitempty"`
}

// Share is the result of issuing a shepherd capability token.
type Share struct {
	Token      string `json:"token"`
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	ExpiresAt  int64  `json:"expires_at"`
}

// Client is a minimal shepherd HTTP client for rapid.
type Client struct {
	base string
	http *http.Client
}

// NewClient returns a client for base.
func NewClient(base string) *Client {
	return &Client{base: strings.TrimRight(base, "/"), http: &http.Client{Timeout: 30 * time.Second}}
}

// Issue issues a shepherd capability token.
func (c *Client) Issue(o ShareOptions) (*Share, error) {
	url := c.base + "/api/keys"
	data, _ := json.Marshal(struct {
		Kind   string `json:"kind"`
		ID     string `json:"id"`
		TTL    string `json:"ttl"`
	}{
		Kind: o.Kind,
		ID:    o.ID,
		TTL:   o.TTL,
	})
	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("shepherd issue: %s", resp.Status)
	}
	var out struct {
		Token    string `json:"token"`
		Kind     string `json:"kind"`
		ID       string `json:"id"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &Share{
		Token:      out.Token,
		Kind:       out.Kind,
		ID:         out.ID,
		ExpiresAt:  out.ExpiresAt,
	}, nil
}

// Revoke revokes a shepherd capability token.
func (c *Client) Revoke(jti string) error {
	url := c.base + "/api/revoke"
	var body struct {
		JTI string `json:"jti"`
	}
	body.JTI = jti
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	return nil
}
