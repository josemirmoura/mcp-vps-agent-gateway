// Package portico exposes the product version embedded from the canonical VERSION
// manifest at build time. This is independent from the negotiated MCP protocol version.
package portico

import (
	_ "embed"
	"regexp"
	"strings"
)

//go:embed VERSION
var manifestVersion string

// Product versions use SemVer without a leading v in VERSION. Unusable or
// missing build metadata must never be represented as a stable release.
var productSemver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

// Version reports the packaged Portico source version (including -rc.N).
// An immutable Git tag and successful release gate are still required to
// establish that a binary is a published release.
func Version() string {
	return versionFromManifest(manifestVersion)
}

func versionFromManifest(raw string) string {
	v := strings.TrimSpace(raw)
	parts := productSemver.FindStringSubmatch(v)
	if parts == nil {
		return "dev"
	}
	// Numeric SemVer prerelease identifiers cannot have leading zeroes.
	if parts[4] != "" {
		for _, identifier := range strings.Split(parts[4], ".") {
			if len(identifier) > 1 && identifier[0] == '0' {
				numeric := true
				for _, digit := range identifier {
					if digit < '0' || digit > '9' {
						numeric = false
						break
					}
				}
				if numeric {
					return "dev"
				}
			}
		}
	}
	return "v" + v
}
