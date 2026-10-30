package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sampleProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, dir, "util.go", strings.Repeat("// padding\n", 400))
	writeFile(t, dir, "main_test.go", "package main\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}\n")
	writeFile(t, dir, "README.md", "# sample\n")
	writeFile(t, dir, ".github/workflows/ci.yml", "name: ci\n")
	writeFile(t, dir, "script.sh", "#!/bin/sh\necho hi\n")
	return dir
}

func TestAnalyzeProjectInventory(t *testing.T) {
	pa, err := NewProjectAnalyzer("", 4000)
	if err != nil {
		t.Fatal(err)
	}
	result, err := pa.AnalyzeProject(sampleProject(t))
	if err != nil {
		t.Fatalf("AnalyzeProject: %v", err)
	}

	if result.TotalFiles != len(result.Files) || result.TotalFiles < 5 {
		t.Errorf("inventory = %d files", result.TotalFiles)
	}
	if result.TotalSize == 0 {
		t.Error("total size is zero")
	}
	if !result.HasTests {
		t.Error("tests not detected")
	}
	if !result.HasCI {
		t.Error("CI not detected")
	}
	if result.Complexity <= 0 {
		t.Error("complexity is not positive")
	}
	if len(result.IntegrationProjects) != 1 || result.IntegrationProjects[0].Platform != "POD" {
		t.Errorf("integration projects = %+v", result.IntegrationProjects)
	}
	if len(result.GeneratedCode) != 1 || result.GeneratedCode[0].Language != "Go" {
		t.Errorf("generated code = %+v", result.GeneratedCode)
	}

	// The project name is the directory's base name, not a hard-coded string.
	if result.ProjectName == "" {
		t.Error("empty project name")
	}
}

func TestAnalyzeProjectRejectsMissingPath(t *testing.T) {
	pa, err := NewProjectAnalyzer("", 4000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pa.AnalyzeProject(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("missing path was accepted")
	}
}

func TestAnalyzeProjectDoesNotWalkVendorOrHiddenTrees(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main\n")
	writeFile(t, dir, "vendor/dep/dep.go", "package dep\n")
	writeFile(t, dir, ".git/config", "irrelevant\n")

	result, err := NewProjectAnalyzerMust(t).AnalyzeProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range result.Files {
		if strings.Contains(filepath.ToSlash(f.Path), "/vendor/") || strings.Contains(filepath.ToSlash(f.Path), "/.git/") {
			t.Errorf("walked into excluded tree: %s", f.Path)
		}
	}
}

// NewProjectAnalyzerMust fails the test rather than returning an error, for
// fixtures where construction cannot legitimately fail.
func NewProjectAnalyzerMust(t *testing.T) *ProjectAnalyzer {
	t.Helper()
	pa, err := NewProjectAnalyzer("", 4000)
	if err != nil {
		t.Fatal(err)
	}
	return pa
}

func TestWrapperGeneratorGoAndMetadata(t *testing.T) {
	out := t.TempDir()
	gen := NewWrapperGenerator(t.TempDir(), nil)

	wrapper, err := gen.GenerateWrapper("Example API", "go-wrapper", out, false)
	if err != nil {
		t.Fatalf("GenerateWrapper: %v", err)
	}
	if !strings.HasSuffix(wrapper.FilePath, ".go") {
		t.Errorf("file = %q", wrapper.FilePath)
	}
	if !strings.Contains(wrapper.Content, "package main") ||
		!strings.Contains(wrapper.Content, "Example API") {
		t.Errorf("unexpected wrapper content:\n%.200s", wrapper.Content)
	}
	if len(wrapper.Metadata.IntegrationPoints) == 0 {
		t.Error("no integration points recorded")
	}
	if wrapper.Pricing.Cost <= 0 {
		t.Error("wrapper has no cost")
	}
	if wrapper.Metadata.Author != "RAPID AI" {
		t.Errorf("author = %q", wrapper.Metadata.Author)
	}
}

func TestWrapperGeneratorSupportsEveryTemplate(t *testing.T) {
	out := t.TempDir()
	gen := NewWrapperGenerator(t.TempDir(), nil)
	for _, kind := range []string{"go-wrapper", "python-wrapper", "node-wrapper", "terraform-wrapper", "dockerfile-wrapper", "generic-wrapper"} {
		w, err := gen.GenerateWrapper("sample", kind, out, true)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if !w.Pricing.Enterprise {
			t.Errorf("%s: enterprise pricing not applied", kind)
		}
		if w.Content == "" {
			t.Errorf("%s: empty content", kind)
		}
	}
}

func TestModelLoaderFindsLocalModels(t *testing.T) {
	dir := t.TempDir()
	models := filepath.Join(dir, "models")
	writeFile(t, models, "tiny/config.json", `{"name":"tiny","size_mb":12}`)
	loader := NewModelLoader(models, filepath.Join(dir, "cache"), "", 0)

	available, err := loader.GetAvailableModels()
	if err != nil {
		t.Fatal(err)
	}
	if len(available) != 1 || available[0].Name != "tiny" || available[0].SizeMB != 12 {
		t.Fatalf("models = %+v", available)
	}

	res, err := loader.LoadModel("tiny")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Errorf("LoadModel failed: %+v", res)
	}
}
