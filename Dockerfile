FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum* ./

# Download dependencies (if go.sum exists)
RUN if [ -f go.sum ]; then go mod download; fi

# Copy source code
COPY . ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o proxyai .

# Use distroless as minimal base image to package the proxy binary
FROM gcr.io/distroless/static:nonroot

# Set working directory
WORKDIR /

# Copy the binary from builder
COPY --from=builder /app/proxyai /proxyai

# Use non-root user
USER nonroot:nonroot

# Expose default port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/proxyai"]