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


func TestFilesystemActionsAreExplicit(t *testing.T) {
	cfg := &Config{
		Version: 1, Mode: "scoped",
		Filesystem: FilesystemPolicy{
			Read: []string{"/opt/app"}, Write: []string{"/opt/app"},
			Actions: []string{"read", "write", "patch", "chmod"},
		},
	}
	if !cfg.CanFilesystem("patch") || !cfg.CanFilesystem("chmod") {
		t.Fatal("explicit filesystem capabilities were not allowed")
	}
	if cfg.CanFilesystem("remove_recursive") || cfg.CanFilesystem("chown") {
		t.Fatal("unselected destructive filesystem capability was allowed")
	}
}

func TestResourceDimensionsAreIndependent(t *testing.T) {
	cfg := &Config{
		Version: 1, Mode: "scoped",
		Services: ResourcePolicy{Manage: []string{"app.service"}, Actions: []string{"restart"}},
		Docker: ResourcePolicy{Manage: []string{"app"}, Actions: []string{"restart"}},
		Compose: ResourcePolicy{Manage: []string{"/opt/app"}, Actions: []string{"up"}},
		Packages: ResourcePolicy{Manage: []string{"hello"}, Actions: []string{"install"}},
		Users: ResourcePolicy{Manage: []string{"ciuser"}, Actions: []string{"lock"}},
		Groups: ResourcePolicy{Manage: []string{"cigroup"}, Actions: []string{"add"}},
		Firewall: ResourcePolicy{Actions: []string{"allow"}},
		Diagnostics: []string{"system.disk"},
	}
	if !cfg.CanService("app.service", "restart") || cfg.CanService("ssh.service", "restart") {
		t.Fatal("systemd scope leak")
	}
	if !cfg.CanDocker("app", "restart") || cfg.CanDocker("db", "restart") {
		t.Fatal("docker scope leak")
	}
	if !cfg.CanCompose("/opt/app", "up") || cfg.CanCompose("/opt/other", "up") {
		t.Fatal("compose scope leak")
	}
	if !cfg.CanPackage("hello", "install") || cfg.CanPackage("bash", "install") {
		t.Fatal("package scope leak")
	}
	if !cfg.CanUser("ciuser", "lock") || cfg.CanUser("root", "lock") {
		t.Fatal("user scope leak")
	}
	if !cfg.CanGroup("cigroup", "add") || cfg.CanGroup("sudo", "add") {
		t.Fatal("group scope leak")
	}
	if !cfg.CanFirewall("allow") || cfg.CanFirewall("deny") {
		t.Fatal("firewall scope leak")
	}
	if !cfg.CanDiagnostic("system.disk") || cfg.CanDiagnostic("process.list") {
		t.Fatal("diagnostic scope leak")
	}
}

func TestShellCWDAndNetworkScope(t *testing.T) {
	cfg := &Config{
		Version: 1, Mode: "scoped",
		Shell: ShellPolicy{Enabled: true, CWDRoots: []string{"/opt/app"}},
		Network: NetworkPolicy{Mode: "allowlist", Destinations: []string{"api.example.com:443"}},
	}
	if !cfg.CanShellCWD("/opt/app/sub") || cfg.CanShellCWD("/etc") {
		t.Fatal("shell cwd scope is incorrect")
	}
	if !cfg.CanNetworkDestination("api.example.com:443") || cfg.CanNetworkDestination("evil.example:443") {
		t.Fatal("network destination scope is incorrect")
	}
}


func FuzzPolicyLoadNeverPanics(f *testing.F) {
	f.Add("version: 1\nmode: scoped\n")
	f.Add("version: nope\nmode: full\n")
	f.Add("{{{{")
	f.Fuzz(func(t *testing.T, body string) {
		if len(body) > 1<<20 {
			t.Skip()
		}
		p := filepath.Join(t.TempDir(), "policy.yaml")
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		_, _ = Load(p)
	})
}
