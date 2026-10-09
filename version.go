// Package portico exposes the product version embedded from the canonical VERSION
// manifest at build time. This is independent from the negotiated MCP protocol version.
package portico

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var manifestVersion string

// Version reports the actual packaged Portico release (including -rc.N).
func Version() string {
	return versionFromManifest(manifestVersion)
}

// versionFromManifest fails clearly to a development identity when package
// metadata is empty, rather than falsely advertising a stable "v" release.
func versionFromManifest(raw string) string {
	version := strings.TrimSpace(raw)
	if version == "" {
		return "dev"
	}
	return "v" + version
}
