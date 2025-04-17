# Multi AI Proxy

A transparent HTTP proxy written in Go that forwards requests to various AI API providers.

## Features

- Transparently proxies requests to multiple API endpoints:
  - `/openai/` → `api.openai.com`
  - `/anthropic/` → `api.anthropic.com`
  - `/gemini/` → `generativelanguage.googleapis.com`
- Supports streaming responses with Server-Sent Events (SSE)
- Preserves headers, query parameters, and request bodies
- Configurable via command-line flags and environment variables
- Built-in health check endpoint
- Detailed logging and debug mode

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/proxyai.git
cd proxyai

# Build the binary
go build -o proxyai .
```

## Usage

### Basic Usage

```bash
# Start with default settings
./proxyai

# Specify custom listen port
./proxyai --listen=:9000

# Enable debug mode
./proxyai --debug=true
```

### Custom Proxy Targets

You can define custom proxy targets using the `--targets` flag:

```bash
# Add or override proxy targets
./proxyai --targets="/azure/:api.azure.openai.com,/cohere/:api.cohere.ai"
```

### Environment Variables

Proxy targets can also be defined using environment variables:

```bash
# Define a custom proxy target
export PROXY_TARGET_MISTRAL=/mistral/:api.mistral.ai

# Run the proxy
./proxyai
```

## Configuration Options

| Flag             | Environment Variable | Default     | Description                                      |
|------------------|----------------------|-------------|--------------------------------------------------|
| `--listen`       | -                    | `:8080`     | Address and port to listen on                    |
| `--timeout`      | -                    | `60s`       | Request timeout                                  |
| `--header-timeout` | -                  | `10s`       | Read header timeout                              |
| `--debug`        | -                    | `false`     | Enable debug logging                             |
| `--targets`      | -                    | -           | Custom proxy targets (format: `/path/:host,...`) |
| -                | `PROXY_TARGET_*`     | -           | Custom proxy targets (format: `/path/:host`)     |

## Default API Endpoints

| Path Prefix      | Target Host                         |
|------------------|-------------------------------------|
| `/openai/`       | `api.openai.com`                    |
| `/anthropic/`    | `api.anthropic.com`                 |
| `/gemini/`       | `generativelanguage.googleapis.com` |

## Example Requests

### OpenAI API

```bash
# Original API call
curl -X POST https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}'

# Using the proxy
curl -X POST http://localhost:8080/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}'
```

### Anthropic API

```bash
# Original API call
curl -X POST https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model": "claude-3-opus-20240229", "max_tokens": 1000, "messages": [{"role": "user", "content": "Hello"}]}'

# Using the proxy
curl -X POST http://localhost:8080/anthropic/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model": "claude-3-opus-20240229", "max_tokens": 1000, "messages": [{"role": "user", "content": "Hello"}]}'
```

## Project Structure

- `main.go` - Entry point and server setup
- `config.go` - Configuration handling
- `proxy.go` - Proxy functionality
- `utils.go` - Helper utilities

## License

Apache License 2.0