package cloudnode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

type captureBroker struct {
	requests chan wire.Request
}

func (b captureBroker) Call(_ context.Context, request wire.Request) (wire.Response, error) {
	b.requests <- request
	return wire.Response{
		ID:     request.ID,
		OK:     true,
		Result: json.RawMessage(`{"executed":true}`),
	}, nil
}

func TestRunnerUsesLocalBrokerSubjectAndTaskInvocationID(t *testing.T) {
	identity := enrolledTestIdentity(t)
	taskID := "33333333-3333-3333-3333-333333333333"
	leaseID := "44444444-4444-4444-4444-444444444444"

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var mu sync.Mutex
	leased := false
	completionCh := make(chan Completion, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/nodes/" + identity.NodeID + "/heartbeat":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))

		case "/api/v1/nodes/" + identity.NodeID + "/tasks/lease":
			mu.Lock()
			alreadyLeased := leased
			if !leased {
				leased = true
			}
			mu.Unlock()

			if alreadyLeased {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(Task{
				ID:                taskID,
				WorkspaceID:       identity.WorkspaceID,
				DestinationNodeID: identity.NodeID,
				RequestedBy:       "cloud-user-from-origin",
				Operation:         "file.read",
				Resource:          "/opt/project/README.md",
				Action:            "read",
				GrantID:           "grant-1",
				Input:             json.RawMessage(`{"encoding":"utf-8"}`),
				State:             "leased",
				LeaseID:           leaseID,
			})

		case "/api/v1/nodes/" + identity.NodeID + "/tasks/" + taskID + "/complete":
			var completion Completion
			if err := json.NewDecoder(r.Body).Decode(&completion); err != nil {
				t.Fatal(err)
			}
			completionCh <- completion
			w.WriteHeader(http.StatusNoContent)
			cancel()

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	broker := captureBroker{requests: make(chan wire.Request, 1)}
	runner := Runner{
		Client:            client,
		Identity:          identity,
		Broker:            broker,
		BrokerSubject:     "operator-local",
		HeartbeatInterval: time.Hour,
		PollInterval:      10 * time.Millisecond,
		RenewInterval:     time.Hour,
		BrokerTimeout:     time.Second,
	}

	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	select {
	case request := <-broker.requests:
		if request.Subject != "operator-local" {
			t.Fatalf("Broker subject=%q", request.Subject)
		}
		if request.Subject == "cloud-user-from-origin" {
			t.Fatal("Cloud requester was incorrectly trusted as local Broker authority")
		}
		if request.InvocationID != taskID || request.ID != taskID {
			t.Fatalf("request IDs=%+v", request)
		}
		if request.Tool != "file.read" ||
			request.Resource != "/opt/project/README.md" ||
			request.Action != "read" ||
			request.GrantID != "grant-1" {
			t.Fatalf("Broker envelope=%+v", request)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Broker request was not executed")
	}

	select {
	case completion := <-completionCh:
		if completion.LeaseID != leaseID || len(completion.Result) == 0 || completion.Error != "" {
			t.Fatalf("completion=%+v", completion)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Cloud completion was not sent")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runner did not stop after context cancellation")
	}
}

func TestTaskDeliveryWindowRejectsBothExpiredDeadlines(t *testing.T) {
	now := time.Date(2026, time.October, 8, 11, 30, 0, 0, time.UTC)
	past := now.Add(-time.Second)
	future := now.Add(time.Minute)
	for _, tc := range []struct {
		name          string
		leaseDeadline *time.Time
		taskDeadline  *time.Time
		wantError     bool
	}{
		{name: "valid both", leaseDeadline: &future, taskDeadline: &future},
		{name: "expired lease", leaseDeadline: &past, taskDeadline: &future, wantError: true},
		{name: "expired task", leaseDeadline: &future, taskDeadline: &past, wantError: true},
		{name: "lease expires exactly now", leaseDeadline: &now, wantError: true},
		{name: "task expires exactly now", taskDeadline: &now, wantError: true},
		{name: "legacy no deadlines"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTaskDeliveryWindow(Task{
				LeaseExpiresAt: tc.leaseDeadline,
				ExpiresAt:      tc.taskDeadline,
			}, now)
			if (err != nil) != tc.wantError {
				t.Fatalf("delivery check err=%v wantError=%t", err, tc.wantError)
			}
		})
	}
}

func TestRunnerDoesNotDispatchExpiredCloudTaskToBroker(t *testing.T) {
	expired := time.Now().Add(-time.Second)
	broker := captureBroker{requests: make(chan wire.Request, 1)}
	runner := Runner{
		Identity: Identity{NodeID: "node-ci"},
		Broker: broker,
		BrokerSubject: "operator-local",
	}
	err := runner.processTask(t.Context(), Task{
		ID: "task-ci", DestinationNodeID: "node-ci", LeaseID: "lease-ci",
		Operation: "system.info", LeaseExpiresAt: &expired,
	})
	if err == nil {
		t.Fatal("expired task did not fail closed")
	}
	select {
	case request := <-broker.requests:
		t.Fatalf("expired task reached local Broker: %+v", request)
	default:
	}
}

func TestCloudRenewalWatchdogExtendsInitialLease(t *testing.T) {
	identity := enrolledTestIdentity(t)
	taskID := "55555555-5555-5555-5555-555555555555"
	leaseID := "66666666-6666-6666-6666-666666666666"
	var renewals atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/nodes/"+identity.NodeID+"/tasks/"+taskID+"/renew" {
			http.NotFound(w, r)
			return
		}
		renewals.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"lease_expires_at": time.Now().UTC().Add(400 * time.Millisecond).Format(time.RFC3339Nano),
		})
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	initial := time.Now().Add(250 * time.Millisecond)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := Runner{Client: client, Identity: identity, RenewInterval: 40 * time.Millisecond}
	fatal := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		runner.renewLoop(ctx, Task{ID: taskID, LeaseID: leaseID, LeaseExpiresAt: &initial}, fatal)
		close(done)
	}()

	select {
	case err := <-fatal:
		t.Fatalf("valid signed renewal unexpectedly failed: %v", err)
	case <-time.After(550 * time.Millisecond):
	}
	if renewals.Load() < 1 {
		t.Fatal("renewal requests never reached Cloud")
	}
	select {
	case err := <-fatal:
		t.Fatalf("initial lease expiry incorrectly canceled renewed execution: %v", err)
	default:
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("renewal loop did not exit after cancellation")
	}
}

func TestCloudRenewalWatchdogStopsWhenCloudIsUnavailable(t *testing.T) {
	identity := enrolledTestIdentity(t)
	taskID := "77777777-7777-7777-7777-777777777777"
	leaseID := "88888888-8888-8888-8888-888888888888"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	initial := time.Now().Add(220 * time.Millisecond)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	runner := Runner{Client: client, Identity: identity, RenewInterval: 40 * time.Millisecond}
	fatal := make(chan error, 1)
	go runner.renewLoop(ctx, Task{ID: taskID, LeaseID: leaseID, LeaseExpiresAt: &initial}, fatal)
	select {
	case err := <-fatal:
		if err == nil {
			t.Fatal("expired lease watchdog returned no error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("unrenewed lease did not expire locally")
	}
}
