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
   - `rapid-cli wrap`: Analyze and generate in one step
   - `rapid-cli validate`: Validate configuration and module integrations
   - `rapid-cli serve`: Start the dashboard and API

2. **Web Interface** (`pkg/web/`)
   - RESTful API for project analysis and wrapper generation
   - Dashboard hosted by song and served by atp at `/rapid/`
   - Live status of every integrated module

3. **AI Engine** (`pkg/agent/`)
   - `model_loader.go`: Ollama model management and loading
   - `project_analyzer.go`: Project analysis and recommendation generation
   - `wrapper_generator.go`: Business customization wrapper generation

4. **Pricing Engine** (`pkg/pricing/`)
   - `calculator.go`: Cost calculation and breakdown
   - `logger.go`: Transaction logging and auditing

### Real platform integration

RAPID does not mock the Azzurro modules. `NewRAPIDApplication` opens a real
`atp.Server`, and uses what it mounts:

| Module | What RAPID uses it for |
|--------|------------------------|
| **ATP** | The hosted platform itself: routing, secrets vault, header policy, function registry, head/PWA injection |
| **POD** | Persistent records in the `projects`, `analyses` and `wrappers` tables |
| **SONG** | The dashboard document, published as the `rapid` silo and served at `/rapid/` |
| **SHEPHERD** | Revocable capability tokens for `POST /api/share`, verified by shepherd itself |

The module versions are pinned by `go.mod` with `replace` directives pointing
at the sibling `stenella/atp` source tree, so RAPID and STENELLA run the same
platform code.

### Technical Specifications

- **Language**: Go
- **Dependencies**: Go standard library plus the first-party platform modules
- **Web Interface**: Baseline HTML/CSS/JS (no frameworks)
- **License**: MIT
- **Execution**: Local only; no cloud dependency

## Installation and Setup

### Prerequisites

