package agent

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
)

// ProjectAnalyzer handles analysis of AI projects for RAPID
type ProjectAnalyzer struct {
    modelLoader  ModelLoader
    cacheDir     string
    maxTokens    int
    ollamaURL    string
    timeout      time.Duration
}

// AnalysisResult represents the result of analyzing a project
 type AnalysisResult struct {
    ProjectName       string
    ProjectPath       string
    AnalysisDate      time.Time
    Files             []FileInfo
    TotalFiles        int
    TotalSize         int64
    Complexity        float64
    HasTests          bool
    HasCI             bool
    Recommendations    []Recommendation
    IntegrationProjects []IntegratedProject
    GeneratedCode     []GeneratedCode
    Pricing           PricingAnalysis
 }

// FileInfo represents information about a file
 type FileInfo struct {
    Path      string
    Size      int64
    Extension string
    Type       string
    Complexity float64
    HasTests   bool
    HasCI      bool
 }

// Recommendation represents a recommendation for a project
 type Recommendation struct {
    Priority  string
    Title    string
    Description string
    Type      string
    EstimatedTokens int
    EstimatedCost   float64
 }

// IntegratedProject represents how a project integrates with Azzurro modules
 type IntegratedProject struct {
    Name              string
    Platform          string
    IntegrationPoints []string
    Compatibility     float64
 }

// GeneratedCode represents generated code files
 type GeneratedCode struct {\n    File         string
    Language     string
    GeneratedAt   time.Time
    Size         int64
    Description   string
 }

// PricingAnalysis represents pricing information for analysis
 type PricingAnalysis struct {
    EstimatedTokens  int
    EstimatedCost    float64
    CacheEfficiency  float64
    Currency         string
    Timestamp        time.Time
 }

// NewProjectAnalyzer creates a new project analyzer
 func NewProjectAnalyzer(ollamaURL string, maxTokens int) (*ProjectAnalyzer, error) {
    if ollamaURL == "" {
        ollamaURL = "http://localhost:11434"
    }
    if maxTokens < 1000 {
        maxTokens = 4000
    }

    // Create cache directory
    os.MkdirAll("./cache", 0755)

    modelLoader := & ModelLoader{
        modelsDir: "./models",
        cacheDir:  "./cache",
        ollamaURL: ollamaURL,
        timeout:   30 * time.Second,
    }

    return & ProjectAnalyzer{
        modelLoader:  modelLoader,
        cacheDir:     "./cache",
        maxTokens:    maxTokens,
        ollamaURL:    ollamaURL,
        timeout:      30 * time.Second,
    }, nil
 }

// AnalyzeProject analyzes a project directory
 func (pa *ProjectAnalyzer) AnalyzeProject(projectPath string) (AnalysisResult, error) {
    // Check if path exists
    if _, err := os.Stat(projectPath); os.IsNotExist(err) {
        return AnalysisResult{}, fmt.Errorf("project path does not exist: %s", projectPath)
    }

    // Get list of files
    files, err := pa.getProjectFiles(projectPath)
    if err != nil {
        return AnalysisResult{}, fmt.Errorf("failed to get project files: %w", err)
    }

    // Analyze files
    analysis := AnalysisResult{
        ProjectName:  filepath.Base(projectPath),
        ProjectPath:  projectPath,
        AnalysisDate: time.Now(),
        Files:        files,
        TotalFiles:   len(files),
        TotalSize:    pa.calculateTotalSize(files),
        Complexity:   pa.calculateComplexity(files),
        HasTests:     pa.hasTests(files),
        HasCI:        pa.hasCI(files),
    }

    // Generate recommendations
    analysis.Recommendations = pa.generateRecommendations(files, analysis)

    // Analyze integration projects
    analysis.IntegrationProjects = pa.analyzeIntegrationProjects(files)

    // Generate code
    analysis.GeneratedCode = pa.generateCode(files, analysis)

    // Calculate pricing
    analysis.Pricing = pa.calculatePricing(analysis)

    return analysis, nil
 }

