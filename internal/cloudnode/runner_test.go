package cloudnode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
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
