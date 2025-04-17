package internal

import (
	"net"
	"net/http"
	"strings"
)

// Helper function to copy HTTP headers
func copyHeaders(dst, src http.Header) {
	for name, values := range src {
		// Don't copy hop-by-hop headers
		if isHopByHopHeader(name) {
			continue
		}

		for _, value := range values {
			dst.Add(name, value)
		}
	}
}

// Helper function to check if a header is a hop-by-hop header
func isHopByHopHeader(header string) bool {
	header = strings.ToLower(header)
	hopByHopHeaders := map[string]bool{
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailer":             true,
		"transfer-encoding":   true,
		"upgrade":             true,
	}

	return hopByHopHeaders[header]
}

// Helper function to get the client IP
func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// The XFF header can contain a comma-separated list of IPs
		// The leftmost IP is the original client IP
		ips := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(ips[0])
		return clientIP
	}

	// Get the IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If there's an error, just return the RemoteAddr as is
		return r.RemoteAddr
	}

	return ip
}

// Helper function to determine the scheme (http/https)
func getScheme(r *http.Request) string {
	// Check the X-Forwarded-Proto header
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}

	// Check if the request is using TLS
	if r.TLS != nil {
		return "https"
	}

	return "http"
}