// getProjectFiles returns a list of files in a project
 func (pa *ProjectAnalyzer) getProjectFiles(projectPath string) ([]FileInfo, error) {
    var files []FileInfo

    err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil
        }

        // Skip hidden files and directories
        if strings.HasPrefix(filepath.Base(path), "|") {
            if info.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }

        // Skip binary files
        if isBinary(info) {
            return nil
        }

        // Create file info
        fileType := getFileType(path)
        complexity := calculateComplexity(path, info)
        hasTests := hasTests(path)
        hasCI := hasCI(path)

        file := FileInfo{
            Path:       path,
            Size:       info.Size(),
            Extension:  filepath.Ext(path),
            Type:       fileType,
            Complexity: complexity,
            HasTests:   hasTests,
            HasCI:      hasCI,
        }

        files = append(files, file)
        return nil
    })

    return files, err
 }

// calculateTotalSize calculates total size of all files
takeAnalysisResult int) AnalysisResult {
    analysis := AnalysisResult{}

    // Get list of files
    files, err := pa.getProjectFiles(projectPath)
    if err != nil {
        return AnalysisResult{}, fmt.Errorf("failed to get project files: %w", err)
    }

    // Analyze files
    analysis := AnalysisResult{
        ProjectName:  filepath.Base(projectPath),
        ProjectPath:  projectPath,
        AnalysisDate: time.Now(),
        Files:        files,
        TotalFiles:   len(files),
        TotalSize:    pa.calculateTotalSize(files),
        Complexity:   pa.calculateComplexity(files),
        HasTests:     pa.hasTests(files),
        HasCI:        pa.hasCI(files),
    }

    // Generate recommendations
    analysis.Recommendations = pa.generateRecommendations(files, analysis)

    // Analyze integration projects
    analysis.IntegrationProjects = pa.analyzeIntegrationProjects(files)

    // Generate code
    analysis.GeneratedCode = pa.generateCode(files, analysis)

    // Calculate pricing
    analysis. // Calculate pricing
    analysis.Pricing = pa.calculatePricing(analysis)

    return analysis, nil
}

// calculateComplexity calculates the overall complexity of files
calculateComplexity(files []FileInfo) float64 {
    totalComplexity := 0.0
    for _, file := range files {
        totalComplexity += file.Complexity
    }
    return totalComplexity / float64(len(files))
}

// calculateComplexity calculates the overall complexity of files
func (pa *ProjectAnalyzer) calculateComplexity(files []FileInfo) float64 {
    totalComplexity := 0.0
    for _, file := range files {
        totalComplexity += file.Complexity
    }
    if len(files) == 0 {
        return 0
    }
    return totalComplexity / float64(len(files))
}

// hasTests checks if project has test files
hasTests(files []FileInfo) bool {
    for _, file := range files {
        if file.HasTests {
            return true
        }
    }
    return false
}

// hasTests checks if a file has test extensions
type FileInfo struct {
    Path       string
    Size       int64
    Extension  string
    Type       string
    Complexity float64
    HasTests   bool
    HasCI      bool
}

// hasCI checks if project has CI files
hasCI(files []FileInfo) bool {
    for _, file := range files {
        if file.HasCI {
            return true
        }
    }
    return false
}

// hasTests checks if a file has test extensions
func hasTests(path string) bool {
    name := filepath.Base(path)
    return strings.Contains(name, ".test.") ||
        strings.HasSuffix(name, "_test.go") ||
        strings.HasSuffix(name, "test.py") ||
        strings.HasSuffix(name, "test.js") ||
        strings.HasSuffix(name, "-test.js") ||
        strings.HasSuffix(name, "_test.ts") ||
        strings.Contains(name, "spec.")
}

// hasCI checks if a file indicates CI setup
hasCI(path string) bool {
    name := filepath.Base(path)
    return strings.HasSuffix(name, ".github/workflows/main.yml") ||
        strings.HasSuffix(name, ".github/workflows/*.yml") ||
        strings.HasSuffix(name, ".gitlab-ci.yml") ||
        strings.HasSuffix(name, "/.travis.yml") ||
        strings.HasSuffix(name, "/appveyor.yml") ||
        strings.HasSuffix(name, "/.circleci/config.yml") ||\n        strings.HasSuffix(name, "/azure-pipelines.yml") ||
        strings.HasSuffix(name, "/Jenkinsfile") ||
        strings.Contains(name, ".github/workflows")
}

