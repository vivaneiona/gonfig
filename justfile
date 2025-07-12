# getconfig Module Justfile
# Basic Go commands for getconfig dependency
set shell := ["/bin/bash", "-c"]

import 'version.just'

# Show all available commands (default action)
default:
    @just --list

# Format Go code
fmt:
    go fmt ./...

# Vet Go code for potential issues
vet:
    go vet ./...

# Run tests with verbose output
test:
    go test -v ./...

# Run tests with code coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated in coverage.html"

# Clean up build artifacts
clean:
    rm -f coverage.out coverage.html
    rm -rf vendor

# Update and vendor dependencies
deps:
    go mod tidy
    go mod vendor