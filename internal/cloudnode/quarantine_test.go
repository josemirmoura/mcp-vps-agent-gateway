package cloudnode

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuarantineMarkerSurvivesRestartAndIsOperatorCleared(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	if err := CheckQuarantine(state); err != nil {
		t.Fatal(err)
	}
	if err := MarkQuarantine(state); err != nil {
		t.Fatal(err)
	}
	if err := MarkQuarantine(state); err != nil {
		t.Fatalf("quarantine must be idempotent: %v", err)
	}
	if err := CheckQuarantine(state); err == nil || !strings.Contains(err.Error(), "operator review") {
		t.Fatalf("restart unexpectedly bypassed quarantine: %v", err)
	}
	marker := QuarantineMarkerPath(state)
	info, err := os.Lstat(marker)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("quarantine permissions=%#o, want 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "broker_cancellation_unconfirmed\n" {
		t.Fatalf("quarantine marker content=%q", string(data))
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := CheckQuarantine(state); err != nil {
		t.Fatalf("explicit operator clearing did not restore startup: %v", err)
	}
}

func TestQuarantineMarkerCannotFollowExistingSymlink(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	target := filepath.Join(dir, "unrelated-secret")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, QuarantineMarkerPath(state)); err != nil {
		t.Fatal(err)
	}
	if err := CheckQuarantine(state); err == nil {
		t.Fatal("quarantine symlink was unexpectedly treated as absent")
	}
	if err := MarkQuarantine(state); err != nil {
		t.Fatalf("existing marker should keep node quarantined: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("existing symlink target unexpectedly modified: %q", data)
	}
}

func TestQuarantineRejectsMissingOrInvalidStatePath(t *testing.T) {
	if err := CheckQuarantine(""); err == nil {
		t.Fatal("empty state path accepted")
	}
	if err := MarkQuarantine(""); err == nil {
		t.Fatal("empty state path accepted")
	}
	state := filepath.Join(t.TempDir(), "missing", "state.json")
	if err := MarkQuarantine(state); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing state parent directory failure, got %v", err)
	}
}