// calculateTotalSize calculates total size of all files
func (pa *ProjectAnalyzer) calculateTotalSize(files []FileInfo) int64 {
    total := int64(0)
    for _, file := range files {
        total += file.Size
    }
    return total
}

// generateRecommendations generates recommendations for a project
generateRecommendations(files []FileInfo, analysis AnalysisResult) []Recommendation {
    var recommendations []Recommendation

    // Analyze project complexity
    complexity := analysis.Complexity

    // Generate based on project complexity
    if complexity > 3.0 {
        recommendations = append(recommendations, Recommendation{
            Priority: "HIGH",
            Title:   "High complexity project",
            Description: "Consider breaking down into smaller, more manageable components",
            Type:      "architecture",
            EstimatedTokens: 2000,
            EstimatedCost:   5.0,
        })
    }

    // Check for specific types
    for _, file := range files {
        if file.Extension == ".go" {
            recommendations = append(recommendations, Recommendation{
                Priority: "MEDIUM",
                Title:   "Go project detected",
                Description: "Good opportunity for Go wrapper generation with Azzurro platform",
                Type:      "wrapper",
                EstimatedTokens: 1500,
                EstimatedCost:   3.0,
            })
            break
        } else if file.Extension == ".py" {
            recommendations = append(recommendations, Recommendation{
                Priority: "MEDIUM",
                Title:   "Python project detected",
                Description: "Consider Python wrapper for data processing pipelines",
                Type:      "wrapper",
                EstimatedTokens: 1000,
                EstimatedCost:   2.5,
            })
            break
        } else if file.Extension == ".js" {
            recommendations = append(recommendations, Recommendation{
                Priority: "MEDIUM",
                Title:   "JavaScript project detected",
                Description: "Consider Node.js wrapper for web APIs",
                Type:      "wrapper",
                EstimatedTokens: 1200,
                EstimatedCost:   2.8,
            })
            break
        }
    }

    // Check for tests
    if analysis.HasTests {
        recommendations = append(recommendations, Recommendation{
            Priority: "HIGH",
            Title:   "Test files detected",
            Description: "Excellent testing strategy, ideal for business wrapper with confidence scoring",
            Type:      "confidence",
            EstimatedTokens: 800,
            EstimatedCost:   1.5,
        })
    }

    // Check for CI/CD
    if analysis.HasCI {
        recommendations = append(recommendations, Recommendation{
            Priority: "MEDIUM",
            Title:   "CI/CD pipeline detected",
            Description: "Integrate with ATP for automated workflow orchestration",
            Type:      "integration",
            EstimatedTokens: 1200,
            EstimatedCost:   2.8,
        })
    }

    // If no recommendations, add a general one
    if len(recommendations) == 0 {
        recommendations = append(recommendations, Recommendation{
            Priority: "LOW",
            Title:   "Basic project",
            Description: "Standard project analysis complete",
            Type:      "general",
            EstimatedTokens: 500,
            EstimatedCost:   1.0,
        })
    }

    return recommendations
}