- Go 1.22+
- The sibling `stenella/atp` checkout (for the platform modules)
- Ollama service (http://localhost:11434), when model-backed analysis is used

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
go build -o rapid-server ./cmd/rapid-server
RAPID_DATA_DIR=./data RAPID_PORT=8080 ./rapid-server

# Dashboard: http://localhost:8080/rapid/  (served by song)
# API:       http://localhost:8080/api/health
# POD:       http://localhost:8080/pod/      (served by pod)
```

`/` redirects to `/rapid/`, so there is one dashboard document rather than two.
The master secret comes from `RAPID_SECRET` (at least 32 bytes) or is generated
once and kept at `<data-dir>/.rapid-secret`.

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

### rapid-cli wrap

Analyze a directory and generate the wrapper its dominant language calls for, in
one step.

```bash
rapid-cli wrap <path>
```

### rapid-cli validate

Check the configuration and exercise each module with a real operation (a pod
write/read, a shepherd token, a song store lookup, an atp vault/registry check).

```bash
rapid-cli validate
```

### rapid-cli serve

Start the dashboard and API on the real atp platform.

```bash
rapid-cli serve --data ./data --host localhost --port 8080
```

## API Endpoints

All endpoints return JSON. Errors use `{"error": "..."}` with a 4xx/5xx status.

### Health

```
GET /api/health
GET /api/health?check=1
```

```json
{
  "status": "healthy",
  "ok": true,
  "timestamp": "2026-10-29T12:00:00Z",
  "version": "1.0.0",
  "platform": "azzurro",
  "data_dir": "./data",
  "components": { "rapid": true, "atp": true, "pod": true, "shepherd": true, "song": true, "secrets": true, "registry": true },
  "module_report": { "status": "healthy", "version": "…", "components": [ … ] }
}
```

The module checks are `atp.Server.Health()`, so every atp application reports the
same components. `?check=1` (or `?probe=1`) runs `atp.Server.Check()` instead: it
writes, reads and deletes a disposable pod record and vault secret, and issues
and verifies a shepherd token. `status` is `degraded` when any component fails.

### Projects

```
GET /api/projects
POST /api/projects
```

`GET` returns the catalog stored in pod. `POST` stores one project:

```json
{ "name": "Example", "description": "…", "language": "Go", "status": "active" }
```

### Analyze a project

```
POST /api/projects/analyze
```

```json
{ "path": "/absolute/path/to/project" }
```

The response is the complete analysis: file inventory, complexity, test/CI
detection, recommendations, integration mapping, proposed files and pricing.
The run is also recorded in pod's `analyses` table.

### Wrappers

```
GET  /api/wrappers
POST /api/wrappers
```

`POST` generates a wrapper and records it:

```json
{ "path": "/absolute/path/to/project", "template_type": "go-wrapper", "enterprise": false }
```

`template_type` is optional; without it RAPID picks the template from the
languages present (Go, Python, Node.js or generic).

### Analyses

```
GET /api/analyses
```

Recorded analysis runs, newest first.

### Integrations

```
GET /api/integrations
```

```json
{ "ok": true, "failures": [] }
```

### Share

```
POST /api/share
```

```json
{ "kind": "wrapper", "id": "abc123", "ttl": "168h" }
```

Returns a shepherd capability token whose metadata names the target:

```json
{ "token": "…", "kind": "wrapper", "id": "abc123", "jti": "…", "issuedAt": "…", "expires": "…" }
```

### Revoke a share

```
POST /api/share/revoke
```

```json
{ "token": "…" }
```

The token value is the credential, so a caller can cancel a link it no longer
has the original response for. The call resolves the token through
`shepherd/capability` and revokes its `jti` in shepherd's deny list:

```json
{ "revoked": true, "kind": "wrapper", "id": "abc123" }
```

`?token=…` works as an alternative to the JSON body. Revocation is shepherd's
normal deny list: in memory, and effective for every host in the process.

### Export

```
GET /api/export?table=projects&format=csv
GET /api/export?table=analyses&format=json&columns=id,project_name,analyzed_at
```

Downloads one of the `projects`, `analyses` or `wrappers` record sets using
**pod's shared export writer** (`pod.ExportRecords`): the same JSON document
shape and the same BOM-prefixed, properly quoted CSV the platform uses
everywhere. `format` is `json` (default) or `csv`; `columns` is an optional
comma-separated projection. The response carries `Content-Disposition:
attachment; filename=rapid-<table>.<ext>`. An unknown table or format is a
`400`, never a truncated file.

### Metrics

```
GET /api/metrics
```

```json
{
  "uptime_seconds": 3600,
  "counts": { "projects": 12, "analyses": 4, "wrappers": 3 },
  "pricing": { "total_transactions": 4, "total_revenue": 0.0123, "average_cost": 0.0031, … }
}
```

Counts are read from pod, not from a second counter that could drift; pricing
comes from RAPID's own `PriceLogger`.

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

### POD (record store)

RAPID uses `atp.Server.Pod()` for:
- Project metadata (`projects` table)
- Every analysis run (`analyses` table, including the full result as JSON)
- Every generated wrapper (`wrappers` table)
- Every export (`GET /api/export`), through `pod.ExportRecords` rather than a
  second CSV/JSON writer

The tables are created and widened with `EnsureStructSchema`, so adding a field
to a record needs no migration.

### SHEPHERD (capabilities and policy)

RAPID uses `atp.Server.Shepherd()` for:
- Revocable share tokens (`POST /api/share` and `POST /api/share/revoke`, backed by `shepherd/capability`)
- The platform's firewall, rate limiting and token verification
- Token ownership tracking, available to any host that reads the same claims

### SONG (hosting)

RAPID uses `atp.Server.Song()` for:
- Publishing the dashboard into the `rapid` silo, served at `/rapid/`
- The shared `/_song/viewkit.js` library the dashboard loads for relative time
- The same static/template/component pipeline every other hosted app uses
- Offline-friendly assets with no external request

### ATP (platform)

RAPID runs on atp for:
- Request routing and hosted-application discovery
- The encrypted secrets vault and header policy
- The serverless function registry
- Generated PWA head fragments and capability surfaces

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

Every generation writes the wrapper and its metadata, named from the project,
template and date:

```
generated_wrappers/
├── my-project_go-wrapper_20261029.go
└── my-project_go-wrapper_20261029.go.metadata.json
```

The actual files a project requires are listed in the analysis result
(`generated_code`); the Go, Python and Node.js templates each produce a
self-contained service that reports the Azzurro module integration.

## Development and Testing

### Running Tests

```bash
go test ./...
go vet ./...
```

The tests build a real platform on a temporary data directory: they exercise
pod persistence, a zipped-up shepherd token, song hosting and the HTTP surface.
No mock module is used or needed.

### Integration tests

`app_test.go` checks that `NewRAPIDApplication` mounts the real pod, song and
shepherd services and that a token issued by RAPID verifies with shepherd's own
manager. `pkg/web/server_test.go` drives the API and the song-hosted dashboard
through `httptest`, and `pkg/agent/` covers the scanner and every template.

### Development Workflow

1. **Code Analysis**: Use `rapid-cli analyze` to understand existing code
2. **Wrapper Generation**: Use `rapid-cli generate` or `rapid-cli wrap`
3. **Validation**: Use `rapid-cli validate` before shipping
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

RAPID collects and reports the following metrics (see `GET /api/metrics`):

- **Analysis Performance**: Time taken for project analysis
- **Wrapper Generation Speed**: Rate of wrapper creation
- **Cost Tracking**: Real-time cost monitoring (`pricing.total_revenue`)
- **Cache Efficiency**: Hit/Miss ratios (`pricing.cache_hit_rate`)
- **Resource Utilization**: uptime, and the pod record counts per table

### Health Monitoring

The web interface provides:
- **Real-time status dashboard** (component KPIs from `atp.Server.Health()`)
- **API health checking** (`/api/health`, with `?check=1` for a live probe)
- **Performance metrics** (`/api/metrics`)
- **Error tracking** (the JSON error envelope on every endpoint)
- **Usage analytics** (record counts read from pod)
- **Export controls** and **share revocation** in the dashboard

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

## ISSUES
- Fix the module structure to allow internal package imports
- Update go.mod to properly reference internal packages
- Try building RAPID CLI with corrected module structure
- Test RAPID CLI analyze command functionality
- Test RAPID CLI generate and wrap commands