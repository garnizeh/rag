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

### 1. Clone and Setup

```bash
git clone <repository-url>
cd rag
make dev-setup
```

### 2. Install Ollama

Follow the [Ollama installation guide](https://ollama.ai) for your platform.

### 3. Pull a Model

```bash
ollama pull deepseek-r1:1.5b
```

### 4. Run Development Client

```bash
make dev-client
```

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

### Development Workflow

1. **Setup**: `make dev-setup`
2. **Development**: `make quick` (format, test, build)
3. **Testing**: `make test-coverage`
4. **Cleanup**: `make clean`

## Configuration

Configuration files should be placed in the `configs/` directory. Examples:

- `config.example.yaml` - Example configuration
- `config.example.json` - Alternative JSON format

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

[Add your license here]

## Support

For questions and support, please [open an issue](link-to-issues) or contact the development team.
