package types

import "time"

// ProxyTarget defines a proxy mapping from path prefix to target host
type ProxyTarget struct {
	PathPrefix string
	TargetHost string
}

// Config represents the global application configuration
type Config struct {
	ProxyTargets      []ProxyTarget
	ListenAddr        string
	RequestTimeout    time.Duration
	ReadHeaderTimeout time.Duration
	Debug             bool
}
