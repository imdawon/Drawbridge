# Drawbridge Development Guide

## Commands
- Build all platforms: `./build/all.sh`
- Build specific platform: `./build/[platform]_[arch].sh [version]`
- Run tests: `go test ./...`
- Run specific test: `go test ./path/to/package -run TestName`
- Format code: `go fmt ./...`
- Build templ files: `templ generate ./cmd/dashboard/ui/templates/...`

## Code Style Guidelines
- **Imports**: Standard library first, project imports second, third-party last
- **Formatting**: Use tabs for indentation, standard Go formatting
- **Types**: Use strong typing, avoid interface{} when possible
- **Naming**:
  - CamelCase for exported functions/types
  - camelCase for unexported functions/variables
  - ALL_CAPS for constants
- **Error Handling**:
  - Check errors immediately
  - Use fmt.Errorf with %w for error wrapping
  - Use slog for error logging
- **Comments**: Document exported functions and complex logic
- **Tests**: Write tests for critical functionality
- **Packages**: Keep packages small with focused responsibilities