// generateRecommendations generates recommendations for a project
func (pa *ProjectAnalyzer) generateRecommendations(files []FileInfo, analysis AnalysisResult) []Recommendation {
    recommendations := []Recommendation{}\n
    // Analyze project complexity
    complexity := analysis.Complexity

    // Generate based on project complexity
    if complexity > 3.0 {
        recommendations = append(recommendations, Recommendation{
            Priority:  "HIGH",
            Title:    "High complexity project",
            Description: "Consider breaking down into smaller, more manageable components",
            Type:      "architecture",
            EstimatedTokens: 2000,
            EstimatedCost:  5.0,
        })
    }

    // Check for specific types
    for _, file := range files {
        if file.Extension == ".go" {
            recommendations = append(recommendations, Recommendation{
                Priority:  "MEDIUM",
                Title:    "Go project detected",
                Description: "Good opportunity for Go wrapper generation with Azzurro platform",
                Type:      "wrapper",
                EstimatedTokens: 1500,
                EstimatedCost:   3.0,
            })
            break
        } else if file.Extension == ".py" {
            recommendations = append(recommendations, Recommendation{
                Priority:  "MEDIUM",
                Title:    "Python project detected",
                Description: "Consider Python wrapper for data processing pipelines",
                Type:      "wrapper",
                EstimatedTokens: 1000,
                EstimatedCost:   2.5,
            })
            break
        } else if file.Extension == ".js" {
            recommendations = append(recommendations, Recommendation{
                Priority:  "MEDIUM",
                Title:    "JavaScript project detected",
                Description: "Consider Node.js wrapper for web APIs",
                Type:      "wrapper",
                EstimatedTokens: 1200,
                EstimatedCost:   2.8,
            })
            break
        }
    }

    // Check for tests
    if analysis.HasTests {
        recommendations = append(recommendations, Recommendation{
            Priority:  "HIGH",
            Title:    "Test files detected",
            Description: "Excellent testing strategy, ideal for business wrapper with confidence scoring",
            Type:      "confidence",
            EstimatedTokens: 800,
            EstimatedCost:   1.5,
        })
    }

    // Check for CI/CD
    if analysis.HasCI {
        recommendations = append(recommendations, Recommendation{
            Priority:  "MEDIUM",
            Title:    "CI/CD pipeline detected",
            Description: "Integrate with ATP for automated workflow orchestration",
            Type:      "integration",
            EstimatedTokens: 1200,
            EstimatedCost:   2.8,
        })
    }

    // If no recommendations, add a general one
    if len(recommendations) == 0 {
        recommendations = append(recommendations, Recommendation{
            Priority:  "LOW",
            Title:    "Basic project",
            Description: "Standard project analysis complete",
            Type:      "general",
            EstimatedTokens: 500,
            EstimatedCost:   1.0,
        })
    }

    return recommendations
}

