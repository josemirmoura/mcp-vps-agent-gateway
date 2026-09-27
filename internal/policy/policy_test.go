package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func writePolicy(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "policy.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadRejectsUnknownField(t *testing.T) {
	p := writePolicy(t, "version: 1\nmode: scoped\nunknown_field: true\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestServicePolicy(t *testing.T) {
	cfg := &Config{
		Version: 1, Mode: "scoped",
		Services: ResourcePolicy{
			Inspect: []string{"app-*"},
			Manage:  []string{"vps-agent-test.service"},
			Actions: []string{"status", "restart"},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if !cfg.CanService("vps-agent-test.service", "restart") {
		t.Fatal("expected allowed restart")
	}
	if cfg.CanService("postgres.service", "restart") {
		t.Fatal("unexpected restart permission")
	}
	if !cfg.CanService("app-api", "status") {
		t.Fatal("expected inspect permission")
	}
}

func TestFullCannotEnableWithoutFeatureFlag(t *testing.T) {
	cfg := &Config{Version: 1, Mode: "full", Enabled: true}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected Full feature gate failure")
	}
}
