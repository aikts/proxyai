package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aikts/proxyai/internal"
)

func main() {
	// Parse flags
	flag.Parse()

	// Get the custom targets flag value after parsing
	customTargets := flag.Lookup("targets").Value.String()

	// Process custom targets from command line if provided
	if customTargets != "" {
		processCustomTargets(customTargets)
	}

	// Load additional or override proxy targets from environment variables
	loadProxyTargetsFromEnv()

	// Set up logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Log configuration
	logConfig()

	// Register proxy handlers
	registerHandlers()

	// Configure the server with timeouts
	server := &http.Server{
		Addr:              config.ListenAddr,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		Handler:           nil, // Use default ServeMux
	}

	// Set up graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown failed: %v", err)
		}
	}()

	// Start the server
	log.Printf("Starting multi-API proxy server on %s", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}

// Register all HTTP handlers
func registerHandlers() {
	// Create handlers for each proxy target path
	for _, proxyTarget := range config.ProxyTargets {
		// Create a copy of the target for the closure
		http.HandleFunc(proxyTarget.PathPrefix, func(w http.ResponseWriter, r *http.Request) {
			internal.ProxyHandler(w, r, proxyTarget, config)
		})

		log.Printf("Registered handler for %s -> %s", proxyTarget.PathPrefix, proxyTarget.TargetHost)
	}

	// Add a health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}