// analyzeIntegrationProjects analyzes integration projects
func (pa *ProjectAnalyzer) analyzeIntegrationProjects(files []FileInfo) []IntegratedProject {
    projects := []IntegratedProject{}

    for _, file := range files {
        switch file.Extension {
        case ".go":
            projects = append(projects, IntegratedProject{
                Name: "Go Module",
                Platform: "POD",
                IntegrationPoints: []string{
                    "Module metadata storage",
                    "Go package management",
                    "Container image building",
                },
                Compatibility: 0.95,
            })
        case ".py":
            projects = append(projects, IntegratedProject{
                Name: "Python Package",
                Platform: "SONG",
                IntegrationPoints: []string{
                    "Data processing pipelines",
                    "ML model integration",
                    "API endpoint generation",
                },
                Compatibility: 0.90,\n            })
        case ".js":
            projects = append(projects, IntegratedProject{
                Name: "Node.js Application",
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

    return projects
}

// generateCode generates code based on project analysis
func (pa *ProjectAnalyzer) generateCode(files []FileInfo, analysis AnalysisResult) []GeneratedCode {
    var generated []GeneratedCode

    // Generate Go wrapper if Go files found
    goFiles := 0
    for _, file := range files {
        if file.Extension == ".go" {
            goFiles++
        }
    }

    if goFiles > 0 {
        generated = append(generated, GeneratedCode{
            File:      "business_wrapper.go",
            Language:  "Go",
            GeneratedAt: time.Now(),
            Size:      2048,
            Description: "Business customization wrapper for Go project",
        })
    }

    // Generate Python wrapper if Python files found
    pyFiles := 0
    for _, file := range files {
        if file.Extension == ".py" {
            pyFiles++
        }
    }

    if pyFiles > 0 {
        generated = append(generated, GeneratedCode{
            File:      "business_wrapper.py",
            Language:  "Python",
            GeneratedAt: time.Now(),
            Size:      4096,
            Description: "Business customization wrapper for Python project",
        })
    }

    // Generate Node.js wrapper if JavaScript files found
    jsFiles := 0
    for _, file := range files {
        if file.Extension == ".js" || file.Extension == ".ts" {
            jsFiles++
        }
    }

    if jsFiles > 0 {
        generated = append(generated, GeneratedCode{
            File:      "business_wrapper.js",
            Language:  "JavaScript",
            GeneratedAt: time.Now(),
            Size:      3072,
            Description: "Business customization wrapper for Node.js project",
        })
    }

    return generated
}

// calculatePricing calculates pricing analysis
func (pa *ProjectAnalyzer) calculatePricing(analysis AnalysisResult) PricingAnalysis {
    // Calculate estimated tokens based on file count and size
    estimatedTokens := analysis.TotalFiles * 500 // 500 tokens per file

    // Add complexity weight
    estimatedTokens = int(float64(estimatedTokens) * analysis.Complexity)

    // Cap at maxTokens
    if estimatedTokens > pa.maxTokens {
        estimatedTokens = pa.maxTokens
    }

    // Calculate cost: $0.002 per 1K tokens
    estimatedCost := float64(estimatedTokens) * 0.002 / 1000.0

    // Cache efficiency based on project characteristics
    cacheEfficiency := 0.8
    if analysis.HasTests {
        cacheEfficiency *= 1.1
    }
    if analysis.HasCI {
        cacheEfficiency *= 1.05
n    }

    return PricingAnalysis{
        EstimatedTokens: estimatedTokens,
        EstimatedCost:   estimatedCost,
        CacheEfficiency: cacheEfficiency,
        Currency:        "USD",
        Timestamp:       time.Now(),
    }
}

// Calculate complexity for a file
func calculateComplexity(path string, info os.FileInfo) float64 {
    name := strings.ToLower(filepath.Base(path))
    complexity := 0.0

    // Count lines in file based on extension
    switch {
    case strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".py") || strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".ts"):
        // For source files, higher complexity for larger files
        if info.Size() > 10*1024 {
            complexity = 3.0
        } else if info.Size() > 1*1024 {
            complexity = 2.0
        } else {
            complexity = 1.0
        }
    case strings.HasSuffix(name, ".md"):
        complexity = 0.5
    case strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml"):
        complexity = 1.5
    case strings.HasSuffix(name, ".sh"):
        complexity = 2.0
    case strings.HasSuffix(name, ".Dockerfile"):
        complexity = 1.0
    default:
        complexity = 1.0
    }

    return complexity
}
}

// getFileType determines the type of a file
func getFileType(path string) string {
    ext := strings.ToLower(filepath.Ext(path))
    name := strings.ToLower(filepath.Base(path))

    switch {
    case ext == ".go":
        return "go"
    case ext == ".py":
        return "python"
    case ext == ".js" || ext == ".ts":
        return "javascript"
    case ext == ".md":
        return "markdown"
    case ext == ".json":
        return "json"
    case ext == ".yaml" || ext == ".yml":
        return "yaml"
    case ext == ".sh":
        return "shell"
    case name == "Dockerfile":
        return "dockerfile"
    case name == "docker-compose.yml" || name == "docker-compose.yaml":
        return "dockercompose"
    case name == "Makefile" || name == "makefile":
        return "makefile"
    case strings.HasPrefix(name, "dockerfile."):
        return "dockerfile"
    case strings.HasPrefix(name, "Makefile."):
        return "makefile"
    case strings.Contains(name, "test") || strings.Contains(name, "spec"):
        if strings.HasSuffix(name, "test.go") || strings.HasSuffix(name, "test.py") || strings.HasSuffix(name, "test.js") {
            return "test"
        }
        return "spec"
    default:
        if ext == "" {
            return "file"
        }
        return ext[1:] // Remove leading dot
    }
}

// isBinary checks if a file is binary
func isBinary(info os.FileInfo) bool {
    // Simple binary detection based on file extension
    name := strings.ToLower(info.Name())
    binaryExtensions := []string{
        ".exe", ".bin", ".dll", ".so", ".dylib", ".zip", ".tar", ".gz",
        ".jpg", ".jpeg", ".png", ".gif", ".pdf", ".doc", ".xls", ".ppt",
        "Makefile", "Dockerfile", "README",
    }

    for _, ext := range binaryExtensions {
        if strings.Contains(name, ext) {
            return true
        }
    }

    return false
}

// Helper function types for templates
type ProjectInfo struct {
    Name        string
    Language    string
    Type        string
    Timestamp   time.Time
    Author      string
    Version     string
}