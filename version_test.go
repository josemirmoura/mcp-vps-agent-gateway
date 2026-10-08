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
