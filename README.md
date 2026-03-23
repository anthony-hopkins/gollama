# gollama

`gollama` is a Go-based client library and CLI application for interacting with LLM servers (like Ollama or other compatible APIs) using both streaming and non-streaming modes.

## Features

- **Client Library**: A reusable Go package (`client`) for making chat completions.
- **CLI Application**: A command-line interface to interact with the LLM from your terminal.
- **Streaming Support**: Real-time token streaming for a more interactive experience.
- **Non-streaming Support**: Standard request-response for programmatic use.
- **Modern Go**: Built with Go 1.22, leveraging modern idioms and standard library features.

## Installation

Ensure you have [Go 1.22+](https://go.dev/dl/) installed.

### Clone the repository

```bash
git clone https://github.com/anthony-hopkins/gollama.git
cd gollama
```

### Build the application

```bash
go build -o gollama cmd/app/main.go
```

## Usage

The CLI application requires a `baseURL` and a `prompt`. It supports two modes: `stream` and `nonstream`.

### Non-streaming Mode (Default)

```bash
./gollama --baseURL http://localhost:11434 --prompt "What is the capital of France?" --mode nonstream
```

### Streaming Mode

```bash
./gollama --baseURL http://localhost:11434 --prompt "Write a short poem about Go programming." --mode stream
```

### Flags

- `--baseURL`: The base URL of your LLM server (e.g., `http://localhost:11434/v1`).
- `--prompt`: The message you want to send to the model.
- `--mode`: Either `stream` or `nonstream` (default: `nonstream`).

## Project Structure

- `client/`: Contains the core logic for interacting with the LLM API.
  - `client.go`: Implementation of the HTTP client, request/response structures, and methods for chat completions.
- `cmd/app/`: The entry point for the CLI application.
  - `main.go`: Handles CLI flags and coordinates with the `client` package.
- `go.mod`: Go module definition.

## Requirements

- Go 1.22 or higher.
- An LLM server running and accessible via HTTP (e.g., [Ollama](https://ollama.com/)).

## License

[Add License Information Here, e.g., MIT]
