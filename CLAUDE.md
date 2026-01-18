# CLAUDE.md - AI Assistant Guide for VTK Project

## Project Overview

**Project Name:** vtk
**Language:** Go (Golang)
**License:** GNU General Public License v3.0
**Repository Status:** Early initialization stage
**Current Branch:** `claude/add-claude-documentation-SsBqX`

This is a Go-based project currently in its initial setup phase. The repository contains foundational files (`.gitignore`, `LICENSE`) and is ready for active development.

---

## Repository Structure

### Current State
```
vtk/
├── .git/                 # Git repository metadata
├── .gitignore           # Go-specific ignore patterns
├── LICENSE              # GPLv3 full license text
└── CLAUDE.md           # This file - AI assistant guide
```

### Expected Go Project Structure

When development begins, the project should follow standard Go conventions:

```
vtk/
├── cmd/                 # Main applications for this project
│   └── vtk/            # Main application entry point
│       └── main.go
├── internal/           # Private application and library code
│   ├── pkg/           # Internal packages
│   └── app/           # Application logic
├── pkg/                # Public library code (if any)
├── api/                # API definitions (OpenAPI/Swagger, Protocol Buffers)
├── configs/            # Configuration file templates or defaults
├── scripts/            # Build, install, analysis scripts
├── test/               # Additional external test apps and test data
├── docs/               # Design and user documents
├── examples/           # Example applications
├── go.mod              # Go module definition
├── go.sum              # Go dependency checksums
├── Makefile            # Build automation (optional)
├── README.md           # Project documentation
└── CLAUDE.md           # This file
```

