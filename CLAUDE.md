# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

```bash
# Build the binary
go build -o proxyai .

# Run the proxy
./proxyai
./proxyai --listen=:9000 --debug=true

# Lint (includes formatting)
make lint

# Format only
make fmt

# Initialize development environment (install linting tools)
make init

# Manage dependencies
make mod
```

## Architecture

This is a transparent HTTP proxy for AI API providers (OpenAI, Anthropic, Gemini). Requests to `/openai/*` are forwarded to `api.openai.com`, `/anthropic/*` to `api.anthropic.com`, etc.

### Code Organization

- `main.go` - Server startup, graceful shutdown, HTTP handler registration
- `config.go` - Flag parsing, environment variable loading (`PROXY_TARGET_*`), configuration types initialization
- `internal/proxy.go` - Core proxy logic: request construction, header copying, response streaming (including SSE)
- `internal/utils.go` - Header utilities (hop-by-hop filtering), client IP extraction
- `internal/types/types.go` - `Config` and `ProxyTarget` struct definitions

### Request Flow

1. Incoming request hits registered path prefix handler (e.g., `/openai/`)
2. `ProxyHandler` strips the prefix, constructs target URL with `https://` scheme
3. Headers are copied (excluding hop-by-hop headers like `Connection`, `Transfer-Encoding`)
4. X-Forwarded-For/Proto headers added unless `--no-forwarded` flag is set
5. Response streamed back; SSE responses (`text/event-stream`) get special handling with flush-after-write

### Configuration

Proxy targets can be added via:
- `--targets` flag: `/path/:host,/path2/:host2`
- Environment variables: `PROXY_TARGET_NAME=/path/:host`

Both override default targets if path prefix matches.
