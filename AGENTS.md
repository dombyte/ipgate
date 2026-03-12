# IPGate - Agent Documentation

## Project Overview

High-performance IP blocking system in Go with caching.

## Branch Management

- **Never commit or merge directly into `master` or `main` branches.**
- For new features, fixes, or changes, create a new branch following the schema:
  - `{feat,fix,doc,refactor,chore}/{name}`
  - Example: `feat/add-dark-mode`, `fix/cache-leak`, `doc/update-readme`

## Tools
go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -ignore "(?:.*_test\.go|.*test.*\.go)" ./..
go run github.com/securego/gosec/v2/cmd/gosec@v2.23.0 ./...
go run github.com/gordonklaus/ineffassign@latest ./...
go run golang.org/x/tools/cmd/deadcode@latest ./...
go run github.com/client9/misspell/cmd/misspell@latest -w . -j 200
go fmt ./...
go vet ./...
## Project Structure

```
IPGate/
├── config/
│   ├── blocklists/
│   │   └── blocklist.txt
│   ├── templates/
│   │   └── error.html
│   ├── whitelists/
│   │   └── whitelist.txt
│   ├── config.yaml
│   └── test_config.yaml
├── cmd/
│   └── ipgate/
│       ├── main.go
│       ├── main_test.go
│       └── router_integration_test.go
├── internal/
│   ├── blocklist/
│   │   ├── blocklist.go
│   │   ├── blocklist_test.go
│   │   └── watcher.go
│   ├── cache/
│   │   ├── cache.go
│   │   └── cache_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── handler/
│   │   ├── handlers.go
│   │   ├── debug_handlers.go
│   │   └── handler_test.go
│   ├── middleware/
│   │   ├── middleware.go
│   │   └── logging.go
│   ├── models/
│   │   ├── models.go
│   │   └── models_test.go
│   ├── routes/
│   │   ├── routes.go
│   │   └── routes_test.go
│   └── utils/
│       ├── utils.go
│       └── utils_test.go
├── docker-compose.debug.yml
├── docker-compose.yml
├── Dockerfile
├── Dockerfile.test
├── go.mod
├── go.sum
├── ipgate
├── README.md
```

## Package Structure Guidelines

### Core Packages

- **`internal/blocklist/`**: IP blocklist management (loading, validation, watching)
- **`internal/cache/`**: Caching system for blocked/allowed IPs
- **`internal/config/`**: Configuration loading and management
- **`internal/handler/`**: HTTP request handlers
- **`internal/middleware/`**: HTTP middleware functions
- **`internal/models/`**: Data structures and models
- **`internal/routes/`**: Route definitions and composition
- **`internal/utils/`**: Utility functions and helpers

### Package Responsibilities

#### Middleware Package (`internal/middleware/`)

**Purpose**: HTTP middleware functions that process requests/responses

**Rules**:

- All middleware functions must return `func(http.Handler) http.Handler`
- Middleware should be stateless or use dependency injection
- Each middleware type should be in its own file:
  - `logging.go` - Request logging
  - `auth.go` - Authentication (if re-added)
  - `rate_limit.go` - Rate limiting (if re-added)
  - `security.go` - Security headers (if re-added)

**Example**:

```go
// internal/middleware/logging.go
package middleware

import (
    "net/http"
    "time"
    "github.com/dombyte/ipgate/internal/models"
)

func RequestLoggingMiddleware(deps *models.HandlerDeps) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            // Logging logic
            next.ServeHTTP(w, r)
            // Duration logging
        })
    }
}
```

#### Handler Package (`internal/handler/`)

**Purpose**: HTTP request handlers that generate responses

**Rules**:

- Handlers should focus on request processing and response generation
- Business logic should be delegated to other packages
- Split handlers by function:
  - `handlers.go` - Main public API handlers
  - `debug_handlers.go` - Debug/admin endpoints
  - `auth_handlers.go` - Authenticated endpoints (if re-added)

**Example**:

```go
// internal/handler/handlers.go
package handler

import (
    "net/http"
    "github.com/dombyte/ipgate/internal/models"
)

func ErrorHandler(deps *models.HandlerDeps) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Handle IP blocking logic
    })
}

func HealthHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Health check logic
    }
}
```

#### Models Package (`internal/models/`)

**Purpose**: Data structures and request/response models

**Rules**:

- All data structures should be defined here
- No business logic in models
- Split by type if needed:
  - `response_models.go` - Response structures
  - `request_models.go` - Request payloads
  - `error_models.go` - Error responses

**Example**:

```go
// internal/models/models.go
package models

type ErrorPageData struct {
    RequestID   string
    ClientIP    string
    StatusCode  int
    BlockReason string
}

type HandlerDeps struct {
    Config *config.Config
    Cache  *cache.Cache
}
```

#### Utility Package (`internal/utils/`)

**Purpose**: Reusable utility functions

**Rules**:

- Generic functions that don't belong in domain packages
- HTTP utilities, validation, file operations
- Split by domain:
  - `http_utils.go` - HTTP helpers
  - `ip_utils.go` - IP address utilities
  - `validation_utils.go` - Validation functions

**Example**:

```go
// internal/utils/http_utils.go
package utils

import (
    "encoding/json"
    "net/http"
)

func Encode(w http.ResponseWriter, status int, v any) error {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    return json.NewEncoder(w).Encode(v)
}
```

### Structure Best Practices

1. **Single Responsibility Principle**: Each package should have one clear purpose
2. **Dependency Direction**: Dependencies should flow downward (routes → handlers → services → models)
3. **No Circular Dependencies**: Avoid circular imports between packages
4. **Clear API Boundaries**: Each package should expose a clean, minimal API
5. **Avoid Global State**: Use dependency injection instead of global variables
6. **Testable Components**: Design packages to be easily testable in isolation

