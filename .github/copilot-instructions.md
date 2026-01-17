# Copilot Instructions for gh-dormant-users

This document provides guidelines for AI-assisted code contributions to the `gh-dormant-users` project.

## Project Overview

`gh-dormant-users` is a GitHub CLI extension written in Go that identifies dormant users in organizations by analyzing various activity types (commits, issues, issue-comments, pr-comments) over a configurable time period.

## Technology Stack

- **Language**: Go 1.22.5+
- **CLI Framework**: Cobra (command-line interface)
- **GitHub Integration**: `github.com/cli/go-gh` (GitHub REST API)
- **UI/Output**: pterm (pretty terminal output)
- **Architecture**: Modular internal packages with separation of concerns

## Code Organization

```
.
├── main.go                          # Entry point
├── cmd/
│   ├── root.go                      # Root command setup
│   └── report.go                    # Report command implementation
├── internal/
│   ├── activity/                    # Activity checking logic
│   ├── commits/                     # Commit analysis
│   ├── issues/                      # Issue analysis
│   ├── pullrequests/                # PR analysis
│   ├── users/                       # User management
│   ├── repository/                  # Repository management
│   ├── date/                        # Date utilities
│   └── limiter/                     # Rate limiting
└── .github/
    ├── workflows/                   # GitHub Actions
    └── release-drafter.yml          # Release automation
```

## Development Guidelines

### Go Code Style

- Follow standard Go conventions (https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Use clear, descriptive variable and function names
- Keep functions focused and reasonably sized
- Add comments for exported functions and complex logic

### Package Structure

- **cmd/**: Command definitions and CLI argument handling
- **internal/**: Core business logic organized by feature/domain
  - Each package should have a single responsibility
  - Use meaningful package names that reflect their purpose
  - Consider using type definitions and methods for domain modeling (e.g., `User`, `Users` types)

### Function Guidelines

- Use the GitHub REST API through `github.com/cli/go-gh`
- Handle errors explicitly; avoid silent failures
- Use `pterm` for user-facing output (info, warnings, errors)
- Implement goroutines with proper synchronization (e.g., `sync.WaitGroup`) for parallel operations
- Use `sync.Mutex` when managing shared state across goroutines

### Testing & Building

- **Build**: `go build` must complete without errors
- **Unit Tests**: 
  - **Required**: Any modification to an existing package or addition of a new package must include comprehensive unit tests
  - **Coverage**: Changes must maintain or improve code coverage to above 80%
  - Run tests with: `go test ./...`
  - Run coverage check with: `go test -cover ./...`
  - Generate coverage report: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`
- **Validation**:
  - Ensure the tool continues to produce valid CSV output with the correct schema
  - All error paths should be tested
  - Test with edge cases (empty organizations, no activity, etc.)
- **Integration Testing**:
  - Test with real organization data if possible
  - Verify no breaking changes to CLI flags or output format

## Common Patterns in This Project

### API Calls with Error Handling
```go
client, err := gh.RESTClient(nil)
if err != nil {
    pterm.Error.Printf("Failed to create REST client: %v\n", err)
    os.Exit(1)
}
```

### Parallel Data Fetching
Use goroutines with `sync.WaitGroup` for concurrent API calls while respecting GitHub API rate limits via the `limiter` package.

### User-Facing Output
Always use `pterm` for console output:
- `pterm.Info.Printf()` for informational messages
- `pterm.Error.Printf()` for error messages
- `pterm.DefaultBox` for structured output
- Progress spinners for long-running operations

### CSV Schema
The tool outputs a CSV with columns: `Username`, `Email`, `Active`, `ActivityTypes`. Maintain this schema in any modifications.

## Contribution Areas

**Safe to modify:**
- Internal package logic (analysis algorithms, new activity checks)
- Command flag definitions and help text
- Output formatting and UI improvements
- Performance optimizations (respecting API limits)
- Error handling and validation
- **Note**: Each change requires corresponding unit tests with >80% coverage

**Be cautious with:**
- CLI command structure (changes may break existing workflows)
- CSV output schema (downstream users may depend on current structure)
- GitHub API version changes
- **Note**: Changes require updated tests and coverage verification

**Do not modify without explicit request:**
- Release automation workflow
- License and copyright headers

## Dependencies

- `github.com/cli/go-gh`: GitHub REST API client
- `github.com/pterm/pterm`: Terminal UI library
- `github.com/spf13/cobra`: CLI framework

Avoid introducing new dependencies without justification. Discuss in an issue first if adding new external packages.

## Documentation

- Ensure CLI flag descriptions are clear and match the README
- Update README if adding new flags or commands
- Add comments to exported functions and types
- Document complex algorithms or non-obvious logic

## Key Files to Reference

- **README.md**: User-facing documentation and usage examples
- **cmd/report.go**: Main command logic and flag definitions
- **internal/activity/activity.go**: Core activity checking logic
- **go.mod**: Dependency management

## Testing Guidelines

### Test File Naming and Location
- Test files must be in the same package as the code being tested
- Follow Go convention: `filename.go` → `filename_test.go`
- Example: `internal/users/users.go` → `internal/users/users_test.go`

### Test Requirements
- **Mandatory**: Every new package or modification to existing packages requires unit tests
- **Coverage Target**: Minimum 80% code coverage for all changes
- **Error Cases**: Test both success and failure paths
- **Mock External Dependencies**: Use mock clients for GitHub API calls to avoid rate limiting in tests

### Test Execution
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/users/...
```

### Testing Best Practices
- Use table-driven tests for multiple scenarios
- Mock the GitHub REST client (`api.RESTClient`) for predictable test results
- Test concurrent operations with proper synchronization
- Verify CSV output format and data integrity
- Document complex test scenarios with clear comments

---

When making contributions, prioritize clarity, maintainability, and respecting the existing patterns in the codebase.
