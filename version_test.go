package portico

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestVersionMatchesCanonicalManifest(t *testing.T) {
	raw, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatal(err)
	}
	want := "v" + strings.TrimSpace(string(raw))
	if got := Version(); got != want {
		t.Fatalf("Gateway version %q differs from canonical VERSION %q", got, want)
	}
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`).MatchString(want) {
		t.Fatalf("not a SemVer product version: %q", want)
	}
}

func TestVersionFromManifestFailsClosed(t *testing.T) {
	for _, tt := range []struct {
		manifest string
		want     string
	}{
		{"0.1.0-rc.7\n", "v0.1.0-rc.7"},
		{"2.4.1", "v2.4.1"},
		{"1.2.3-beta.2+build.5", "v1.2.3-beta.2+build.5"},
		{"", "dev"},
		{"\n \t", "dev"},
		{"v0.1.0", "dev"},
		{"0.1", "dev"},
		{"01.2.3", "dev"},
		{"0.1.0-rc.07", "dev"},
		{"0.1.0-", "dev"},
		{"0.1.0+bad..metadata", "dev"},
		{"0.1.0\nUNTRUSTED", "dev"},
	} {
		t.Run(strings.ReplaceAll(tt.manifest, "\n", "_newline_"), func(t *testing.T) {
			if got := versionFromManifest(tt.manifest); got != tt.want {
				t.Fatalf("versionFromManifest(%q)=%q; want %q", tt.manifest, got, tt.want)
			}
		})
	}
}
