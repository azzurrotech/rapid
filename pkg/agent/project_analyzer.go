package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ProjectAnalyzer walks a source tree and turns what it finds into a
// structured, reproducible analysis: file inventory, complexity, and the
// wrapper recommendations that follow from the languages and practices the
// project actually uses.
//
// Everything here is local: no network call is made and no model is consulted
// unless the caller explicitly loads one through ModelLoader.
type ProjectAnalyzer struct {
	modelLoader ModelLoader
	cacheDir    string
	maxTokens   int
	ollamaURL   string
	timeout     time.Duration
}

// AnalysisResult is the whole analysis of one project.
type AnalysisResult struct {
	ProjectName         string              `json:"project_name"`
	ProjectPath         string              `json:"project_path"`
	AnalysisDate        time.Time           `json:"analysis_date"`
	Files               []FileInfo          `json:"files"`
	TotalFiles          int                 `json:"total_files"`
	TotalSize           int64               `json:"total_size"`
	Complexity          float64             `json:"complexity"`
	HasTests            bool                `json:"has_tests"`
	HasCI               bool                `json:"has_ci"`
	Recommendations     []Recommendation    `json:"recommendations"`
	IntegrationProjects []IntegratedProject `json:"integration_projects"`
	GeneratedCode       []GeneratedCode     `json:"generated_code"`
	Pricing             PricingAnalysis     `json:"pricing"`
}

// FileInfo describes one source file in the inventory.
type FileInfo struct {
	Path       string  `json:"path"`
	Size       int64   `json:"size"`
	Extension  string  `json:"extension"`
	Type       string  `json:"type"`
	Complexity float64 `json:"complexity"`
	HasTests   bool    `json:"has_tests"`
	HasCI      bool    `json:"has_ci"`
}

// Recommendation is one actionable finding about the project.
type Recommendation struct {
	Priority        string  `json:"priority"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Type            string  `json:"type"`
	EstimatedTokens int     `json:"estimated_tokens"`
	EstimatedCost   float64 `json:"estimated_cost"`
}

// IntegratedProject says how the project maps onto an Azzurro platform module.
type IntegratedProject struct {
	Name              string   `json:"name"`
	Platform          string   `json:"platform"`
	IntegrationPoints []string `json:"integration_points"`
	Compatibility     float64  `json:"compatibility"`
}

// GeneratedCode names a wrapper source file the analysis proposes.
type GeneratedCode struct {
	File        string    `json:"file"`
	Language    string    `json:"language"`
	GeneratedAt time.Time `json:"generated_at"`
	Size        int64     `json:"size"`
	Description string    `json:"description"`
}

// PricingAnalysis is the estimated cost of turning the analysis into a
// wrapper, in the same units the pricing engine bills.
type PricingAnalysis struct {
	EstimatedTokens int       `json:"estimated_tokens"`
	EstimatedCost   float64   `json:"estimated_cost"`
	CacheEfficiency float64   `json:"cache_efficiency"`
	Currency        string    `json:"currency"`
	Timestamp       time.Time `json:"timestamp"`
}

// ProjectInfo is the template payload used by WrapperGenerator.
type ProjectInfo struct {
	Name      string    `json:"name"`
	Language  string    `json:"language"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Author    string    `json:"author"`
	Version   string    `json:"version"`
}

// NewProjectAnalyzer builds an analyzer. An empty Ollama URL falls back to the
// local default; maxTokens is clamped to a useful minimum.
func NewProjectAnalyzer(ollamaURL string, maxTokens int) (*ProjectAnalyzer, error) {
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	if maxTokens < 1000 {
		maxTokens = 4000
	}

	cacheDir := "./cache"
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("agent: create cache directory: %w", err)
	}

	loader := NewModelLoader("./models", cacheDir, ollamaURL, 30*time.Second)
	return &ProjectAnalyzer{
		modelLoader: *loader,
		cacheDir:    cacheDir,
		maxTokens:   maxTokens,
		ollamaURL:   ollamaURL,
		timeout:     30 * time.Second,
	}, nil
}

