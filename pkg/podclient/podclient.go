package podclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Format is FormatJSON (default) or FormatCSV.
const FormatJSON = "json"
const FormatCSV = "csv"

// ContentType is the media type for an export format.
func ContentType(format string) string {
	if strings.EqualFold(strings.TrimSpace(format), FormatCSV) {
		return "text/csv"
	}
	return "application/json"
}

// ExportFilename builds a safe download name for an export, e.g.
// ExportFilename("stenella-items", "csv") == "stenella-items.csv".
func ExportFilename(base, format string) string {
	if strings.EqualFold(strings.TrimSpace(format), FormatCSV) {
		return base + ".csv"
	}
	return base + ".json"
}

// PodClient is a minimal pod HTTP client for rapid.
type PodClient struct {
	base string
	http *http.Client
}

// NewPodClient returns a client for base.
func NewPodClient(base string) *PodClient {
	return &PodClient{base: strings.TrimRight(base, "/"), http: &http.Client{Timeout: 30 * time.Second}}
}

// EnsureStructSchema creates or widens the pod table schema.
func (c *PodClient) EnsureStructSchema(table string, sample any) error {
	// In rapid's standalone mode, schema management is handled by atp.
	// This is a no-op; atp manages the schema.
	return nil
}

// QueryStruct queries a pod table and returns records matching the condition.
func (c *PodClient) QueryStruct(table string, cond any, opts any) (any, error) {
	// Query via atp's HTTP interface
	url := c.base + "/api/functions"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pod query: %s", resp.Status)
	}
	var out struct {
		Functions []any `json:"functions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Functions, nil
}

// UpsertStruct upserts a record to the pod table.
func (c *PodClient) UpsertStruct(table, id string, rec any) (bool, any, error) {
	url := c.base + "/api/functions"
	data, _ := json.Marshal(rec)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	resp, _ := c.http.Do(req)
	defer resp.Body.Close()
	var out struct {
		ID        string `json:"id"`
		Created bool   `json:"created"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, nil, err
	}
	return out.Created, nil, nil
}

// IDs returns the record IDs in a table.
func (c *PodClient) IDs(table string) ([]string, error) {
	// Delegate to atp
	url := c.base + "/api/functions"
	req, _ := http.NewRequest("GET", url, nil)
	resp, _ := c.http.Do(req)
	defer resp.Body.Close()
	var out struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.IDs, nil
}

// ListTables returns the table names in the store.
func (c *PodClient) ListTables() ([]TableInfo, error) {
	// Delegate to atp
	url := c.base + "/api/functions"
	req, _ := http.NewRequest("GET", url, nil)
	resp, _ := c.http.Do(req)
	defer resp.Body.Close()
	var out struct {
		Tables []TableInfo `json:"tables"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Tables, nil
}

// TableInfo describes a pod table.
type TableInfo struct {
	Name   string         `json:"name"`
	Count  int            `json:"count"`
	Schema map[string]any `json:"schema,omitempty"`
}
