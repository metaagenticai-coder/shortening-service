# Contributing to URL Shortening Service

Thank you for your interest in contributing to the URL Shortening Service! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/metaagenticai/shortening-service/issues)
2. If not, create a new issue with:
   - Clear description of the bug
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version, etc.)
   - Relevant logs or error messages

### Suggesting Enhancements

1. Check if the enhancement has been suggested in [Issues](https://github.com/metaagenticai/shortening-service/issues)
2. Create a new issue with:
   - Clear description of the enhancement
   - Use case and motivation
   - Proposed implementation approach (if applicable)

### Pull Requests

1. Fork the repository
2. Create a feature branch from `develop`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. Make your changes following our coding standards
4. Write or update tests as needed
5. Ensure all tests pass:
   ```bash
   make test
   ```

6. Run linter and fix any issues:
   ```bash
   make lint
   ```

7. Commit your changes with clear, descriptive messages:
   ```bash
   git commit -m "Add feature: description of change"
   ```

8. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

9. Create a Pull Request to the `develop` branch

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- PostgreSQL 15
- Redis 7.0
- Make

### Setup

1. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/shortening-service.git
   cd shortening-service
   ```

2. Install dependencies:
   ```bash
   make install-deps
   ```

3. Start local infrastructure:
   ```bash
   make docker-compose-up
   ```

4. Run migrations:
   ```bash
   make migrate-up
   ```

5. Run tests:
   ```bash
   make test
   ```

## Coding Standards

### Go Style Guide

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use meaningful variable and function names
- Keep functions small and focused (single responsibility)
- Write idiomatic Go code

### Code Organization

- Place new features in appropriate packages under `internal/`
- Public APIs go in `pkg/`
- Service entry points in `cmd/`
- Keep business logic separate from HTTP handlers

### Testing

- Write unit tests for all new code
- Aim for 80%+ code coverage
- Use table-driven tests where appropriate
- Mock external dependencies
- Write integration tests for critical paths

Example test structure:
```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("FunctionName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Documentation

- Add comments for exported functions, types, and packages
- Use complete sentences in comments
- Include examples in documentation where helpful
- Update README.md if adding new features

### Commit Messages

Follow conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Example:
```
feat(shortcode): add custom short code generation

Implement support for user-specified custom short codes
with validation and collision detection.

Closes #123
```

## Review Process

1. All PRs require at least one approval
2. CI checks must pass
3. Code coverage should not decrease
4. Documentation must be updated if needed

## Questions?

Feel free to open an issue for any questions about contributing!