// AnalyzeProject inventories a directory and derives the analysis.
func (pa *ProjectAnalyzer) AnalyzeProject(projectPath string) (AnalysisResult, error) {
	info, err := os.Stat(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return AnalysisResult{}, fmt.Errorf("project path does not exist: %s", projectPath)
		}
		return AnalysisResult{}, fmt.Errorf("stat %s: %w", projectPath, err)
	}
	if !info.IsDir() {
		return AnalysisResult{}, fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	files, err := pa.getProjectFiles(projectPath)
	if err != nil {
		return AnalysisResult{}, fmt.Errorf("list project files: %w", err)
	}

	analysis := AnalysisResult{
		ProjectName:  filepath.Base(filepath.Clean(projectPath)),
		ProjectPath:  projectPath,
		AnalysisDate: time.Now().UTC(),
		Files:        files,
		TotalFiles:   len(files),
		TotalSize:    totalSize(files),
		Complexity:   averageComplexity(files),
		HasTests:     anyFile(files, func(f FileInfo) bool { return f.HasTests }),
		HasCI:        anyFile(files, func(f FileInfo) bool { return f.HasCI }),
	}
	analysis.Recommendations = generateRecommendations(files, analysis)
	analysis.IntegrationProjects = analyzeIntegrationProjects(files)
	analysis.GeneratedCode = generateCode(files)
	analysis.Pricing = pa.calculatePricing(analysis)
	return analysis, nil
}

