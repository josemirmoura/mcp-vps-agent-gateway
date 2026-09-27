package hostexec

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPathWithoutHostRoot(t *testing.T) {
	t.Setenv(envHostRoot, "")
	if got := Path("/opt/app"); got != "/opt/app" {
		t.Fatalf("got %q", got)
	}
}

func TestPathWithHostRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv(envHostRoot, root)
	want := filepath.Join(root, "opt", "app")
	if got := Path("/opt/app"); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCommandUsesChrootWhenConfigured(t *testing.T) {
	root := t.TempDir()
	t.Setenv(envHostRoot, root)
	name, args := commandArgs("systemctl", "status", "x.service")
	if name != "chroot" {
		t.Fatalf("name=%q", name)
	}
	want := []string{root, "systemctl", "status", "x.service"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args=%v want=%v", args, want)
	}
}

func TestInvalidRelativeRootFailsClosedToNative(t *testing.T) {
	t.Setenv(envHostRoot, "relative")
	if Root() != "" {
		t.Fatal("relative host root should be ignored")
	}
	_ = os.Getenv(envHostRoot)
}


func TestCommandUsesNsenterWhenConfigured(t *testing.T) {
	root := t.TempDir()
	t.Setenv(envHostRoot, root)
	t.Setenv("VPS_AGENT_HOST_NSENTER", "1")
	name, args := commandArgs("systemd-run", "--unit=x", "--", "/bin/true")
	if name != "nsenter" {
		t.Fatalf("name=%q", name)
	}
	wantPrefix := []string{
		"--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid", "--cgroup",
		"--root=/proc/1/root", "--wdns=/", "--", "systemd-run",
	}
	if len(args) < len(wantPrefix) || !reflect.DeepEqual(args[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("args=%v want prefix=%v", args, wantPrefix)
	}
}
