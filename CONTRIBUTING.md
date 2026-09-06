# Contributing to Eino Middleware

Thank you for your interest in contributing to Eino Middleware! This document provides guidelines for contributing to the project.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and collaborative environment.

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce** the behavior
- **Expected vs actual behavior**
- **Environment details** (Go version, OS, etc.)
- **Code samples** or test cases if applicable

### Suggesting Enhancements

Enhancement suggestions are welcome! Please provide:

- **Clear use case** for the enhancement
- **Expected behavior** and API design
- **Potential implementation** approach (optional)
- **Impact** on existing users

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Follow the code style**:
   - Run `gofmt` and `golangci-lint`
   - Add comments for exported functions
   - Keep functions focused and testable
3. **Add tests**:
   - Unit tests for new functionality
   - Coverage should not decrease
   - Tests should be deterministic and fast
4. **Update documentation**:
   - Update README if adding new features
   - Add examples for new middleware
   - Document breaking changes
5. **Commit message format**:
   ```
   [package] Short description (50 chars max)
   
   Longer explanation if needed. Wrap at 72 characters.
   Explain the problem this commit solves and why this approach.
   
   Fixes #123
   ```
   Examples:
   - `[ratelimit] Add Redis distributed rate limiter`
   - `[metrics] Fix TTFT histogram bucket configuration`
   - `[docs] Update quick start guide`

## Project Structure

```
eino-middleware/
├── ratelimit/          # Rate limiting (independent module)
│   ├── go.mod
│   ├── ratelimit.go
│   └── ratelimit_test.go
├── circuitbreaker/     # Circuit breaker (independent module)
│   ├── go.mod
│   ├── circuitbreaker.go
│   └── circuitbreaker_test.go
├── metrics/            # Prometheus metrics (independent module)
├── trace/              # Distributed tracing (independent module)
├── fallback/           # Fallback strategies (independent module)
├── observability/      # Eino callbacks integration (independent module)
├── logger/             # Structured logging (independent module)
└── examples/           # Usage examples
    └── basic/
```

Each package is an **independent Go module** with its own `go.mod`.

## Development Workflow

### Setup

```bash
git clone https://github.com/yourusername/eino-middleware.git
cd eino-middleware

# Install dependencies for all modules
./scripts/install-deps.sh

# Run all tests
./scripts/test-all.sh
```

### Adding a New Middleware

1. **Create package directory**:
   ```bash
   mkdir newmiddleware
   cd newmiddleware
   go mod init github.com/yourusername/eino-middleware/newmiddleware
   ```

2. **Define interface** (keep it minimal):
   ```go
   // Package newmiddleware provides...
   package newmiddleware
   
   // Middleware interface
   type Middleware interface {
       Process(ctx context.Context, req *Request) (*Response, error)
   }
   ```

3. **Write tests first** (TDD):
   ```go
   func TestNewMiddleware(t *testing.T) {
       // Test cases...
   }
   ```

4. **Implement functionality**

5. **Add example** in `examples/newmiddleware/`

6. **Update main README** with usage

### Running Tests

```bash
# Test a single package
cd ratelimit
go test -v -race -cover

# Test all packages
cd ..
go test ./... -race -cover

# Run linters
golangci-lint run ./...
```

### Code Style

- **Follow [Effective Go](https://golang.org/doc/effective_go.html)**
- **Use `gofmt`** (enforced by CI)
- **Exported functions** must have godoc comments
- **Error messages** should be lowercase and not end with punctuation
- **Context** should be first parameter
- **Interfaces** should be small (1-3 methods)

Example:
```go
// Allow checks if the request is allowed by the rate limiter.
// Returns ErrRateLimitExceeded if limit is reached.
func (l *Limiter) Allow(ctx context.Context) error {
    // Implementation...
}
```

## Testing Guidelines

### Unit Tests

- **Table-driven tests** for multiple cases
- **Use `t.Run()`** for subtests
- **Mock external dependencies** (Redis, HTTP clients, etc.)
- **Test edge cases** (nil inputs, empty strings, zero values)

Example:
```go
func TestTokenBucket_Allow(t *testing.T) {
    tests := []struct {
        name    string
        rate    float64
        burst   int
        calls   int
        wantErr bool
    }{
        {"within limit", 10, 20, 5, false},
        {"exceed limit", 10, 20, 25, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation...
        })
    }
}
```

### Integration Tests

- **Use build tags**: `// +build integration`
- **Require external services**: Redis, Prometheus, etc.
- **Run in CI** only on main branch

## Documentation

### Code Comments

```go
// Package ratelimit provides rate limiting middleware for LLM applications.
//
// It supports both in-memory token bucket and Redis-based distributed
// rate limiting suitable for multi-instance deployments.
//
// Example usage:
//
//     limiter := ratelimit.NewTokenBucket(10.0, 20)
//     if err := limiter.Allow(ctx); err != nil {
//         return http.StatusTooManyRequests
//     }
package ratelimit
```

### README Updates

When adding features, update:
- **Quick Start** section with usage example
- **Project Structure** if adding new package
- **Installation** section if dependencies change

## Releasing

Maintainers will handle releases using semantic versioning:

- **MAJOR**: Incompatible API changes
- **MINOR**: Backward-compatible functionality
- **PATCH**: Backward-compatible bug fixes

Example: `v0.1.0` → `v0.2.0` (new middleware) → `v0.2.1` (bugfix)

## Questions?

- **Open an issue** for design discussions
- **Check existing issues** before creating new ones
- **Be patient and respectful** — this is a community project

Thank you for contributing! 🚀