// getProjectFiles walks the tree, skipping VCS/build directories, hidden
// entries and binary files, so the inventory describes source, not noise.
func (pa *ProjectAnalyzer) getProjectFiles(projectPath string) ([]FileInfo, error) {
	files := make([]FileInfo, 0, 64)
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// A dangling link or unreadable directory must not abort the walk.
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := filepath.Base(path)
		if info.IsDir() {
			if path == projectPath {
				return nil
			}
			switch name {
			case ".git", ".hg", ".svn", "node_modules", "vendor", "dist", "build", "target",
				".idea", ".vscode", "models", "cache", "generated_wrappers":
				return filepath.SkipDir
			}
			if strings.HasPrefix(name, ".") && name != ".github" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") && name != ".gitlab-ci.yml" && name != ".travis.yml" {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 || isBinary(info) {
			return nil
		}

		files = append(files, FileInfo{
			Path:       path,
			Size:       info.Size(),
			Extension:  strings.ToLower(filepath.Ext(path)),
			Type:       getFileType(path),
			Complexity: calculateComplexity(path, info),
			HasTests:   hasTests(path),
			HasCI:      hasCI(path),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func anyFile(files []FileInfo, pred func(FileInfo) bool) bool {
	for _, f := range files {
		if pred(f) {
			return true
		}
	}
	return false
}

func totalSize(files []FileInfo) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}

func averageComplexity(files []FileInfo) float64 {
	if len(files) == 0 {
		return 0
	}
	var total float64
	for _, f := range files {
		total += f.Complexity
	}
	return total / float64(len(files))
}

// hasTests reports whether a file name is a test or spec.
func hasTests(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(name, "_test.go") ||
		strings.HasSuffix(name, "test.py") ||
		strings.HasSuffix(name, "-test.js") ||
		strings.HasSuffix(name, "_test.ts") ||
		strings.Contains(name, ".test.") ||
		strings.Contains(name, ".spec.") ||
		strings.HasSuffix(name, "spec.rb")
}

// hasCI reports whether a file indicates a CI pipeline. Walk hands us the full
// path, so the check looks at the path segments rather than the base name.
func hasCI(path string) bool {
	p := filepath.ToSlash(path)
	switch {
	case strings.Contains(p, "/.github/workflows/") && (strings.HasSuffix(p, ".yml") || strings.HasSuffix(p, ".yaml")):
		return true
	case strings.HasSuffix(p, "/.gitlab-ci.yml"), strings.HasSuffix(p, "/.travis.yml"),
		strings.HasSuffix(p, "/appveyor.yml"), strings.HasSuffix(p, "/.circleci/config.yml"),
		strings.HasSuffix(p, "/azure-pipelines.yml"), strings.HasSuffix(p, "/Jenkinsfile"):
		return true
	}
	return false
}

// generateRecommendations turns the inventory into advice, one recommendation
// per concern, so the output does not repeat itself.
func generateRecommendations(files []FileInfo, analysis AnalysisResult) []Recommendation {
	recs := make([]Recommendation, 0, 5)

	if analysis.Complexity > 3.0 {
		recs = append(recs, Recommendation{
			Priority:        "HIGH",
			Title:           "High complexity project",
			Description:     "Consider breaking the project into smaller, independently testable components.",
			Type:            "architecture",
			EstimatedTokens: 2000,
			EstimatedCost:   5.0,
		})
	}

	lang := dominantLanguage(files)
	switch lang {
	case "Go":
		recs = append(recs, Recommendation{
			Priority:        "MEDIUM",
			Title:           "Go project detected",
			Description:     "Good opportunity for a Go wrapper with native ATP, POD and SHEPHERD integration.",
			Type:            "wrapper",
			EstimatedTokens: 1500,
			EstimatedCost:   3.0,
		})
	case "Python":
		recs = append(recs, Recommendation{
			Priority:        "MEDIUM",
			Title:           "Python project detected",
			Description:     "Consider a Python wrapper for data-processing and model-serving pipelines.",
			Type:            "wrapper",
			EstimatedTokens: 1000,
			EstimatedCost:   2.5,
		})
	case "JavaScript":
		recs = append(recs, Recommendation{
			Priority:        "MEDIUM",
			Title:           "JavaScript project detected",
			Description:     "Consider a Node.js wrapper for web and API surfaces.",
			Type:            "wrapper",
			EstimatedTokens: 1200,
			EstimatedCost:   2.8,
		})
	}

	if analysis.HasTests {
		recs = append(recs, Recommendation{
			Priority:        "HIGH",
			Title:           "Test files detected",
			Description:     "The existing tests give the wrapper a confidence baseline; keep them in the generated package.",
			Type:            "confidence",
			EstimatedTokens: 800,
			EstimatedCost:   1.5,
		})
	}
	if analysis.HasCI {
		recs = append(recs, Recommendation{
			Priority:        "MEDIUM",
			Title:           "CI pipeline detected",
			Description:     "Wire the generated wrapper into the existing pipeline through ATP's function registry.",
			Type:            "integration",
			EstimatedTokens: 1200,
			EstimatedCost:   2.8,
		})
	}
	if len(recs) == 0 {
		recs = append(recs, Recommendation{
			Priority:        "LOW",
			Title:           "Basic project",
			Description:     "Standard project analysis complete; no language-specific wrapper is indicated yet.",
			Type:            "general",
			EstimatedTokens: 500,
			EstimatedCost:   1.0,
		})
	}
	return recs
}

// dominantLanguage returns the language with the most source files, or "" when
// the project exposes no supported language.
func dominantLanguage(files []FileInfo) string {
	counts := map[string]int{}
	for _, f := range files {
		switch f.Extension {
		case ".go":
			counts["Go"]++
		case ".py":
			counts["Python"]++
		case ".js", ".ts", ".jsx", ".tsx":
			counts["JavaScript"]++
		}
	}
	best, bestN := "", 0
	for lang, n := range counts {
		if n > bestN || (n == bestN && lang < best) {
			best, bestN = lang, n
		}
	}
	return best
}

// analyzeIntegrationProjects maps the languages present onto the platform
// module that carries them, once per language rather than once per file.
func analyzeIntegrationProjects(files []FileInfo) []IntegratedProject {
	seen := map[string]bool{}
	out := make([]IntegratedProject, 0, 3)
	add := func(p IntegratedProject) {
		if seen[p.Name] {
			return
		}
		seen[p.Name] = true
		out = append(out, p)
	}
	for _, f := range files {
		switch f.Extension {
		case ".go":
			add(IntegratedProject{
				Name:     "Go Module",
				Platform: "POD",
				IntegrationPoints: []string{
					"Project metadata storage",
					"Analysis and wrapper records",
					"ATP function registration",
				},
				Compatibility: 0.95,
			})
		case ".py":
			add(IntegratedProject{
				Name:     "Python Package",
				Platform: "SONG",
				IntegrationPoints: []string{
					"Data-processing pipelines",
					"ML model integration",
					"API endpoint generation",
				},
				Compatibility: 0.90,
			})
		case ".js", ".ts", ".jsx", ".tsx":
			add(IntegratedProject{
				Name:     "Node.js Application",
				Platform: "ATP",
				IntegrationPoints: []string{
					"Web server orchestration",
					"API gateway integration",
					"Workflow automation",
				},
				Compatibility: 0.88,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// generateCode lists the wrapper files the analysis proposes, one per language
// actually present.
func generateCode(files []FileInfo) []GeneratedCode {
	now := time.Now().UTC()
	generated := make([]GeneratedCode, 0, 3)
	has := func(exts ...string) bool {
		for _, f := range files {
			for _, e := range exts {
				if f.Extension == e {
					return true
				}
			}
		}
		return false
	}
	if has(".go") {
		generated = append(generated, GeneratedCode{
			File: "business_wrapper.go", Language: "Go", GeneratedAt: now,
			Size: 2048, Description: "Business customization wrapper for a Go project",
		})
	}
	if has(".py") {
		generated = append(generated, GeneratedCode{
			File: "business_wrapper.py", Language: "Python", GeneratedAt: now,
			Size: 4096, Description: "Business customization wrapper for a Python project",
		})
	}
	if has(".js", ".ts", ".jsx", ".tsx") {
		generated = append(generated, GeneratedCode{
			File: "business_wrapper.js", Language: "JavaScript", GeneratedAt: now,
			Size: 3072, Description: "Business customization wrapper for a Node.js project",
		})
	}
	return generated
}

func (pa *ProjectAnalyzer) calculatePricing(analysis AnalysisResult) PricingAnalysis {
	tokens := analysis.TotalFiles * 500
	tokens = int(float64(tokens) * maxFloat(analysis.Complexity, 1))
	if tokens > pa.maxTokens {
		tokens = pa.maxTokens
	}
	efficiency := 0.8
	if analysis.HasTests {
		efficiency *= 1.1
	}
	if analysis.HasCI {
		efficiency *= 1.05
	}
	return PricingAnalysis{
		EstimatedTokens: tokens,
		EstimatedCost:   float64(tokens) * 0.002 / 1000,
		CacheEfficiency: efficiency,
		Currency:        "USD",
		Timestamp:       time.Now().UTC(),
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// calculateComplexity scores a file from its type and size.
func calculateComplexity(path string, info os.FileInfo) float64 {
	name := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(name, ".go"), strings.HasSuffix(name, ".py"),
		strings.HasSuffix(name, ".js"), strings.HasSuffix(name, ".ts"),
		strings.HasSuffix(name, ".jsx"), strings.HasSuffix(name, ".tsx"):
		switch {
		case info.Size() > 10*1024:
			return 3.0
		case info.Size() > 1024:
			return 2.0
		default:
			return 1.0
		}
	case strings.HasSuffix(name, ".md"):
		return 0.5
	case strings.HasSuffix(name, ".json"), strings.HasSuffix(name, ".yaml"), strings.HasSuffix(name, ".yml"):
		return 1.5
	case strings.HasSuffix(name, ".sh"):
		return 2.0
	default:
		return 1.0
	}
}

// getFileType classifies a path for the inventory.
func getFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	name := strings.ToLower(filepath.Base(path))
	switch {
	case ext == ".go":
		return "go"
	case ext == ".py":
		return "python"
	case ext == ".js", ext == ".ts", ext == ".jsx", ext == ".tsx":
		return "javascript"
	case ext == ".md":
		return "markdown"
	case ext == ".json":
		return "json"
	case ext == ".yaml", ext == ".yml":
		return "yaml"
	case ext == ".sh":
		return "shell"
	case name == "dockerfile" || strings.HasPrefix(name, "dockerfile."):
		return "dockerfile"
	case name == "docker-compose.yml" || name == "docker-compose.yaml":
		return "dockercompose"
	case name == "makefile" || strings.HasPrefix(name, "makefile."):
		return "makefile"
	case strings.Contains(name, "test") || strings.Contains(name, "spec"):
		return "test"
	case ext == "":
		return "file"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}

// isBinary uses the extension list, not a substring match: a README.md or a
// Makefile is text even though it has no source extension.
func isBinary(info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}
	switch strings.ToLower(filepath.Ext(info.Name())) {
	case ".exe", ".bin", ".dll", ".so", ".dylib", ".a", ".o",
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z",
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".ico", ".pdf",
		".mp3", ".mp4", ".mov", ".woff", ".woff2", ".ttf":
		return true
	}
	return false
}

// MarshalAnalysis is a convenience for callers that persist the full result.
func MarshalAnalysis(a AnalysisResult) (string, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
