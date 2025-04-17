package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aikts/proxyai/internal/types"
)

// ProxyHandler handles the proxying of HTTP requests
func ProxyHandler(w http.ResponseWriter, r *http.Request, target types.ProxyTarget, config types.Config) {
	requestStart := time.Now()
	requestID := fmt.Sprintf("%s-%d", r.Method, requestStart.UnixNano())

	// Log incoming request
	if config.Debug {
		log.Printf("[%s] Incoming request: %s %s (target: %s)", requestID, r.Method, r.URL.Path, target.TargetHost)
	} else {
		log.Printf("Incoming request: %s %s", r.Method, r.URL.Path)
	}

	// Extract the actual API path by removing the proxy prefix
	apiPath := strings.TrimPrefix(r.URL.Path, target.PathPrefix)

	// Construct the target URL
	targetURL := url.URL{
		Scheme:   "https",
		Host:     target.TargetHost,
		Path:     apiPath,
		RawQuery: r.URL.RawQuery,
	}

	if config.Debug {
		log.Printf("[%s] Proxying to: %s", requestID, targetURL.String())
	}

	// Create a context with timeout for the request
	ctx, cancel := context.WithTimeout(r.Context(), config.RequestTimeout)
	defer cancel()

	// Create a new request to the target URL
	targetReq, err := http.NewRequestWithContext(ctx, r.Method, targetURL.String(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
		log.Printf("[%s] Error creating request: %v", requestID, err)
		return
	}

	// Copy all headers from the original request
	for name, values := range r.Header {
		// Skip the Connection header to avoid issues with proxying
		if strings.ToLower(name) == "connection" {
			continue
		}

		for _, value := range values {
			targetReq.Header.Add(name, value)
		}
	}

	// Add X-Forwarded headers
	targetReq.Header.Set("X-Forwarded-For", getClientIP(r))
	targetReq.Header.Set("X-Forwarded-Proto", getScheme(r))

	// Ensure the Host header is set correctly
	targetReq.Host = target.TargetHost

	// Create a HTTP client with appropriate configuration
	client := &http.Client{
		// No timeout here as we want to support long-running streaming responses
		// The context will handle the timeout
		Timeout: 0,
		// Configure transport with proper TLS settings
		Transport: &http.Transport{
			DisableCompression:    false,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	// Execute the request
	resp, err := client.Do(targetReq)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := "Failed to execute request: " + err.Error()

		// Check for context deadline exceeded
		if ctx.Err() == context.DeadlineExceeded {
			statusCode = http.StatusGatewayTimeout
			errMsg = "Request timed out"
		}

		http.Error(w, errMsg, statusCode)
		log.Printf("[%s] Error executing request: %v", requestID, err)
		return
	}
	defer resp.Body.Close()

	// Copy the response headers back to the client
	copyHeaders(w.Header(), resp.Header)

	// Check if the response should be streamed as SSE
	isSSE := false
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		isSSE = true

		// Ensure proper SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Remove Content-Length as it's not applicable for streaming
		w.Header().Del("Content-Length")

		if config.Debug {
			log.Printf("[%s] Streaming response as SSE", requestID)
		}
	}

	// Set the status code
	w.WriteHeader(resp.StatusCode)

	// Ensure streaming is properly enabled for SSE
	if isSSE {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		} else {
			log.Printf("[%s] Warning: ResponseWriter does not support Flush, streaming may not work properly", requestID)
		}
	}

	// Stream the response body back to the client
	bytesTransferred, err := streamResponse(w, resp.Body, isSSE, requestID)

	// Calculate request duration
	duration := time.Since(requestStart)

	// Log completion
	if err != nil && errors.Is(err, io.EOF) {
		log.Printf("[%s] Error streaming response: %v", requestID, err)
	}

	log.Printf("Completed request: %s %s in %s, transferred %d bytes", r.Method, r.URL.Path, duration, bytesTransferred)
}

// Helper function to stream the response
func streamResponse(w http.ResponseWriter, body io.ReadCloser, isSSE bool, requestID string) (int64, error) {
	var totalBytes int64
	buf := make([]byte, 4096)

	for {
		n, err := body.Read(buf)
		totalBytes += int64(n)

		if n > 0 {
			// Write the data to the response writer
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return totalBytes, fmt.Errorf("error writing response: %w", writeErr)
			}

			// If this is a server-sent event, flush the buffer after each write
			if isSSE {
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}

		if err != nil {
			if err != io.EOF {
				return totalBytes, err
			}
			break
		}
	}

	return totalBytes, nil
}
