# RAG (Retrieval-Augmented Generation) System

A Go-based RAG system that integrates with Ollama for enhanced text generation with retrieval capabilities.

## Features

- **Retrieval-Augmented Generation**: Combines information retrieval with language model generation
- **Ollama Integration**: Seamless integration with Ollama for local LLM inference
- **Modular Architecture**: Clean separation of concerns with well-defined packages
- **Development Tools**: Comprehensive development environment with testing and linting

## Prerequisites

- Go 1.25 or later
- Ollama (for LLM inference)
- Make (for build automation)

## Quick Start

1) Clone and setup

```bash
git clone <repository-url>
cd rag
make dev-setup
```

2) Install Ollama

Follow the [Ollama installation guide](https://ollama.ai) for your platform.

3) Pull a model (example)

```bash
ollama pull deepseek-r1:1.5b
```

4) Initialize the database (apply migrations and seed)

There is a script to apply migrations and seed initial AI schema/template data. Run:

```bash
go run ./scripts/db_init
```

5) Run the server (development)

```bash
make run
# or
go run ./cmd/server
```

6) Run the development Ollama client (optional)

```bash
make dev-client
```

Default server URL: http://localhost:8080

## Project Structure

```
rag/
├── cmd/                    # Main applications
│   ├── server/            # Main RAG server
│   └── dev/               # Development utilities
│       └── ollama-client/ # Ollama client example
├── pkg/                   # Public packages
├── internal/              # Private packages
├── api/                   # API definitions and handlers
├── configs/               # Configuration files
├── scripts/               # Build and deployment scripts
├── test/                  # Integration tests
├── .docs/                 # Project documentation
└── .vscode/              # VS Code configuration
```

## Development
- [qri-io/jsonschema](https://github.com/qri-io/jsonschema) - JSON Schema compilation/validation used for schema validation on write

## AI API examples (quick)

Below are minimal curl examples to exercise the AI schema/template endpoints exposed under `/v1/ai` (the `/v1` prefix is protected by JWT middleware in normal runs).

Create or update a schema

```bash
curl -X POST http://localhost:8080/v1/ai/schemas \
	-H "Authorization: Bearer $JWT" \
	-H "Content-Type: application/json" \
	-d '{"version":"v1","description":"Greeting schema","schema_json":{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","properties":{"name":{"type":"string"}},"required":["name"]}}'
```

Get a schema by version

```bash
curl -X GET "http://localhost:8080/v1/ai/schemas/get?version=v1" -H "Authorization: Bearer $JWT"
```

Reload compiled schemas (invalidate cache)

```bash
curl -X POST http://localhost:8080/v1/ai/schemas/reload -H "Authorization: Bearer $JWT"
```

Create or update a template

```bash
curl -X POST http://localhost:8080/v1/ai/templates \
	-H "Authorization: Bearer $JWT" \
	-H "Content-Type: application/json" \
	-d '{"name":"greeting","version":"v1","template_text":"Hello {{name}}","schema_version":"v1"}'
```

Get a template

```bash
curl -X GET "http://localhost:8080/v1/ai/templates/get?name=greeting&version=v1" -H "Authorization: Bearer $JWT"
```

Delete a template

```bash
curl -X DELETE "http://localhost:8080/v1/ai/templates/delete?name=greeting&version=v1" -H "Authorization: Bearer $JWT"
```

Bruno collection: the `bruno/` folder in the repo contains pre-built requests (signup/signin, schema/template examples). Use `local.bru` to set `base_url` and manage `jwt_token` when running with Bruno.

### Available Make Targets

- `make help` - Show all available targets
- `make build` - Build the main application
- `make run` - Run the main application
- `make dev-client` - Run the development Ollama client
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage report
- `make fmt` - Format code
- `make lint` - Run linter
- `make clean` - Clean build artifacts

Additional CI-related targets added for convenience:

- `make ci-smoke` - Run focused CI smoke tests locally (migrations + ollama)
- `make ci-lint` - Run linter locally (mirror CI)
- `make ci-full` - Run full test suite with coverage and write coverage.out

### Development Workflow

1. **Setup**: `make dev-setup`
2. **Development**: `make quick` (format, test, build)
3. **Testing**: `make test-coverage`
4. **Cleanup**: `make clean`

## Configuration

Configuration files should be placed in the `configs/` directory. Examples:

- `config.example.yaml` - Example configuration
- `config.example.json` - Alternative JSON format

### Example YAML snippet (configs/dev.yaml)

```yaml
addr: ":8080"
jwt_secret: "your_jwt_secret_key"
database_path: "rag.db"
timeout: "15s"
token_duration: "1h"
migrate_on_start: true

engine:
	model: "deepseek-r1:1.5b"
	timeout: "20s"
	min_confidence: 0.5
	template_version: "v1"

ollama:
	base_url: "http://localhost:11434"
	models:
		- "deepseek-r1:1.5b"
	timeout: "30s"
	retries: 3
	backoff: "500ms"
	circuit_failure_threshold: 5
	circuit_reset: "30s"
```

## Dependencies

The project uses Go modules for dependency management. Key dependencies include:

- [Ollama Go API](https://github.com/ollama/ollama) - For LLM integration
- Additional dependencies will be added as the project evolves

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Format code (`make fmt`)
6. Run linter (`make lint`)
7. Commit your changes (`git commit -m 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

## Development Environment

### VS Code Setup

The project includes VS Code configuration in `.vscode/`:

- `launch.json` - Debug configurations
- Recommended extensions for Go development

### Code Quality

- **Formatting**: Use `gofmt` (via `make fmt`)
- **Linting**: Use `golangci-lint` (via `make lint`)
- **Testing**: Write tests for all new functionality
- **Coverage**: Maintain good test coverage

## Architecture

The system follows Go best practices and standard project layout:

- **cmd/**: Entry points for different applications
- **pkg/**: Libraries that can be imported by external applications
- **internal/**: Private application and library code
- **api/**: API definitions, protocol buffers, OpenAPI specs

## License

This project is licensed under the MIT License. See the `LICENSE` file in the repository root for the full text.

## Support

For questions and support, please [open an issue](link-to-issues) or contact the development team.
