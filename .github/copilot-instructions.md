# Copilot Instructions for RAG Project

## Project Overview

This is a **Retrieval-Augmented Generation (RAG) system** built in Go that integrates with Ollama for local LLM inference. The system combines information retrieval with language model generation to provide enhanced, context-aware responses.

## Technology Stack

- **Language**: Go 1.25+
- **AI/ML**: Ollama (local LLM inference)
- **Architecture**: Modular, following Go best practices
- **Build Tool**: Make
- **Version Control**: Git

## Project Structure and Conventions

### Directory Layout
```
rag/
├── cmd/                    # Applications (main packages)
│   ├── server/            # Main RAG server application
│   └── dev/               # Development utilities
├── pkg/                   # Public libraries (exportable)
├── internal/              # Private application code
├── api/                   # API definitions, handlers, middleware
├── configs/               # Configuration files and examples
├── scripts/               # Build, deployment, and utility scripts
├── test/                  # Integration and end-to-end tests
├── .docs/                 # Project documentation and tasks
└── .vscode/              # VS Code specific configuration
```

### Coding Standards

1. **Go Conventions**:
   - Follow standard Go formatting (`gofmt`)
   - Use meaningful package and variable names
   - Write comprehensive tests for all functionality
   - Document public APIs with comments

2. **Import Organization**:
   ```go
   import (
       // Standard library
       "context"
       "fmt"
       
       // Third-party packages
       "github.com/ollama/ollama/api"
       
       // Internal packages
       "github.com/garnizeh/rag/internal/retrieval"
       "github.com/garnizeh/rag/pkg/embeddings"
   )
   ```

3. **Error Handling**:
   - Always handle errors explicitly
   - Use wrapped errors for context: `fmt.Errorf("operation failed: %w", err)`
   - Log errors appropriately

## Development Workflow

### Setup Commands
```bash
# Initial setup
make dev-setup

# Development cycle
make quick  # format + test + build

# Testing
make test-coverage

# Linting
make lint
```

### Common Patterns

1. **Context Usage**: Always pass `context.Context` as the first parameter for operations that might be cancelled or have timeouts.

2. **Configuration**: Use struct-based configuration with sensible defaults:
   ```go
   type Config struct {
       OllamaURL    string `yaml:"ollama_url" default:"http://localhost:11434"`
       ModelName    string `yaml:"model_name" default:"deepseek-r1:1.5b"`
       Timeout      time.Duration `yaml:"timeout" default:"30s"`
   }
   ```

3. **Interfaces**: Define interfaces for testability:
   ```go
   type LLMClient interface {
       Generate(ctx context.Context, prompt string) (string, error)
   }
   ```

## Key Components

### 1. Ollama Integration
- Located in `pkg/ollama/` or `internal/llm/`
- Handle connection management and model interaction
- Implement retry logic and error handling

### 2. Retrieval System
- Document indexing and search capabilities
- Vector embeddings for semantic search
- Integration with various data sources

### 3. API Layer
- RESTful API for RAG operations
- Proper HTTP status codes and error responses
- Request/response validation

### 4. Configuration Management
- YAML/JSON configuration files
- Environment variable overrides
- Validation and defaults

## Testing Guidelines

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test component interactions
3. **End-to-End Tests**: Test complete workflows

### Test Structure
```go
func TestFunctionName(t *testing.T) {
    // Arrange
    setup()
    
    // Act
    result, err := functionUnderTest()
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

## Build and Deployment

### Make Targets
- `make build` - Build the main application
- `make test` - Run all tests
- `make lint` - Run code linting
- `make clean` - Clean build artifacts
- `make dev-client` - Run development Ollama client

### Environment Configuration
- Development: Use `config.dev.yaml`
- Production: Use `config.prod.yaml`
- Local: Use `config.local.yaml` (gitignored)

## AI/Copilot Specific Guidelines

When helping with this project:

1. **Understand the Context**: This is a RAG system, so suggestions should consider:
   - Information retrieval patterns
   - LLM integration best practices
   - Performance and scalability

2. **Go Idioms**: Prefer Go-idiomatic solutions:
   - Use channels for concurrency
   - Prefer composition over inheritance
   - Keep interfaces small and focused

3. **Error Handling**: Always include proper error handling in suggestions

4. **Performance**: Consider performance implications, especially for:
   - Vector operations
   - LLM API calls
   - Large document processing

5. **Security**: Be mindful of:
   - Input validation
   - API rate limiting
   - Secure configuration handling

## Dependencies Management

- Use `go mod` for dependency management
- Pin major versions for stability
- Regular updates via `make deps-update`
- Vendor dependencies if needed for reproducible builds

## Documentation

- Keep README.md updated with setup instructions
- Document public APIs with Go doc comments
- Maintain .docs/ folder with architecture decisions and tasks
- Include examples in documentation

## Debugging and Logging

- Use structured logging (consider [`slog`](https://pkg.go.dev/golang.org/x/exp/slog))
- Include request IDs for tracing
- Log important operations and errors
- Use appropriate log levels (debug, info, warn, error)

## Performance Considerations

- Profile code for bottlenecks
- Use connection pooling for external services
- Implement caching where appropriate
- Monitor memory usage for large document processing

---

## Windows & Make Notes

If you are developing on Windows, you may need to install [Make](https://chocolatey.org/packages/make) and [GNU Core Utilities](https://gnuwin32.sourceforge.net/packages/coreutils.htm) for full compatibility with the provided Makefile. Alternatively, use PowerShell commands for cleaning and other tasks.

---

## Configuration Files

Example configuration files are located in the `configs/` directory:
- `config.example.yaml`
- `config.example.json`

Copy and rename these files for your environment (e.g., `config.dev.yaml`, `config.local.yaml`).

---

## License & Contributing

See the main `README.md` for licensing and contribution guidelines. To contribute:
1. Fork the repository
2. Create a feature branch
3. Submit a pull request

---

## Useful Links

- [Ollama](https://ollama.ai)
- [Go Documentation](https://golang.org/doc/)
- [Golangci-lint](https://golangci-lint.run/)
