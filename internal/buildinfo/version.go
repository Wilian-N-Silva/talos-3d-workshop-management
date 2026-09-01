// Package buildinfo exposes server build compatibility metadata.
package buildinfo

const (
	// MinimumDesktopVersion is the oldest desktop release accepted by this server.
	MinimumDesktopVersion = "0.0.0"
)

// ServerVersion is replaced by release builds through -ldflags -X.
var ServerVersion = "dev"
