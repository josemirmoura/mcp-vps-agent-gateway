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
	return "v" + strings.TrimSpace(manifestVersion)
}
