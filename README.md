# RAPID - Azzurro Technology AI Analysis and Wrapper Generation

## Overview

**RAPID** is Azzurro Technology's primary AI analysis and business customization wrapper generation tool. It automates the identification of AI projects within source code repositories and generates comprehensive business customization wrappers that seamlessly integrate with the Azzurro platform ecosystem.

## Key Features

- **AI-powered project identification**: Automatically detects and analyzes open source projects for AI opportunities
- **Business customization wrapper generation**: Creates production-ready wrapper solutions for identified projects
- **Seamless Azzurro integration**: Works with POD, SHEPHERD, SONG, and ATP modules
- **Local Ollama execution only**: No cloud dependencies, fully self-contained
- **Baseline implementation**: Built with standard HTML/CSS/JavaScript (no frameworks)
- **Enterprise licensing**: MIT licensed for commercial use

## Architecture

### Core Components

1. **CLI Interface** (`cmd/rapid-cli/`)
   - `rapid-cli analyze`: Analyze directories for AI project opportunities
   - `rapid-cli generate`: Generate business customization wrappers
   - `rapid-cli complete`: Complete multi-step workflows

2. **Web Interface** (`pkg/web/`)
   - RESTful API for project analysis and wrapper generation
   - Real-time monitoring dashboard
   - Integration with all Azzurro modules

3. **AI Engine** (`pkg/agent/`)
   - `model_loader.go`: Ollama model management and loading
   - `project_analyzer.go`: Project analysis and recommendation generation
   - `wrapper_generator.go`: Business customization wrapper generation

4. **Pricing Engine** (`pkg/pricing/`)
   - `calculator.go`: Cost calculation and breakdown
   - `logger.go`: Transaction logging and auditing

### Technical Specifications

- **Language**: Go
- **Dependencies**: Go standard library only (no third-party)
- **Web Interface**: Baseline HTML/CSS/JS (no frameworks)
- **License**: MIT
- **Execution**: Local Ollama models only

## Installation and Setup

### Prerequisites

