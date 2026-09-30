package broker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

func sensitiveTestBroker(t *testing.T) (*Broker, string, *state.Store) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT", root)
	t.Setenv("VPS_AGENT_HOST_ROOT", "")
	cfg := &policy.Config{
		Version: 1,
		Mode: "scoped",
		Filesystem: policy.FilesystemPolicy{
			Read: []string{root}, Write: []string{root},
			Actions: []string{"list", "stat", "read", "write", "patch", "copy", "move", "remove", "hash"},
		},
		Shell: policy.ShellPolicy{Enabled: true, CWDRoots: []string{root}, MaxRuntimeSeconds: 60, MaxOutputBytes: 1 << 20},
		Network: policy.NetworkPolicy{Mode: "blocked"},
		Privilege: policy.PrivilegePolicy{Admin: "broker-only"},
		Replay: policy.ReplayPolicy{RequireIdempotencyForSafeWrites: true},
		Grant: policy.GrantPolicy{MaxTTLMinutes: 60, OutOfBandApproval: true, StepUpAuth: true},
	}
	if err := cfg.Validate(); err != nil { t.Fatal(err) }
	fs, err := securefs.New([]string{root}, []string{root}, securefs.DefaultMaxBytes)
	if err != nil { t.Fatal(err) }
	store, err := state.Open(":memory:")
	if err != nil { t.Fatal(err) }
	return &Broker{
		Policy: cfg, FS: fs, State: store,
		AdminToken: "operator-secret", ExpectedSubject: "alice",
	}, root, store
}

func TestSensitiveEnvRequiresSeparateTemporaryApproval(t *testing.T) {
	ctx := context.Background()
	b, root, store := sensitiveTestBroker(t)
	defer store.Close()

	envPath := filepath.Join(root, ".env")
	templatePath := filepath.Join(root, ".env.example")
	aliasPath := filepath.Join(root, "settings.txt")
	if err := os.WriteFile(envPath, []byte("private-value"), 0600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(templatePath, []byte("SAFE_TEMPLATE=1"), 0644); err != nil { t.Fatal(err) }
	if err := os.Link(envPath, aliasPath); err != nil { t.Fatal(err) }

	for _, path := range []string{envPath, aliasPath} {
		resp := b.Handle(ctx, wire.Request{ID: "read-denied-" + filepath.Base(path), Subject: "alice", Tool: "file.read", Resource: path})
		if resp.OK { t.Fatalf("protected path unexpectedly readable: %s", path) }
	}
	template := b.Handle(ctx, wire.Request{ID: "template", Subject: "alice", Tool: "file.read", Resource: templatePath})
	if !template.OK { t.Fatalf(".env.example should remain readable: %+v", template) }

	requestArgs, _ := json.Marshal(map[string]any{"path": envPath, "access": "read", "ttl_seconds": 60})
	request := b.Handle(ctx, wire.Request{
		ID: "secret-request", Subject: "alice", Tool: "permissions.request_sensitive_access",
		Resource: envPath, InvocationID: "secret-request-1", Args: requestArgs,
	})
	if !request.OK { t.Fatalf("sensitive request failed: %+v", request) }
	var pending struct {
		RequestID string `json:"request_id"`
		ApprovalToken string `json:"approval_token"`
	}
	if err := json.Unmarshal(request.Result, &pending); err != nil { t.Fatal(err) }
	if pending.RequestID == "" || pending.ApprovalToken == "" { t.Fatalf("invalid pending approval: %+v", pending) }

	confirmArgs, _ := json.Marshal(map[string]any{
		"request_id": pending.RequestID, "approval_token": pending.ApprovalToken, "decision": "approve",
	})
	approved := b.Handle(ctx, wire.Request{
		ID: "secret-confirm", Subject: "alice", Tool: "permissions.confirm_sensitive_access", Args: confirmArgs,
	})
	if !approved.OK { t.Fatalf("sensitive approval failed: %+v", approved) }

	allowed := b.Handle(ctx, wire.Request{ID: "read-approved", Subject: "alice", Tool: "file.read", Resource: envPath})
	if !allowed.OK { t.Fatalf("explicitly approved .env remained locked: %+v", allowed) }

	leakPath := filepath.Join(root, "leaked.txt")
	copyArgs, _ := json.Marshal(map[string]any{"destination": leakPath})
	copyLeak := b.Handle(ctx, wire.Request{
		ID: "copy-leak", Subject: "alice", Tool: "file.copy", Resource: envPath,
		InvocationID: "copy-leak-1", Args: copyArgs,
	})
	if copyLeak.OK {
		t.Fatal("protected .env was copied into an unprotected filename")
	}
	if _, err := os.Stat(leakPath); !os.IsNotExist(err) {
		t.Fatalf("secret downgrade created destination: %v", err)
	}

	aliasStillDenied := b.Handle(ctx, wire.Request{ID: "alias-still-denied", Subject: "alice", Tool: "file.read", Resource: aliasPath})
	if aliasStillDenied.OK { t.Fatal("exact-path approval must not silently authorize a hardlink alias") }

	revokeArgs, _ := json.Marshal(map[string]any{"path": envPath})
	revoked := b.Handle(ctx, wire.Request{
		ID: "secret-revoke", Subject: "alice", Tool: "permissions.revoke_sensitive_access",
		Resource: envPath, InvocationID: "secret-revoke-1", Args: revokeArgs,
	})
	if !revoked.OK { t.Fatalf("sensitive revoke failed: %+v", revoked) }
	lockedAgain := b.Handle(ctx, wire.Request{ID: "read-revoked", Subject: "alice", Tool: "file.read", Resource: envPath})
	if lockedAgain.OK { t.Fatal("revoked .env access remained readable") }
}

func TestDiscoverScopeShowsNamesWithoutOpeningContents(t *testing.T) {
	ctx := context.Background()
	b, root, store := sensitiveTestBroker(t)
	defer store.Close()
	for _, name := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0750); err != nil { t.Fatal(err) }
	}
	if err := os.WriteFile(filepath.Join(root, "top-secret.txt"), []byte("hidden"), 0600); err != nil { t.Fatal(err) }

	args, _ := json.Marshal(map[string]any{"limit": 20})
	resp := b.Handle(ctx, wire.Request{ID: "discover", Subject: "alice", Tool: "permissions.discover_scope", Args: args})
	if !resp.OK { t.Fatalf("discover failed: %+v", resp) }
	var out struct {
		PhysicalCeiling string `json:"physical_ceiling"`
		DiscoveryOnly bool `json:"discovery_only"`
		Directories []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"directories"`
	}
	if err := json.Unmarshal(resp.Result, &out); err != nil { t.Fatal(err) }
	if out.PhysicalCeiling != root || !out.DiscoveryOnly { t.Fatalf("unexpected discovery response: %+v", out) }
	if len(out.Directories) != 2 { t.Fatalf("expected directory names only, got %+v", out.Directories) }
	for _, d := range out.Directories {
		if d.Name == "top-secret.txt" { t.Fatal("discovery leaked a file entry") }
	}
}
