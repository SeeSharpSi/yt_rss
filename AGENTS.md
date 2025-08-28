# Agent Guidelines for yt_rss2

This document outlines the conventions and commands for agentic coding in the `yt_rss2` repository.

## Build, Lint, and Test Commands

*   **Build:** `go build`
*   **Generate Templ Files:** `templ generate` or `go run github.com/a-h/templ/cmd/templ generate`
*   **Run All Tests:** `go test ./...`
*   **Run Single Test:** `go test ./path/to/package -run <TestName>`
*   **Format Code:** `go fmt ./...`
*   **Tidy Dependencies:** `go mod tidy`

## Code Style Guidelines

*   **Imports:** Follow standard Go import grouping and ordering. Use `go mod tidy` to manage dependencies.
*   **Formatting:** Adhere to `go fmt` standards.
*   **Naming Conventions:** Use `CamelCase` for exported identifiers (functions, types, variables) and `camelCase` for unexported identifiers.
*   **Error Handling:** Use Go's idiomatic error handling by returning `error` as the last return value and checking `if err != nil`.
*   **Types:** Utilize Go's strong typing system consistently.
