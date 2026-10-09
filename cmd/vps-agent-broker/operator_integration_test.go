package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
)

// This exercises the real BFF over HTTP and the real restricted Unix socket,
// policy, SQLite grant transaction and audit. It never uses Docker/production.
func TestOperatorPortalRealBrokerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp("", "portico-approval-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	physical := filepath.Join(dir, "physical")
	baseline := filepath.Join(physical, "baseline")
	for _, name := range []string{"baseline", "read", "denied", "work"} {
		if err := os.MkdirAll(filepath.Join(physical, name), 0750); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT", physical)
	t.Setenv("VPS_AGENT_HOST_ROOT", "")
	t.Setenv("VPS_AGENT_OPERATOR_ID", "fixture-owner")
	cfg := &policy.Config{Version: 1, Mode: "scoped", Filesystem: policy.FilesystemPolicy{Read: []string{baseline}, Write: []string{baseline}, Actions: []string{"list", "stat", "read", "write"}}, Network: policy.NetworkPolicy{Mode: "blocked"}, Privilege: policy.PrivilegePolicy{Admin: "broker-only"}, Grant: policy.GrantPolicy{MaxTTLMinutes: 60, OutOfBandApproval: true, StepUpAuth: true}}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	fs, err := securefs.New([]string{baseline}, []string{baseline}, securefs.DefaultMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	store, err := state.Open(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	b := &broker.Broker{State: store, Policy: cfg, FS: fs, AdminToken: "fixture-admin-only", ExpectedSubject: "fixture-agent", InstanceID: "fixture-node"}
	ids := map[string]string{}
	for _, name := range []string{"read", "denied", "work"} {
		access := "read"
		if name == "work" {
			access = "work"
		}
		a, err := store.CreateRootApproval(ctx, "fixture-agent", filepath.Join(physical, name), access, 5*time.Minute, 5*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		ids[name] = a.ID
	}
	path := filepath.Join(baseline, ".env")
	if err := os.WriteFile(path, []byte("synthetic-fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := store.CreateApproval(ctx, "fixture-agent", []string{"sensitive.read:" + hex.EncodeToString([]byte(path))}, 5*time.Minute, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ids["sensitive"] = a.ID
	socket := filepath.Join(dir, "operator.sock")
	token := strings.Repeat("f", 32)
	srv := ipc.NewServer(socket, &operatorBridge{broker: b, token: token})
	srv.AllowPeerUIDs(uint32(os.Getuid()))
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx) }()
	defer func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("operator IPC did not stop")
		}
	}()
	for deadline := time.Now().Add(3 * time.Second); ; {
		if _, err := os.Stat(socket); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("operator socket unavailable")
		}
		time.Sleep(10 * time.Millisecond)
	}
	raw, _ := json.Marshal(ids)
	command := exec.CommandContext(ctx, "python3", "../../tests/integration/operator_real_broker.py")
	command.Env = append(os.Environ(), "PORTICO_OPERATOR_SOCKET="+socket, "PORTICO_OPERATOR_APPROVAL_TOKEN="+token, "PORTICO_OPERATOR_ID=fixture-owner", "PORTICO_OPERATOR_NODE_ID=fixture-node", "PORTICO_OPERATOR_PUBLIC_ORIGIN=https://operator.fixture.test", "PORTICO_OPERATOR_PHYSICAL_CEILING="+physical, "PORTICO_FIXTURE_IDS="+string(raw))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("real portal/IPC integration failed: %v\n%s", err, output)
	}
	t.Log(string(output))
	for name, id := range ids {
		a, err := store.GetApproval(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		want := "approved"
		if name == "denied" {
			want = "denied"
		}
		if a.Status != want {
			t.Fatalf("%s: %s", name, a.Status)
		}
	}
	roots, err := store.ListActiveRootDelegations(ctx, "fixture-agent")
	if err != nil || len(roots) != 2 {
		t.Fatalf("roots=%v err=%v", roots, err)
	}
	grants, err := store.ListActiveGrants(ctx, "fixture-agent")
	if err != nil || len(grants) != 1 {
		t.Fatalf("grants=%v err=%v", grants, err)
	}
	if err := store.VerifyAudit(ctx); err != nil {
		t.Fatal(err)
	}
	audit, err := store.ListAuditAfter(ctx, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	decisions := 0
	for _, row := range audit {
		if row.Event.ApprovalID != "" {
			decisions++
			if row.Event.Subject != "fixture-owner" || row.Event.Requester != "fixture-agent" || row.Event.InstanceID != "fixture-node" {
				t.Fatal("decision lost identity provenance")
			}
		}
	}
	if decisions != 4 {
		t.Fatalf("atomic decisions=%d", decisions)
	}
}
