package sandbox

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func TestBuildSystemdRunArgsBlockedNetwork(t *testing.T) {
	args, err := BuildSystemdRunArgs(Spec{
		Unit: "vps-agent-job-123", User: "vps-agent-exec",
		Command: "echo ok", CWD: "/srv/app",
		ReadOnlyPaths: []string{"/srv/app"}, ReadWritePaths: []string{"/srv/app"}, InaccessiblePaths: []string{"/etc"}, Runtime: time.Minute,
		NetworkMode: "blocked",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--property=NoNewPrivileges=yes",
		"--property=ProtectSystem=strict",
		"--property=PrivateNetwork=yes",
		"--property=ReadOnlyPaths=/srv/app",
		"--property=ReadWritePaths=/srv/app",
		"--property=InaccessiblePaths=/etc",
		"--property=ProtectProc=invisible",
		"--property=PrivateDevices=yes",
	}
	for _, w := range want {
		if !slices.Contains(args, w) {
			t.Errorf("missing %q in %v", w, args)
		}
	}
	got := strings.Join(args[len(args)-4:], " ")
	if got != "-- /bin/sh -c echo ok" {
		t.Fatalf("unexpected command tail %q", got)
	}
}

func TestBuildSystemdRunArgsAllowlistFailsClosed(t *testing.T) {
	_, err := BuildSystemdRunArgs(Spec{
		Unit: "job", Command: "true", CWD: "/tmp", NetworkMode: "allowlist",
	})
	if err == nil {
		t.Fatal("network allowlist must fail closed until implemented")
	}
}
