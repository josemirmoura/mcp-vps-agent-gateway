package cloudnode

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

// This executor intentionally disregards cancellation, simulating an IPC
// operation whose remote Broker may still be performing host-side work.
type ignoringCancellationBroker struct {
	calls   atomic.Int64
	release chan struct{}
}

func (b *ignoringCancellationBroker) Call(_ context.Context, request wire.Request) (wire.Response, error) {
	b.calls.Add(1)
	<-b.release
	return wire.Response{ID: request.ID, OK: true}, nil
}

type cooperativeCancellationBroker struct {
	calls atomic.Int64
}

func (b *cooperativeCancellationBroker) Call(ctx context.Context, _ wire.Request) (wire.Response, error) {
	b.calls.Add(1)
	<-ctx.Done()
	return wire.Response{}, ctx.Err()
}

func TestRunnerStopsPollingIfBrokerIgnoresCancellation(t *testing.T) {
	identity := enrolledTestIdentity(t)
	var leases atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/nodes/" + identity.NodeID + "/heartbeat":
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/nodes/" + identity.NodeID + "/tasks/lease":
			if leases.Add(1) != 1 {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			now := time.Now().UTC()
			_ = json.NewEncoder(w).Encode(Task{
				ID:                "11111111-1111-1111-1111-111111111111",
				DestinationNodeID: identity.NodeID,
				Operation:         "system.info",
				LeaseID:           "22222222-2222-2222-2222-222222222222",
				LeaseExpiresAt:    timePtr(now.Add(time.Minute)),
				ExpiresAt:         timePtr(now.Add(2 * time.Minute)),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	broker := &ignoringCancellationBroker{release: make(chan struct{})}
	defer close(broker.release)
	runner := Runner{
		Client:             client,
		Identity:           identity,
		Broker:             broker,
		BrokerSubject:      "operator-local",
		HeartbeatInterval:  time.Hour,
		PollInterval:       time.Millisecond,
		RenewInterval:      time.Hour,
		BrokerTimeout:      120 * time.Millisecond,
		BrokerDrainTimeout: 50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	err = runner.Run(ctx)
	if !errors.Is(err, ErrBrokerStillRunning) {
		t.Fatalf("expected fail-closed connector shutdown, got %v", err)
	}
	if got := broker.calls.Load(); got != 1 {
		t.Fatalf("Broker call count=%d, expected exactly one", got)
	}
	if got := leases.Load(); got != 1 {
		t.Fatalf("connector polled for another Cloud task while Broker was active: leases=%d", got)
	}
}

func TestCanceledCooperativeBrokerNeverReportsCloudSuccess(t *testing.T) {
	identity := enrolledTestIdentity(t)
	var completions atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/nodes/"+identity.NodeID+"/tasks/"+
			"33333333-3333-3333-3333-333333333333/complete" {
			completions.Add(1)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	broker := &cooperativeCancellationBroker{}
	runner := Runner{
		Client:             client,
		Identity:           identity,
		Broker:             broker,
		BrokerSubject:      "operator-local",
		RenewInterval:      time.Hour,
		BrokerTimeout:      90 * time.Millisecond,
		BrokerDrainTimeout: 100 * time.Millisecond,
	}
	now := time.Now().UTC()
	err = runner.processTask(t.Context(), Task{
		ID:                "33333333-3333-3333-3333-333333333333",
		DestinationNodeID: identity.NodeID,
		Operation:         "system.info",
		LeaseID:           "44444444-4444-4444-4444-444444444444",
		LeaseExpiresAt:    timePtr(now.Add(time.Minute)),
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline cancellation, got %v", err)
	}
	if broker.calls.Load() != 1 {
		t.Fatalf("cooperative Broker calls=%d", broker.calls.Load())
	}
	if completions.Load() != 0 {
		t.Fatalf("Cloud completion was sent despite canceled Broker: %d", completions.Load())
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
