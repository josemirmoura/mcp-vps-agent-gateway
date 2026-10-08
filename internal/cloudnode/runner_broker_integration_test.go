package cloudnode

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

type capturedCloudCompletion struct {
	taskID string
	value  Completion
}

func TestCloudConnectorRoutesToRealLocalBrokerWithDenyAndAudit(t *testing.T) {
	const localPrincipal = "local-broker-principal"
	const untrustedCloudRequester = "remote-cloud-user"
	const allowedTaskID = "33333333-3333-4333-8333-333333333333"
	const deniedTaskID = "44444444-4444-4444-8444-444444444444"

	root := t.TempDir()
	allowedPath := filepath.Join(root, "permitted.txt")
	if err := os.WriteFile(allowedPath, []byte("local-authorized-only"), 0o600); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(t.TempDir(), "broker.sock")
	store, err := state.Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	fs, err := securefs.New([]string{root}, []string{root}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	configuration := &policy.Config{
		Version: 1,
		Mode:    "scoped",
		Filesystem: policy.FilesystemPolicy{
			Read:  []string{root},
			Write: []string{root},
		},
		Network:   policy.NetworkPolicy{Mode: "blocked"},
		Privilege: policy.PrivilegePolicy{Admin: "broker-only"},
		Replay:    policy.ReplayPolicy{RequireIdempotencyForSafeWrites: true},
	}
	if err := configuration.Validate(); err != nil {
		t.Fatal(err)
	}
	realBroker := &broker.Broker{
		Policy:          configuration,
		FS:              fs,
		State:           store,
		ExpectedSubject: localPrincipal,
		InstanceID:      "synthetic-broker",
	}
	brokerCtx, stopBroker := context.WithCancel(t.Context())
	defer stopBroker()
	brokerErr := make(chan error, 1)
	go func() {
		brokerErr <- ipc.NewServer(socket, realBroker).Serve(brokerCtx)
	}()
	defer func() {
		stopBroker()
		select {
		case err := <-brokerErr:
			if err != nil {
				t.Errorf("Broker shutdown: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("Broker socket did not stop")
		}
	}()

	for attempt := 0; attempt < 100; attempt++ {
		if _, err := os.Stat(socket); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(socket); err != nil {
		t.Fatalf("Broker socket never appeared: %v", err)
	}

	identity := enrolledTestIdentity(t)
	var mu sync.Mutex
	leaseIndex := 0
	completionCh := make(chan capturedCloudCompletion, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(io.LimitReader(request.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		verifySignedHTTP(t, request, body, identity.PublicKey)

		prefix := "/api/v1/nodes/" + identity.NodeID
		switch {
		case request.URL.Path == prefix+"/heartbeat":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case request.URL.Path == prefix+"/tasks/lease":
			mu.Lock()
			index := leaseIndex
			leaseIndex++
			mu.Unlock()
			if index >= 2 {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			task := Task{
				ID:                allowedTaskID,
				WorkspaceID:       identity.WorkspaceID,
				DestinationNodeID: identity.NodeID,
				RequestedBy:       untrustedCloudRequester,
				Operation:         "file.read",
				Resource:          allowedPath,
				State:             "leased",
				LeaseID:           "55555555-5555-4555-8555-555555555555",
			}
			if index == 1 {
				task.ID = deniedTaskID
				task.Resource = "/etc/shadow"
				task.LeaseID = "66666666-6666-4666-8666-666666666666"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(task)
		case strings.HasSuffix(request.URL.Path, "/complete"):
			var completion Completion
			if err := json.Unmarshal(body, &completion); err != nil {
				http.Error(w, "invalid completion", http.StatusBadRequest)
				return
			}
			taskID := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, prefix+"/tasks/"), "/complete")
			select {
			case completionCh <- capturedCloudCompletion{taskID: taskID, value: completion}:
			default:
				http.Error(w, "duplicate completion", http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	runner := Runner{
		Client:            client,
		Identity:          identity,
		Broker:            ipc.Client{Socket: socket, Timeout: time.Second},
		BrokerSubject:     localPrincipal,
		HeartbeatInterval: time.Hour,
		PollInterval:      10 * time.Millisecond,
		RenewInterval:     time.Hour,
		BrokerTimeout:     time.Second,
	}
	runnerDone := make(chan error, 1)
	go func() { runnerDone <- runner.Run(ctx) }()

	completed := make(map[string]Completion, 2)
	for len(completed) != 2 {
		select {
		case event := <-completionCh:
			if _, exists := completed[event.taskID]; exists {
				t.Fatalf("duplicate completion for task %s", event.taskID)
			}
			completed[event.taskID] = event.value
		case <-ctx.Done():
			t.Fatalf("connector failed to complete two tasks: %v", ctx.Err())
		}
	}
	cancel()
	select {
	case err := <-runnerDone:
		if err != nil {
			t.Fatalf("connector shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("connector did not stop after cancellation")
	}

	allowed, ok := completed[allowedTaskID]
	if !ok {
		t.Fatal("allowed task never completed")
	}
	var allowedReply wire.Response
	if err := json.Unmarshal(allowed.Result, &allowedReply); err != nil {
		t.Fatal(err)
	}
	if allowed.Error != "" || !allowedReply.OK ||
		!strings.Contains(string(allowedReply.Result), "local-authorized-only") {
		t.Fatalf("locally allowed Broker response invalid: %+v, %+v", allowed, allowedReply)
	}

	denied, ok := completed[deniedTaskID]
	if !ok {
		t.Fatal("denied task never completed")
	}
	var deniedReply wire.Response
	if err := json.Unmarshal(denied.Result, &deniedReply); err != nil {
		t.Fatal(err)
	}
	if denied.Error != "local Broker denied operation" {
		t.Fatalf("Broker denial was marked as Cloud success: %+v", denied)
	}
	if deniedReply.OK || deniedReply.Error == nil ||
		deniedReply.Error.Code != "permission_denied" {
		t.Fatalf("Cloud request bypassed local Broker denial: %+v", deniedReply)
	}

	if err := store.VerifyAudit(t.Context()); err != nil {
		t.Fatalf("Broker local audit chain is invalid: %v", err)
	}
	events, err := store.ListAuditAfter(t.Context(), 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	decisions := make(map[string]string)
	for _, record := range events {
		if record.Event.ActionID == allowedTaskID || record.Event.ActionID == deniedTaskID {
			if record.Event.Subject != localPrincipal {
				t.Fatalf("untrusted Cloud requester became local authority: %+v", record.Event)
			}
			decisions[record.Event.ActionID] = record.Event.Decision
		}
	}
	if decisions[allowedTaskID] != "allow" || decisions[deniedTaskID] != "deny" {
		t.Fatalf("Broker audit did not record both authorization decisions: %+v", decisions)
	}
}
