package cloudnode

import (
	"context"
	"encoding/json"
	"errors"
	"os"
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

// TestIntegrationActualCloudAndLocalBroker is invoked only by the private
// Cloud PostgreSQL job. It deliberately imports no private Cloud source:
// the CI runs the actual public connector against a real Cloud HTTP server.
func TestIntegrationActualCloudAndLocalBroker(t *testing.T) {
	cloudURL := strings.TrimSpace(os.Getenv("PORTICO_E2E_CLOUD_URL"))
	statePath := strings.TrimSpace(os.Getenv("PORTICO_E2E_NODE_STATE"))
	tasksPath := strings.TrimSpace(os.Getenv("PORTICO_E2E_TASK_IDS"))
	if cloudURL == "" && statePath == "" && tasksPath == "" {
		t.Skip("opt-in cross-repository Cloud e2e environment not configured")
	}
	if cloudURL == "" || statePath == "" || tasksPath == "" ||
		os.Getenv("PORTICO_E2E_ACK") != "YES_EPHEMERAL" {
		t.Fatal("incomplete or unauthorized cross-repository acceptance configuration")
	}
	if !strings.HasPrefix(cloudURL, "http://127.0.0.1:") {
		t.Fatal("cross-repository CI acceptance must use a loopback Cloud endpoint")
	}

	var ids struct {
		AllowedID string `json:"allowed_id"`
		DeniedID  string `json:"denied_id"`
	}
	raw, err := os.ReadFile(tasksPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ids); err != nil {
		t.Fatal(err)
	}
	if ids.AllowedID == "" || ids.DeniedID == "" || ids.AllowedID == ids.DeniedID {
		t.Fatal("Cloud task fixture lacks distinct allowed and denied task IDs")
	}

	enrolled, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(cloudURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	const subject = "ephemeral-local-broker"
	allowedRoot := t.TempDir()
	fs, err := securefs.New([]string{allowedRoot}, []string{allowedRoot}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	pol := &policy.Config{
		Version: 1,
		Mode:    "scoped",
		Filesystem: policy.FilesystemPolicy{
			Read:  []string{allowedRoot},
			Write: []string{allowedRoot},
		},
		Network:   policy.NetworkPolicy{Mode: "blocked"},
		Privilege: policy.PrivilegePolicy{Admin: "broker-only"},
	}
	if err := pol.Validate(); err != nil {
		t.Fatal(err)
	}
	localAudit, err := state.Open(filepath.Join(t.TempDir(), "local-audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer localAudit.Close()
	socketPath := filepath.Join(t.TempDir(), "broker.sock")
	localBroker := &broker.Broker{
		Policy:          pol,
		FS:              fs,
		State:           localAudit,
		ExpectedSubject: subject,
		InstanceID:      "ephemeral-node",
	}
	brokerCtx, stopBroker := context.WithCancel(t.Context())
	defer stopBroker()
	brokerErrors := make(chan error, 1)
	go func() { brokerErrors <- ipc.NewServer(socketPath, localBroker).Serve(brokerCtx) }()
	defer func() {
		stopBroker()
		select {
		case err := <-brokerErrors:
			if err != nil {
				t.Errorf("Broker socket shutdown: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("Broker socket leaked")
		}
	}()
	for attempt := 0; attempt < 100; attempt++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("Broker IPC did not start: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	runner := Runner{
		Client:            client,
		Identity:          enrolled.Identity,
		Broker:            ipc.Client{Socket: socketPath, Timeout: 2*time.Second},
		BrokerSubject:     subject,
		HeartbeatInterval: 200 * time.Millisecond,
		PollInterval:      100 * time.Millisecond,
		RenewInterval:     time.Second,
		BrokerTimeout:     4 * time.Second,
	}
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	decisions := map[string]string{}
	for len(decisions) < 2 {
		events, err := localAudit.ListAuditAfter(ctx, 0, 25)
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range events {
			if event.Event.ActionID != ids.AllowedID && event.Event.ActionID != ids.DeniedID {
				continue
			}
			if event.Event.Subject != subject {
				t.Fatalf("Cloud user obtained Broker principal authority: %+v", event.Event)
			}
			decisions[event.Event.ActionID] = event.Event.Decision
		}
		if len(decisions) == 2 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("Cloud lease -> Broker execution did not finish: %v", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	if decisions[ids.AllowedID] != "allow" || decisions[ids.DeniedID] != "deny" {
		t.Fatalf("local Broker policy did not enforce allow/deny: %+v", decisions)
	}
	if err := localAudit.VerifyAudit(ctx); err != nil {
		t.Fatalf("Broker append-only audit chain failed verification: %v", err)
	}
	// Give the connector time to acknowledge both signed completions to the
	// real Cloud HTTP API before cancelling its next idle lease poll.
	select {
	case <-time.After(time.Second):
	case <-ctx.Done():
		t.Fatal(errors.New("interoperability run canceled before Cloud completion"))
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Cloud connector failed to stop")
	}
	t.Log("real Cloud -> signed public connector -> local Broker -> signed Cloud completion: Broker allow/deny and local audit verified")
}
