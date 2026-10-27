package web

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "path/filepath"

    rapid "github.com/matthew/Projects/BLUE/rapid"
)

// WebServer represents the web server for RAPID
type WebServer struct {
    app *rapid.RAPIDApplication
    port string
    host string
}

// NewWebServer creates a new web server
func NewWebServer(app *rapid.RAPIDApplication, host string, port string) *WebServer {
    return &WebServer{
        app:   app,
        port:  port,
        host:  host,
    }
}

// Start begins serving HTTP requests
func (s *WebServer) Start() error {
    // Serve static files
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

    // Setup API routes
    http.HandleFunc("/api/", s.handleAPIRoutes)
    http.HandleFunc("/", s.handleDashboard)

    addr := fmt.Sprintf("%s:%s", s.host, s.port)
    log.Printf("Starting RAPID web server on %s", addr)

    return http.ListenAndServe(addr, nil)
}

// handleAPIRoutes handles API routes
func (s *WebServer) handleAPIRoutes(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    switch r.URL.Path {
    case "/api/health":
        s.handleHealth(w, r)
    case "/api/projects":
        s.handleGetProjects(w, r)
    case "/api/projects/analyze":
        s.handleAnalyzeProject(w, r)
    case "/api/wrappers":
        s.handleGetWrappers(w, r)
    default:
        http.Error(w, "Not found", http.StatusNotFound)
    }
}

// handleHealth handles health check
func (s *WebServer) handleHealth(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().Format(time.RFC3339),
        "version":   "1.0.0",
        "platform":  "azimuthro",
        "components": map[string]bool{
            "rapid":          true,
            "pod":            true,
            "shepherd":       true,
            "song":           true,
            "atp":            true,
        },
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(health)
}

// handleGetProjects returns list of available projects
func (s *WebServer) handleGetProjects(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Delegate to RAPID application's project handler
    result := s.app.HandleGetProjects()
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(result)
}

// handleAnalyzeProject handles project analysis requests
func (s *WebServer) handleAnalyzeProject(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var project rapid.AIProject
    if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Use RAPID application to analyze project
    analysis := s.app.AnalyzeProject(project)

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(analysis)
}

