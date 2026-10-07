package cloudnode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPendingAndEnrolledStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	identity, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	pending := NodeState{
		Identity: identity,
		Enrollment: &PendingEnrollment{
			Token:    "bootstrap-secret",
			NodeName: "node-a",
			Platform: "linux",
		},
	}
	if err := SaveState(path, pending); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("state mode=%#o", got)
	}

	loaded, err := LoadState(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Enrollment == nil || loaded.Enrollment.Token != "bootstrap-secret" {
		t.Fatalf("loaded pending state=%+v", loaded)
	}

	loaded.Identity.NodeID = "node-1"
	loaded.Identity.WorkspaceID = "workspace-1"
	loaded.Identity.Fingerprint = "fingerprint-1"
	loaded.Enrollment = nil
	if err := SaveState(path, loaded); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "bootstrap-secret") {
		t.Fatal("consumed bootstrap token remained in enrolled state")
	}

	finalState, err := LoadState(path)
	if err != nil {
		t.Fatal(err)
	}
	if finalState.Enrollment != nil || finalState.Identity.NodeID != "node-1" {
		t.Fatalf("final state=%+v", finalState)
	}
}

func TestIdentityRejectsBroadPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	identity, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	state := NodeState{
		Identity: Identity{
			NodeID:      "node-1",
			WorkspaceID: "workspace-1",
			PrivateKey:  identity.PrivateKey,
			PublicKey:   identity.PublicKey,
			Fingerprint: "fingerprint-1",
		},
	}
	if err := SaveState(path, state); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadState(path); err == nil {
		t.Fatal("expected broad identity permissions to be rejected")
	}
}