### What NOT to Do

- **Don't mix middleware and handlers** in the same package
- **Don't put business logic** in handler packages
- **Don't create utility functions** in domain packages
- **Don't duplicate data structures** across packages


### Migration Guidelines

When refactoring existing code:

1. **Create new packages** before moving code
2. **Update imports** in dependent packages
3. **Test thoroughly** after each move
4. **Keep old files** until new structure is verified
5. **Remove old files** only after confirmation


## Key Components

- **Config**: YAML-based configuration
- **Blocklist**: Local/remote IP blocklist loading and validation
- **Cache**: In-memory cache for DENY (blocked) and ALLOW (not blocked or whitelisted) IPs with TTL and pruning
- **Handler**: HTTP endpoints for blocking and debugging
- **Watcher**: File watcher for automatic file reloading
- **Models**: Request/Response types and data structures (HandlerDeps, ErrorPageData)
- **Routes**: Centralized route management with middleware composition
- **Utils**: Generic utility functions for HTTP encoding/decoding

## Build and Test

```bash
# Build
go build -o ipgate ./cmd/ipgate

# Test
go clean -testcache && go test ./... -cover -race

```

## Code Style

- `go fmt ./...` before commits
- Go conventions (camelCase, short functions)
- Comments for public functions
- Focused, testable functions
- Secure
- Performant

## Testing

- Unit tests for all new code (`_test.go`)
- > 90% coverage target
- Test edge cases (invalid IPs, empty lists, cache operations, whitelist changes, blacklist changes)
- Check for race conditions with Go’s race detector
- Test with fuzzing to uncover edge-case exploits
- Test for proper security implementation like describled in the Security Section
- Use Vet to examine suspicious constructs
- Docker for integration tests

## Security

- Validate all inputs (IPs, file paths)
- Proper file permissions
- Updated dependencies
- Harden against common go exploits and vulnerability

## Go Code Style and Best Practices

### Code Organization

- **Standard Go Project Structure**: Follow the standard Go project layout with `cmd/`, `internal/`, and `pkg/` directories
- **Separation of Concerns**: Keep related functionality in separate packages (e.g., `handler`, `models`, `routes`, `utils`)
- **Dependency Injection**: Use dependency injection pattern with a central `HandlerDeps` struct
- **Constructor Pattern**: Implement `NewServer()` or similar constructor functions for initializing components

### Code Style

- **Formatting**: Always run `go fmt ./...` before committing
- **Naming**: Use camelCase for variables and functions, PascalCase for types and exported functions
- **Function Length**: Keep functions short and focused (ideally < 20 lines)
- **Error Handling**: Return errors explicitly, don't panic
- **Comments**: Document public functions and types with Godoc comments
- **Imports**: Group imports by standard library, third-party, and project imports

### HTTP Handlers

- **Return `http.Handler`**: Handler functions should return `http.Handler` instead of `http.HandlerFunc` for better middleware compatibility
- **Dependency Injection**: Use `HandlerDeps` struct to pass all dependencies to handlers
- **Middleware Pattern**: Implement middleware as functions that return `func(http.Handler) http.Handler`
- **Route Management**: Centralize route definitions in `routes.go`

### Testing

- **Test Files**: Create `_test.go` files for each package
- **Test Coverage**: Aim for > 90% coverage
- **Test Isolation**: Avoid global state in tests; create fresh instances for each test
- **Test Patterns**:
  - Use `httptest.NewRecorder()` and `http.NewRequest()` for HTTP tests
  - Test both happy paths and error cases
  - Test edge cases (empty inputs, invalid data, etc.)
  - Use `t.Run()` for sub-tests


### Common Patterns

- **Dependency Injection**: Pass dependencies as parameters rather than using global variables
- **Middleware Composition**: Chain middleware functions for flexible request processing
- **Error Handling**: Use consistent error response formats
- **Configuration**: Use YAML for configuration with proper struct tags
- **Logging**: Use structured logging with context information


## Known Issues and Solutions

This section documents common problems encountered by agents and their solutions. Always check here before spending time debugging issues that have been solved before.

### Test Execution Order Issues


### Koanf Configuration Unmarshaling

**Problem**: Koanf fails to unmarshal complex nested structures like slices of objects

**Symptoms**:

- Boolean fields not being populated correctly (`debug_endpoint`, `watch_files_enabled`)
- File configurations returning empty arrays (`blacklist_files`, `whitelist_files`)
- Koanf store contains correct data but struct fields remain empty
- Debug output shows: `DEBUG: After unmarshal - BlacklistFiles: 0, WhitelistFiles: 0`

**Root Cause**:

- Koanf uses `koanf` as the default struct tag for unmarshaling
- Configuration struct uses `yaml` tags instead of `koanf` tags
- Mismatch between struct tags and Koanf's default tag name
- Complex nested structures (slices of objects) require proper tag matching

**Solution**:

Use `UnmarshalWithConf` with explicit `Tag: "yaml"` parameter:

```go
// Before (not working):
if err := c.k.Unmarshal("", cfg); err != nil {
    return nil, err
}

// After (working):
if err := c.k.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
    return nil, err
}
```

**Files Modified**:

- `internal/config/api.go` - Changed unmarshaling to use yaml tags

**Verification**:

- Check that file configurations are properly populated: `BlacklistFiles: 1, WhitelistFiles: 1`
- Verify boolean fields are set correctly: `debug_endpoint: true`
- Confirm all configuration values are loaded from YAML file

**Related Issues**:

- Boolean field unmarshaling failures
- Empty slice unmarshaling for complex nested structures
- Struct tag mismatch between Koanf defaults and application conventions