- Go 1.21+
- Ollama service (http://localhost:11434)
- Docker (optional, for testing)

### Building and Running

```bash
# Clone the repository
cd /path/to/rapid

# Build the CLI binary
go build -o rapid-cli cmd/rapid-cli/main.go

# Create necessary directories
mkdir -p ./models ./cache ./templates ./generated_wrappers

# Test the CLI
./rapid-cli analyze ./path/to/your/project --output text

# Generate a wrapper
./rapid-cli generate "MyProject" "go-wrapper" --output-dir ./business-wrappers
```

### Web Server

```bash
# Start the web server
go run ./cmd/rapid-server/main.go

# Access the dashboard at http://localhost:8080
```

## Commands

### rapid-cli analyze

Analyze a directory for AI project opportunities.

```bash
rapid-cli analyze <path> [flags]

FLAGS:
  --output json|text     Output format (default: text)
  --verbose              Enable verbose output
  --max-tokens N         Maximum tokens for analysis (default: 4000)
```

**Example:**
```bash
rapid-cli analyze ./my-project --output text
```

### rapid-cli generate

Generate a business customization wrapper for a project.

```bash
generate <project-name> <template-type> [flags]

FLAGS:
  --output-dir dir       Output directory for generated files (default: ./generated_wrappers)
  --template-type t      Type of wrapper to generate (default: go-wrapper)
  --enterprise           Generate with enterprise pricing
  --language l           Target programming language (default: go)
```

**Available Template Types:**
- `go-wrapper`: Go-based business wrapper
- `python-wrapper`: Python-based business wrapper
- `node-wrapper`: Node.js-based business wrapper
- `terraform-wrapper`: Terraform infrastructure wrapper
- `dockerfile-wrapper`: Docker-based deployment wrapper

**Example:**
```bash
rapid-cli generate "Docker" "go-wrapper" --output-dir ./business-wrappers
```

### rapid-cli complete

Complete a multi-step workflow (placeholder for future functionality).

## API Endpoints

### Web Server API

All API endpoints return JSON responses with consistent structure.

#### Health Check
```
GET /api/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-26T12:00:00Z",
  "version": "1.0.0",
  "platform": "azimuthro",
  "components": {
    "rapid": true,
    "pod": true,
    "shepherd": true,
    "song": true,
    "atp": true
  }
}
```

#### List Projects
```
GET /api/projects
```

**Response:**
```json
{
  "projects": [
    {
      "name": "Go",
      "lang": "Go",
      "type": "module",
      "size": 10240,
      "complexity": 2.5,
      "has_tests": true,
      "has_ci": true,
      "status": "active"
    }
  ]
}
```

#### Analyze Project
```
POST /api/projects/analyze
```

**Request Body:**
```json
{
  "name": "MyProject",
  "lang": "Go",
  "type": "module",
  "size": 10240,
  "complexity": 2.5,
  "has_tests": true,
  "has_ci": true
}
```

**Response:**
```json
{
  "analysis": {
    "project_name": "MyProject",
    "complexity": 2.5,
    "has_tests": true,
    "has_ci": true,
    "recommendations": [
      {
        "priority": "MEDIUM",
        "title": "Go project detected",
        "description": "Good opportunity for Go wrapper generation",
        "type": "wrapper",
        "estimated_tokens": 1500,
        "estimated_cost": 3.0
      }
    ],
    "generated_code": [
      {
        "file": "business_wrapper.go",
        "language": "Go",
        "size": 2048,
        "description": "Business customization wrapper"
      }
    ]
  },
  "pricing": {
    "estimated_tokens": 4000,
    "estimated_cost": 5.0,
    "cache_efficiency": 0.8,
    "currency": "USD"
  }
}
```

#### List Wrappers
```
GET /api/wrappers
```

**Response:**
```json
{
  "wrappers": [
    {
      "name": "Go Wrapper",
      "type": "go",
      "description": "Standard Go wrapper for Azzurro platform integration",
      "supported_projects": ["Go", "Python", "Docker"],
      "template_file": "go-wrapper.template"
    }
  ]
}
```

## Pricing and Economics

### Cost Structure

RAPID operates on a pay-as-you-go model with competitive pricing:

| Operation | Cost per 1K tokens | Notes |
|-----------|-------------------|-------|
| Analysis  | $0.002            | Base analysis cost |
| Operations | $0.50           | Per operation (ε.g., wrapper generation) |
| Cache Hit | - 90% discount    | Excellent cache efficiency |
| Enterprise | – 30% discount | Premium rates for enterprise customers |

### Example Costs

- **Simple Analysis** (500 tokens): $1.00
- **Complex Analysis** (4,000 tokens): $8.00
- **Wrapper Generation** (2,000 tokens): $4.00
- **Enterprise Analysis** (with 30% discount): $5.60

### Pricing Efficiency

- **Average Cache Hit Rate**: 85%
- **Average Cost Reduction**: 75% with caching
- **Enterprise Discount**: 30% off standard rates

## Integration with Azzurro Modules

### POD (Project Operations Database)

RAPID integrates with POD for:
- Project metadata storage and retrieval
- Persistent analysis results
- Cross-project analytics

### SHEPHERD (Security and Compliance)

RAPID works with SHEPHERD for:
- Wrapper security scanning
- Compliance checks
- Vulnerability assessment

### SONG (Server Orchestration and Governance)

RAPID integrates with SONG for:
- Web server orchestration
- API gateway management
- Service governance

### ATP (Adaptive Task Processing)

RAPID leverages ATP for:
- Workflow automation
- Task distribution
- Resource optimization

## Templates and Code Generation

### Available Templates

RAPID generates comprehensive business customization wrappers with the following components:

1. **Core Business Logic**: Platform-native business rules and integration points
2. **API Endpoints**: RESTful interfaces for external communication
3. **Configuration Management**: Environment-specific settings
4. **Security Integration**: Authentication and authorization mechanisms
5. **Monitoring and Logging**: Health checks and performance metrics
6. **Documentation**: Generated documentation and usage examples

### Generated File Structure

For a Go project wrapper:
```
generated_wrappers/
├── business_wrapper.go                    # Core wrapper implementation
├── business_wrapper.go.metadata.json       # Wrapper metadata
├── healthcheck.sh                         # Health check script
├── entrypoint.sh                          # Docker entrypoint
├── Makefile                               # Build automation
└── README.md                              # Documentation
```

## Development and Testing

### Running Tests

```bash
# Install dependencies (if any)
go mod tidy

# Run tests
mkdir -p ./test_data
cp -r sample_projects ./test_data/
go test ./... -v
```

### Sample Projects

Sample project structures for testing:

```bash
# Create test directory structure
mkdir -p ./test_data/go-project
cmp -r sample/go/ ./test_data/go-project/

mkdir -p ./test_data/python-project
ncmp -r sample/python/ ./test_data/python-project/

mkdir -p ./test_data/js-project
ncmp -r sample/javascript/ ./test_data/js-project/
```

### Development Workflow

1. **Code Analysis**: Use `rapid-cli analyze` to understand existing code
2. **Wrapper Generation**: Use `rapid-cli generate` to create business solutions
3. **Testing**: Validate generated wrappers with integration tests
4. **Documentation**: Update generated documentation

## Performance Considerations

### Resource Requirements

- **Memory**: 2-4 GB RAM (for Ollama model loading)
- **Storage**: 100 MB-1 GB for models and generated artifacts
- **CPU**: Minimal for wrapper generation (I/O bound)
- **Network**: Local Ollama connection only

### Optimization Opportunities

1. **Caching**: Results are cached based on project hash
2. **Incremental Analysis**: Only modified files are re-analyzed
3. **Parallel Processing**: Multiple analysis agents can run simultaneously
4. **Edge Cases**: Efficient handling of large codebases

## Monitoring and Observability

### System Metrics

RAPID collects and reports the following metrics:

- **Analysis Performance**: Time taken for project analysis
- **Wrapper Generation Speed**: Rate of wrapper creation
- **Cost Tracking**: Real-time cost monitoring
- **Cache Efficiency**: Hit/Miss ratios
- **Resource Utilization**: CPU, memory, disk usage

### Health Monitoring

The web interface provides:
- **Real-time status dashboard**
- **API health checking**
- **Performance metrics**
- **Error tracking**
- **Usage analytics**

## Future Enhancements

### Planned Features

1. **Multi-modal Analysis**: Support for project visual analysis
2. **Smart Templates**: AI-powered template generation
3. **Cloud Integration**: Hybrid on-premise/cloud deployment
4. **Advanced Analytics**: Predictive maintenance and optimization
5. **Plugin Architecture**: Extensible plugin system
6. **Real-time Collaboration**: Multi-user project analysis

### Research Areas

- Code similarity detection
- Automated dependency analysis
- Business rule inference
- Real-time collaboration APIs

## License

```mit
RAPID - Azzurro Technology AI Analysis and Wrapper Generation
Copyright (c) 2025 Azzurro Technology

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Support

For support, please:
1. Check our documentation at https://github.com/azzurrotech/rapid
2. Report issues with the [GitHub issue tracker]
3. Contact our development team at dev@azzurro.tech

## Related Projects

RAPID is part of the Azzurro Technology ecosystem:

- **[POD]** - Project Operations Database
- **[SHEPHERD]** - Security and Compliance Framework
- **[SONG]** - Server Orchestration and Governance
- **[ATP]** - Adaptive Task Processing
- **[PIPELINE]** - Development Pipeline Management

All projects are MIT licensed and designed for local execution with Ollama AI models.

## Acknowledgments

Special thanks to the entire Azzurro Technology team for their support and guidance. This project was developed as part of Azzurro's commitment to open-source AI development tools.

---
*Version 1.0.0 • Last Updated: January 26, 2025*
*Developed by Azzurro Technology for local AI model execution*