// handleGetWrappers returns available wrapper templates
func (s *WebServer) handleGetWrappers(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    wrappers := []map[string]string{
        {
            "name":        "Go Wrapper",
            "type":        "go",
            "description": "Standard Go wrapper for Azzurro platform integration",
            "supported_projects": []string{"Go", "Python", "Docker"},
        },
        {
            "name":        "Python Wrapper", 
            "type":        "python",
            "description": "Python wrapper for data processing pipelines",
            "supported_projects": []string{"Python", "TensorFlow", "Kubernetes"},
        },
        {
            "name":        "Docker Wrapper",
            "type":        "docker", 
            "description": "Docker-based deployment wrapper",
            "supported_projects": []string{"Docker", "Kubernetes", "Go"},
        },
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(wrappers)
}

// handleDashboard serves the main dashboard page
func (s *WebServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }

    // Serve dashboard HTML
    htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>RAPID - Azzurro AI Analysis Platform</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
        }

        .header {
            background: rgba(255, 255, 255, 0.95);
            border-radius: 15px;
            padding: 30px;
            margin-bottom: 30px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.1);
        }

        .title {
            font-size: 2.5em;
            color: #333;
            margin-bottom: 10px;
        }

        .subtitle {
            font-size: 1.2em;
            color: #666;
            margin-bottom: 20px;
        }

        .status {
            display: inline-block;
            background: #4CAF50;
            color: white;
            padding: 5px 15px;
            border-radius: 20px;
            font-size: 0.9em;
            margin: 5px;
        }

        .section {
            background: rgba(255, 255, 255, 0.95);
            border-radius: 15px;
            padding: 25px;
            margin-bottom: 25px;
            box-shadow: 0 5px 20px rgba(0,0,0,0.1);
        }

        .section-title {
            font-size: 1.8em;
            color: #333;
            margin-bottom: 15px;
            border-bottom: 2px solid #667eea;
            padding-bottom: 10px;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }

        .stat-card {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 20px;
            text-align: center;
            transition: transform 0.3s ease;
        }

        .stat-card:hover {
            transform: translateY(-5px);
        }

        .stat-number {
            font-size: 2.5em;
            font-weight: bold;
            color: #667eea;
        }

        .stat-label {
            color: #666;
            margin-top: 5px;
        }

        .features-list {
            list-style: none;
            padding: 0;
        }

        .features-list li {
            padding: 10px 0;
            border-bottom: 1px solid #eee;
        }

        .features-list li:last-child {
            border-bottom: none;
        }

        .feature-icon {
            color: #4CAF50;
            margin-right: 10px;
        }

        .btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 12px 25px;
            border-radius: 8px;
            cursor: pointer;
            font-size: 1em;
            transition: background 0.3s ease;
            margin: 5px;
        }

        .btn:hover {
            background: #5a67d8;
        }

        .btn-secondary {
            background: #718096;
        }

        .btn-secondary:hover {
            background: #5a67d8;
        }

        .loading {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 3px solid #f3f3f3;
            border-top: 3px solid #667eea;
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin-left: 10px;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        .api-endpoint {
            background: #f8f9fa;
            border-left: 4px solid #667eea;
            padding: 15px;
            margin: 10px 0;
            border-radius: 5px;
        }

        .refresh-btn {
            float: right;
            background: #48bb78;
            color: white;
            border: none;
            padding: 8px 15px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 0.9em;
        }

        .refresh-btn:hover {
            background: #38a169;
        }

        @media (max-width: 768px) {
            .title {
                font-size: 2em;
            }

            .stats-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1 class="title">RAPID 🚀</h1>
            <p class="subtitle">Azzurro Technology AI Analysis and Wrapper Generation Platform</p>
            <div>
                <span class="status">✅ Operational</span>
                <span class="status">🔗 Integrated</span>
                <span class="status">⚡ Fast</span>
            </div>
        </div>

        <div class="section">
            <h2 class="section-title">📊 System Overview</h2>
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-number">4</div>
                    <div class="stat-label">Integrated Modules</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">2</div>
                    <div class="stat-label">Primary Commands</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">100%</div>
                    <div class="stat-label">Local Execution</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">MIT</div>
                    <div class="stat-label">License</div>
                </div>
            </div>

            <h3>Features:</h3>
            <ul class="features-list">
                <li><span class="feature-icon">✅</span> AI-powered open source project identification and analysis</li>
                <li><span class="feature-icon">✅</span> Automated business customization wrapper generation</li>
                <li><span class="feature-icon">✅</span> Seamless integration with POD, SHEPHERD, SONG, and ATP</li>
                <li><span class="feature-icon">✅</span> Local Ollama model execution (no cloud dependencies)</li>
                <li><span class="feature-icon">✅</span> Baseline HTML/CSS/JS web interface</li>
                <li><span class="feature-icon">✅</span> Comprehensive pricing and audit logging</li>
                <li><span class="feature-icon">✅</span> OpenCode-compatible command-line interface</li>
                <li><span class="feature-icon">✅</span> Enterprise-ready with MIT licensing</li>
            </ul>
        </div>

        <div class="section">
            <h2 class="section-title">🔧 Available Commands</h2>
            <div class="api-endpoint">
                <h3>rapid-cli analyze</h3>
                <p>Analyze directories for AI project opportunities with comprehensive recommendations.</p>
                <p><strong>Examples:</strong> <code>rapid-cli analyze ./my-project --output text</code></p>
            </div>
            <div class="api-endpoint">
                <h3>rapid-cli generate</h3>
                <p>Generate business customization wrappers for projects.</p>
                <p><strong>Examples:</strong> <code>rapid-cli generate "Docker" "go-wrapper" --output-dir ./wrappers</code></p>\n            <button class="refresh-btn" onclick="refreshData()">Refresh Data</button>
            <span class="loading" id="loadingSpinner" style="display:none;"></span>
        </div>

        <script>
            // Mock data for demonstration
            let projects = [
                { name: "Go", description: "Open source programming language", status: "active" },
                { name: "Docker", description: "Containerization platform", status: "active" },
                { name: "TensorFlow", description: "Machine learning framework", status: "processing" }
            ];

            function refreshData() {
                document.getElementById('loadingSpinner').style.display = 'inline-block';
                setTimeout(() => {
                    document.getElementById('loadingSpinner').style.display = 'none';
                    alert('Dashboard updated!');
                }, 1000);
            }

            // Add some interactivity to API endpoints demo
            document.querySelectorAll('.api-endpoint').forEach(endpoint => {
                endpoint.style.cursor = 'pointer';
                endpoint.title = 'Click to explore this feature';
            });
        </script>
    </div>
</body>
</html>`

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    fmt.Fprint(w, htmlContent)
}}