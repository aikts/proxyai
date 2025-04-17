package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aikts/proxyai/internal/types"
)

// Default configuration values
var defaultConfig = types.Config{
	ProxyTargets: []types.ProxyTarget{
		{PathPrefix: "/openai/", TargetHost: "api.openai.com"},
		{PathPrefix: "/anthropic/", TargetHost: "api.anthropic.com"},
		{PathPrefix: "/gemini/", TargetHost: "generativelanguage.googleapis.com"},
	},
	ListenAddr:        ":8080",
	RequestTimeout:    60 * time.Second,
	ReadHeaderTimeout: 10 * time.Second,
	Debug:             false,
}

// Global configuration
var config types.Config

func init() {
	// Parse command line flags for non-proxy target options
	flag.StringVar(&config.ListenAddr, "listen", defaultConfig.ListenAddr, "Listen address")
	flag.DurationVar(&config.RequestTimeout, "timeout", defaultConfig.RequestTimeout, "Request timeout")
	flag.DurationVar(&config.ReadHeaderTimeout, "header-timeout", defaultConfig.ReadHeaderTimeout, "Read header timeout")
	flag.BoolVar(&config.Debug, "debug", defaultConfig.Debug, "Enable debug logging")

	// Custom proxy target definition flag
	var customTargets string
	flag.StringVar(&customTargets, "targets", "", "Custom proxy targets in format: /path1/:host1,/path2/:host2")

	// Initialize with default proxy targets
	config.ProxyTargets = defaultConfig.ProxyTargets
}

// processCustomTargets processes the custom targets flag
func processCustomTargets(targetsStr string) {
	// Create a map to detect duplicates
	pathPrefixMap := make(map[string]bool)

	// First add defaults to the map
	for _, target := range config.ProxyTargets {
		pathPrefixMap[target.PathPrefix] = true
	}

	// Split by comma to get individual target definitions
	targetDefs := strings.Split(targetsStr, ",")

	// Process each target definition
	for _, targetDef := range targetDefs {
		targetParts := strings.SplitN(targetDef, ":", 2)
		if len(targetParts) != 2 {
			log.Printf("Warning: Invalid proxy target definition: %s", targetDef)
			continue
		}

		pathPrefix := targetParts[0]
		targetHost := targetParts[1]

		// Ensure the path prefix ends with a slash
		if !strings.HasSuffix(pathPrefix, "/") {
			pathPrefix += "/"
		}

		// Check for duplicate path prefixes
		if _, exists := pathPrefixMap[pathPrefix]; exists {
			// Replace the existing target with this one
			for i, target := range config.ProxyTargets {
				if target.PathPrefix == pathPrefix {
					config.ProxyTargets[i].TargetHost = targetHost
					log.Printf("Overriding proxy target for %s: %s", pathPrefix, targetHost)
					break
				}
			}
		} else {
			// Add a new target
			config.ProxyTargets = append(config.ProxyTargets, types.ProxyTarget{
				PathPrefix: pathPrefix,
				TargetHost: targetHost,
			})
			pathPrefixMap[pathPrefix] = true
			log.Printf("Added proxy target from command line: %s -> %s", pathPrefix, targetHost)
		}
	}
}

// loadProxyTargetsFromEnv loads proxy targets from environment variables
func loadProxyTargetsFromEnv() {
	// Create a map to detect duplicates
	pathPrefixMap := make(map[string]bool)

	// First add defaults to the map
	for _, target := range config.ProxyTargets {
		pathPrefixMap[target.PathPrefix] = true
	}

	// Look for environment variables with the PROXY_TARGET_ prefix
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "PROXY_TARGET_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}

			targetDef := parts[1]
			targetParts := strings.SplitN(targetDef, ":", 2)
			if len(targetParts) != 2 {
				log.Printf("Warning: Invalid proxy target definition: %s", targetDef)
				continue
			}

			pathPrefix := targetParts[0]
			targetHost := targetParts[1]

			// Ensure the path prefix ends with a slash
			if !strings.HasSuffix(pathPrefix, "/") {
				pathPrefix += "/"
			}

			// Check for duplicate path prefixes
			if _, exists := pathPrefixMap[pathPrefix]; exists {
				// Replace the existing target with this one
				for i, target := range config.ProxyTargets {
					if target.PathPrefix == pathPrefix {
						config.ProxyTargets[i].TargetHost = targetHost
						log.Printf("Overriding proxy target for %s: %s", pathPrefix, targetHost)
						break
					}
				}
			} else {
				// Add a new target
				config.ProxyTargets = append(config.ProxyTargets, types.ProxyTarget{
					PathPrefix: pathPrefix,
					TargetHost: targetHost,
				})
				pathPrefixMap[pathPrefix] = true
				log.Printf("Added proxy target from environment: %s -> %s", pathPrefix, targetHost)
			}
		}
	}
}

// logConfig logs the current configuration
func logConfig() {
	log.Printf("Configuration:")
	log.Printf("  Listen Address: %s", config.ListenAddr)
	log.Printf("  Request Timeout: %s", config.RequestTimeout)
	log.Printf("  Read Header Timeout: %s", config.ReadHeaderTimeout)
	log.Printf("  Debug Mode: %v", config.Debug)
	log.Printf("  Proxy Targets:")

	for _, target := range config.ProxyTargets {
		log.Printf("    %s -> %s", target.PathPrefix, target.TargetHost)
	}
}
