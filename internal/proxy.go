package internal

import (
	"bytes"
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

	var bodyReader io.Reader

	// Log incoming request
	if config.Debug {
		log.Printf("[%s] Incoming request: %s %s (target: %s)", requestID, r.Method, r.URL.Path, target.TargetHost)
		log.Printf("[%s] Request headers:", requestID)
		for name, values := range r.Header {
			for _, value := range values {
				// Mask sensitive headers
				if strings.ToLower(name) == "authorization" || strings.ToLower(name) == "x-api-key" {
					log.Printf("[%s]   %s: [REDACTED]", requestID, name)
				} else {
					log.Printf("[%s]   %s: %s", requestID, name, value)
				}
			}
		}
		log.Printf("[%s] Query params: %s", requestID, r.URL.RawQuery)
		log.Printf("[%s] Content-Length: %d", requestID, r.ContentLength)
		log.Printf("[%s] Remote addr: %s", requestID, r.RemoteAddr)

		if r.Body != nil {
			// In debug mode, read body for logging
			requestBody, err := io.ReadAll(r.Body)
			_ = r.Body.Close()
			if err != nil {
				http.Error(w, "Failed to read request body: "+err.Error(), http.StatusInternalServerError)
				log.Printf("[%s] Error reading request body: %v", requestID, err)
				return
			}

			if len(requestBody) > 0 {
				const maxBodyLog = 4096
				if len(requestBody) > maxBodyLog {
					log.Printf("[%s] Request body (%d bytes, truncated): %s...", requestID, len(requestBody), string(requestBody[:maxBodyLog]))
				} else {
					log.Printf("[%s] Request body (%d bytes): %s", requestID, len(requestBody), string(requestBody))
				}
				bodyReader = bytes.NewReader(requestBody)
			}
		}
	} else {
		log.Printf("[%s] Incoming request: %s %s -> %s", requestID, r.Method, r.URL.Path, target.TargetHost)
		// Without debug, pass body directly to preserve streaming
		bodyReader = r.Body
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

	// Create a context with timeout for the request
	ctx, cancel := context.WithTimeout(r.Context(), config.RequestTimeout)
	defer cancel()

	// Create a new request to the target URL
	targetReq, err := http.NewRequestWithContext(ctx, r.Method, targetURL.String(), bodyReader)
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

	// Add X-Forwarded headers (unless disabled)
	if !config.DisableForwardedFor {
		targetReq.Header.Set("X-Forwarded-For", getClientIP(r))
		targetReq.Header.Set("X-Forwarded-Proto", getScheme(r))
	}

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
	log.Printf("[%s] Sending request to %s", requestID, targetURL.String())
	resp, err := client.Do(targetReq)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := "Failed to execute request: " + err.Error()

		// Check for context deadline exceeded
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			statusCode = http.StatusGatewayTimeout
			errMsg = "Request timed out"
		}

		http.Error(w, errMsg, statusCode)
		log.Printf("[%s] Error executing request: %v", requestID, err)
		return
	}
	defer resp.Body.Close()

	if config.Debug {
		log.Printf("[%s] Received response: status=%d, content-type=%s, content-length=%s",
			requestID, resp.StatusCode, resp.Header.Get("Content-Type"), resp.Header.Get("Content-Length"))
	}

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
	if config.Debug {
		log.Printf("[%s] Starting response streaming (SSE=%v)", requestID, isSSE)
	}
	bytesTransferred, err := streamResponse(w, resp.Body, isSSE, requestID)

	// Calculate request duration
	duration := time.Since(requestStart)

	// Log completion
	if err != nil && errors.Is(err, io.EOF) {
		log.Printf("[%s] Error streaming response: %v", requestID, err)
	}

	log.Printf("[%s] Completed request: %s %s in %s, transferred %d bytes", requestID, r.Method, r.URL.Path, duration, bytesTransferred)
}

// Helper function to stream the response
func streamResponse(w http.ResponseWriter, body io.ReadCloser, isSSE bool, requestID string) (int64, error) {
	var totalBytes int64
	buf := make([]byte, 4096)
	readCount := 0

	for {
		n, err := body.Read(buf)
		totalBytes += int64(n)
		readCount++

		if n > 0 {
			// Write the data to the response writer
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				log.Printf("[%s] Error writing to client: %v", requestID, writeErr)
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
				log.Printf("[%s] Error reading response body after %d reads, %d bytes: %v", requestID, readCount, totalBytes, err)
				return totalBytes, err
			}
			log.Printf("[%s] Finished streaming: %d reads, %d bytes", requestID, readCount, totalBytes)
			break
		}
	}

	return totalBytes, nil
}