**References:**
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Effective Go](https://go.dev/doc/effective_go)

---

## Development Workflows

### Git Workflow

**Branch Naming Convention:**
- Feature branches: `claude/<feature-description>-<session-id>`
- Example: `claude/add-claude-documentation-SsBqX`

**Branch Strategy:**
- All development happens on feature branches
- Branch names must start with `claude/` and end with matching session ID
- Push with: `git push -u origin <branch-name>`
- **Critical:** Pushing to incorrectly named branches will fail with 403 error

**Commit Guidelines:**
- Use clear, descriptive commit messages
- Follow conventional commits format when possible:
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation
  - `refactor:` for code refactoring
  - `test:` for adding tests
  - `chore:` for maintenance tasks

### Go Development Workflow

1. **Initialize Go Module** (if not already done):
   ```bash
   go mod init github.com/vishnuvyas/vtk
   ```

2. **Add Dependencies:**
   ```bash
   go get <package-name>
   go mod tidy
   ```

3. **Build:**
   ```bash
   go build ./cmd/vtk
   ```

4. **Run:**
   ```bash
   go run ./cmd/vtk
   ```

5. **Test:**
   ```bash
   go test ./...                    # Run all tests
   go test -v ./...                 # Verbose output
   go test -cover ./...             # With coverage
   go test -race ./...              # Race detection
   ```

6. **Format and Lint:**
   ```bash
   go fmt ./...                     # Format code
   go vet ./...                     # Static analysis
   golangci-lint run                # Comprehensive linting (if installed)
   ```

### Network Retry Policy

For git operations (`push`, `fetch`, `pull`):
- Retry up to 4 times on network failures
- Use exponential backoff: 2s, 4s, 8s, 16s
- Example: `git push` → wait 2s if failed → retry → wait 4s if failed → retry, etc.

---

## Code Conventions

### Go Style Guidelines

1. **Naming Conventions:**
   - Use `camelCase` for local variables
   - Use `PascalCase` for exported functions, types, and constants
   - Use `snake_case` for file names (e.g., `user_service.go`)
   - Interface names should be descriptive (e.g., `Reader`, `Writer`, `UserRepository`)

2. **Package Organization:**
   - One package per directory
   - Package names should be lowercase, single-word
   - Avoid `util`, `common`, or `helpers` packages
   - Use meaningful, specific package names

3. **Error Handling:**
   - Always check errors explicitly
   - Wrap errors with context using `fmt.Errorf` with `%w` verb
   - Use custom error types when appropriate
   - Don't panic in library code

4. **Code Structure:**
   - Keep functions small and focused
   - Prefer composition over inheritance
   - Use interfaces for abstraction
   - Document exported functions, types, and packages

5. **Commenting:**
   - Package comments should describe the package purpose
   - Doc comments for exported identifiers should start with the identifier name
   - Use `//` for single-line comments
   - Use `/* */` for block comments sparingly

### Example Code Structure

```go
// Package vtk provides functionality for [description].
package vtk

import (
    "context"
    "fmt"
)

// Config holds the configuration for the VTK application.
type Config struct {
    Host string
    Port int
}

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
    return &Config{
        Host: "localhost",
        Port: 8080,
    }
}

// Start initializes and starts the application.
func (c *Config) Start(ctx context.Context) error {
    if c.Port == 0 {
        return fmt.Errorf("invalid port: %d", c.Port)
    }
    // Implementation
    return nil
}
```

---

## AI Assistant Guidelines

### When Working with This Codebase

1. **Before Making Changes:**
   - Always read files before modifying them
   - Understand existing code patterns and follow them
   - Check for existing implementations before creating new ones

2. **Code Quality:**
   - Run `go fmt` on all modified Go files
   - Ensure code passes `go vet`
   - Add tests for new functionality
   - Maintain test coverage for modified code

3. **Dependencies:**
   - Prefer standard library when possible
   - Document why external dependencies are needed
   - Run `go mod tidy` after adding/removing dependencies
   - Check license compatibility (project is GPL v3)

4. **Security Considerations:**
   - Avoid command injection vulnerabilities
   - Validate all user input
   - Don't commit secrets or credentials
   - Be cautious with file operations and path traversal
   - Use context for cancellation and timeouts

5. **Documentation:**
   - Update README.md when adding features
   - Document public APIs with godoc comments
   - Include examples for complex functionality
   - Keep CLAUDE.md updated with structural changes

6. **Testing:**
   - Write table-driven tests where appropriate
   - Test edge cases and error conditions
   - Use subtests for better organization (`t.Run`)
   - Mock external dependencies

7. **Commit Practices:**
   - Only commit when explicitly requested by user
   - Stage relevant files only (avoid committing generated files)
   - Write meaningful commit messages
   - Don't amend commits that have been pushed

### Files to Ignore

As per `.gitignore`, never commit:
- Compiled binaries (`*.exe`, `*.dll`, `*.so`, `*.dylib`)
- Test binaries (`*.test`)
- Output files (`*.out`)
- Coverage profiles (`*.coverprofile`, `profile.cov`)
- Go workspace files (`go.work`, `go.work.sum`)
- Environment files (`.env`)

### Recommended Tools

- **Testing:** Standard `testing` package, `testify` for assertions
- **Mocking:** `gomock`, `mockery`
- **Linting:** `golangci-lint`
- **Code Coverage:** `go test -cover`, `gocov`
- **Documentation:** `godoc`, `pkgsite`

---

## Common Tasks Reference

### Initialize New Go Module
```bash
go mod init github.com/vishnuvyas/vtk
go mod tidy
```

### Add New Package
```bash
mkdir -p internal/pkg/newpackage
# Create newpackage.go with package declaration
go mod tidy
```

### Run Tests with Coverage
```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Build for Multiple Platforms
```bash
GOOS=linux GOARCH=amd64 go build -o vtk-linux-amd64 ./cmd/vtk
GOOS=darwin GOARCH=amd64 go build -o vtk-darwin-amd64 ./cmd/vtk
GOOS=windows GOARCH=amd64 go build -o vtk-windows-amd64.exe ./cmd/vtk
```

### Vendor Dependencies (if needed)
```bash
go mod vendor
```

---

## License Compliance

This project is licensed under **GNU General Public License v3.0**. When working with this codebase:

1. **All contributions must be compatible with GPL v3**
2. **External dependencies must have GPL-compatible licenses:**
   - Compatible: MIT, BSD, Apache 2.0
   - Incompatible: Proprietary licenses
3. **Derived works must also be GPL v3**
4. **Include license headers in source files if required by team conventions**

Example license header (if needed):
```go
// Copyright (C) YYYY Author Name
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
```

---

## Quick Start for AI Assistants

When first working with this repository:

1. **Check current state:**
   ```bash
   git status
   git log --oneline -5
   ls -la
   ```

2. **If `go.mod` doesn't exist, ask user if they want to initialize:**
   - "Should I initialize this as a Go module? What should the module path be?"

3. **Before implementing features:**
   - Read existing code to understand patterns
   - Check for configuration files
   - Understand the project's purpose from user

4. **After making changes:**
   - Run `go fmt ./...`
   - Run `go vet ./...`
   - Run `go test ./...`
   - Review changes with `git diff`

5. **Before committing:**
   - Verify all tests pass
   - Ensure no secrets are being committed
   - Check that `.gitignore` patterns are respected

---

## Project Status & Next Steps

**Current Status:** Repository initialized with basic files

**Suggested Next Steps:**
1. Initialize Go module (`go mod init`)
2. Create basic project structure (`cmd/`, `internal/`, `pkg/`)
3. Add README.md with project description
4. Set up initial application scaffold
5. Add basic tests
6. Configure CI/CD (GitHub Actions, GitLab CI, etc.)

---

## Additional Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Go Proverbs](https://go-proverbs.github.io/)
- [GPL v3 Full Text](./LICENSE)

---

**Last Updated:** 2026-01-18
**Version:** 1.0.0
**Maintained by:** AI Assistants working with this repository
