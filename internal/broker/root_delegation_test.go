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

func TestDynamicRootApprovalActivatesAndRevokesWithoutPolicyReload(t *testing.T) {
	ctx := context.Background()
	physical := t.TempDir()
	baseline := filepath.Join(physical, "sandbox")
	project := filepath.Join(physical, "project-a")
	if err := os.MkdirAll(baseline, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT", physical)
	t.Setenv("VPS_AGENT_HOST_ROOT", "")

	cfg := &policy.Config{
		Version: 1,
		Mode:    "scoped",
		Filesystem: policy.FilesystemPolicy{
			Read:    []string{baseline},
			Write:   []string{baseline},
			Actions: []string{"list", "stat", "read", "mkdir", "write", "hash"},
		},
		Shell: policy.ShellPolicy{
			Enabled: true, CWDRoots: []string{baseline},
			MaxRuntimeSeconds: 60, MaxOutputBytes: 1 << 20,
		},
		Compose: policy.ResourcePolicy{
			Actions: []string{"validate", "up", "down", "pull"},
		},
		Network: policy.NetworkPolicy{Mode: "blocked"},
		Privilege: policy.PrivilegePolicy{Admin: "broker-only"},
		Replay: policy.ReplayPolicy{RequireIdempotencyForSafeWrites: true},
		Grant: policy.GrantPolicy{MaxTTLMinutes: 60, OutOfBandApproval: true, StepUpAuth: true},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	fs, err := securefs.New([]string{baseline}, []string{baseline}, securefs.DefaultMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	store, err := state.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	b := &Broker{
		Policy: cfg, FS: fs, State: store, AdminToken: "operator-secret",
		ExpectedSubject: "alice",
	}

	writeArgs, _ := json.Marshal(map[string]any{"content": "hello"})
	target := filepath.Join(project, "hello.txt")
	before := b.Handle(ctx, wire.Request{
		ID: "before", Subject: "alice", Tool: "file.write", Resource: target,
		InvocationID: "write-before", Args: writeArgs,
	})
	if before.OK {
		t.Fatal("undelegated project was writable before approval")
	}

	requestArgs, _ := json.Marshal(map[string]any{
		"root": project, "access": "work", "ttl_seconds": 0,
	})
	request := b.Handle(ctx, wire.Request{
		ID: "request", Subject: "alice", Tool: "permissions.request_root_access",
		Resource: project, InvocationID: "request-root-a", Args: requestArgs,
	})
	if !request.OK {
		t.Fatalf("root request failed: %+v", request)
	}
	var pending struct {
		RequestID     string `json:"request_id"`
		Status        string `json:"status"`
		ApprovalToken string `json:"approval_token"`
	}
	if err := json.Unmarshal(request.Result, &pending); err != nil {
		t.Fatal(err)
	}
	if pending.RequestID == "" || pending.Status != "pending" || pending.ApprovalToken == "" {
		t.Fatalf("unexpected pending request: %+v", pending)
	}

	badConfirmArgs, _ := json.Marshal(map[string]any{
		"request_id": pending.RequestID,
		"approval_token": "model-does-not-have-the-token",
		"decision": "approve",
	})
	badConfirm := b.Handle(ctx, wire.Request{
		ID: "bad-confirm", Subject: "alice", Tool: "permissions.confirm_root_access",
		Args: badConfirmArgs,
	})
	if badConfirm.OK {
		t.Fatal("root expansion succeeded without the hidden approval token")
	}

	otherSubjectArgs, _ := json.Marshal(map[string]any{
		"request_id": pending.RequestID,
		"approval_token": pending.ApprovalToken,
		"decision": "approve",
	})
	otherSubject := b.Handle(ctx, wire.Request{
		ID: "other-subject", Subject: "mallory", Tool: "permissions.confirm_root_access",
		Args: otherSubjectArgs,
	})
	if otherSubject.OK {
		t.Fatal("another authenticated subject approved alice's root request")
	}

	confirmArgs, _ := json.Marshal(map[string]any{
		"request_id": pending.RequestID,
		"approval_token": pending.ApprovalToken,
		"decision": "approve",
	})
	approved := b.Handle(ctx, wire.Request{
		ID: "approve", Subject: "alice", Tool: "permissions.confirm_root_access",
		Args: confirmArgs,
	})
	if !approved.OK {
		t.Fatalf("widget approval failed: %+v", approved)
	}

	after := b.Handle(ctx, wire.Request{
		ID: "after", Subject: "alice", Tool: "file.write", Resource: target,
		InvocationID: "write-after", Args: writeArgs,
	})
	if !after.OK {
		t.Fatalf("approved dynamic root was not writable: %+v", after)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "hello" {
		t.Fatalf("unexpected file after dynamic write: %q err=%v", data, err)
	}

	listed := b.Handle(ctx, wire.Request{
		ID: "list", Subject: "alice", Tool: "permissions.list_root_access",
	})
	if !listed.OK || !json.Valid(listed.Result) {
		t.Fatalf("root access listing failed: %+v", listed)
	}

	revokeArgs, _ := json.Marshal(map[string]any{"root": project})
	revoked := b.Handle(ctx, wire.Request{
		ID: "revoke", Subject: "alice", Tool: "permissions.revoke_root_access",
		Resource: project, InvocationID: "revoke-root-a", Args: revokeArgs,
	})
	if !revoked.OK {
		t.Fatalf("root revocation failed: %+v", revoked)
	}

	readAfterRevoke := b.Handle(ctx, wire.Request{
		ID: "read-after-revoke", Subject: "alice", Tool: "file.read", Resource: target,
	})
	if readAfterRevoke.OK {
		t.Fatal("revoked dynamic root remained readable")
	}
}

func TestDynamicRootRequestCannotDelegatePhysicalCeilingOrEscapeIt(t *testing.T) {
	ctx := context.Background()
	physical := t.TempDir()
	t.Setenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT", physical)
	t.Setenv("VPS_AGENT_HOST_ROOT", "")

	cfg := &policy.Config{
		Version: 1, Mode: "scoped",
		Filesystem: policy.FilesystemPolicy{Actions: []string{"read", "write"}},
		Network: policy.NetworkPolicy{Mode: "blocked"},
		Privilege: policy.PrivilegePolicy{Admin: "broker-only"},
		Replay: policy.ReplayPolicy{RequireIdempotencyForSafeWrites: true},
		Grant: policy.GrantPolicy{MaxTTLMinutes: 60},
	}
	store, err := state.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	b := &Broker{Policy: cfg, State: store, ExpectedSubject: "alice"}

	for i, root := range []string{physical, filepath.Dir(physical)} {
		args, _ := json.Marshal(map[string]any{"root": root, "access": "work", "ttl_seconds": 0})
		resp := b.Handle(ctx, wire.Request{
			ID: "deny", Subject: "alice", Tool: "permissions.request_root_access",
			Resource: root, InvocationID: "deny-root-" + string(rune('a'+i)), Args: args,
		})
		if resp.OK {
			t.Fatalf("unsafe root request unexpectedly succeeded for %s", root)
		}
	}
}
