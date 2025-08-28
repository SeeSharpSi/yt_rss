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

## Transcript Package

The `pkg/transcript` package provides YouTube transcript fetching functionality, rewritten from the Python youtube-transcript-api library.

### Usage

```go
import "yt_rss2/pkg/transcript"

// Create API instance
api := transcript.NewYouTubeTranscriptApi()

// Fetch transcript
transcript, err := api.Fetch("VIDEO_ID", []string{"en"}, false)
if err != nil {
    // handle error
}

// Format transcript
formatter := &transcript.TextFormatter{}
text := formatter.FormatTranscript(transcript, nil)
fmt.Println(text)
```

### Available Formatters

- `JSONFormatter`: Outputs transcript as JSON
- `TextFormatter`: Outputs plain text
- `WebVTTFormatter`: Outputs WebVTT format

### CLI Usage

The package includes a `RunCLI` function for command-line usage:

```go
transcript.RunCLI(os.Args)
```

Command-line options:
- `--languages <lang1,lang2>`: Specify language codes (default: en)
- `--format <json|text|vtt>`: Output format (default: text)
- `--list-transcripts`: List available transcripts
