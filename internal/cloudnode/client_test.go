package cloudnode

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

func enrolledTestIdentity(t *testing.T) Identity {
	t.Helper()
	identity, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	identity.NodeID = "11111111-1111-1111-1111-111111111111"
	identity.WorkspaceID = "22222222-2222-2222-2222-222222222222"
	identity.Fingerprint = "fingerprint"
	return identity
}

func verifySignedHTTP(t *testing.T, r *http.Request, body []byte, publicKey string) {
	t.Helper()
	timestamp, err := strconv.ParseInt(r.Header.Get("X-Portico-Timestamp"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	nonce := r.Header.Get("X-Portico-Nonce")
	if len(nonce) < 16 {
		t.Fatalf("nonce=%q", nonce)
	}
	signature, err := base64.RawURLEncoding.DecodeString(r.Header.Get("X-Portico-Signature"))
	if err != nil {
		t.Fatal(err)
	}
	rawPublic, err := base64.RawURLEncoding.DecodeString(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(ed25519.PublicKey(rawPublic), canonicalMessage(
		r.Method, r.URL.Path, timestamp, nonce, body,
	), signature) {
		t.Fatal("invalid request signature")
	}
}

func TestSignedCloudNodeAPI(t *testing.T) {
	identity := enrolledTestIdentity(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		verifySignedHTTP(t, r, body, identity.PublicKey)

		switch r.URL.Path {
		case "/api/v1/nodes/" + identity.NodeID + "/heartbeat":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))

		case "/api/v1/nodes/" + identity.NodeID + "/tasks/lease":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(Task{
				ID:                "33333333-3333-3333-3333-333333333333",
				WorkspaceID:       identity.WorkspaceID,
				DestinationNodeID: identity.NodeID,
				RequestedBy:       "human-user",
				Operation:         "system.info",
				State:             "leased",
				LeaseID:           "44444444-4444-4444-4444-444444444444",
			})

		case "/api/v1/nodes/" + identity.NodeID + "/tasks/33333333-3333-3333-3333-333333333333/renew":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"lease_expires_at":"2030-01-01T00:00:00Z"}`))

		case "/api/v1/nodes/" + identity.NodeID + "/tasks/33333333-3333-3333-3333-333333333333/complete":
			w.WriteHeader(http.StatusNoContent)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	if err := client.Heartbeat(ctx, identity); err != nil {
		t.Fatal(err)
	}
	task, ok, err := client.LeaseTask(ctx, identity)
	if err != nil || !ok || task.Operation != "system.info" {
		t.Fatalf("task=%+v ok=%v err=%v", task, ok, err)
	}
	expiresAt, err := client.RenewTaskLease(ctx, identity, task.ID, task.LeaseID)
	if err != nil {
		t.Fatal(err)
	}
	if !expiresAt.Equal(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expiresAt=%s", expiresAt)
	}
	if err := client.CompleteTask(ctx, identity, task.ID, Completion{
		LeaseID: task.LeaseID,
		Result:  json.RawMessage(`{"ok":true}`),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureEnrollmentRecoversUsingPersistedPendingKey(t *testing.T) {
	var mu sync.Mutex
	attempts := 0
	firstPublicKey := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/nodes/enroll" {
			http.NotFound(w, r)
			return
		}

		var request EnrollmentRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}

		mu.Lock()
		defer mu.Unlock()
		attempts++
		if attempts == 1 {
			firstPublicKey = request.PublicKey
			http.Error(w, "temporary failure", http.StatusServiceUnavailable)
			return
		}
		if request.PublicKey != firstPublicKey {
			t.Fatal("enrollment retry changed node public key")
		}
		if request.Token != "bootstrap-secret" {
			t.Fatalf("token=%q", request.Token)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(EnrollmentResponse{
			NodeID:      "node-1",
			WorkspaceID: "workspace-1",
			Fingerprint: "fingerprint-1",
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(t.TempDir(), "state.json")

	if _, err := EnsureEnrollment(
		t.Context(), client, statePath, "bootstrap-secret", "node-a",
	); !IsStatus(err, http.StatusServiceUnavailable) {
		t.Fatalf("first enrollment error=%v", err)
	}

	pending, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Enrollment == nil || pending.Enrollment.Token != "bootstrap-secret" {
		t.Fatalf("pending=%+v", pending)
	}

	identity, err := EnsureEnrollment(t.Context(), client, statePath, "", "ignored")
	if err != nil {
		t.Fatal(err)
	}
	if identity.NodeID != "node-1" || identity.PublicKey != firstPublicKey {
		t.Fatalf("identity=%+v", identity)
	}

	finalState, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if finalState.Enrollment != nil {
		t.Fatal("bootstrap token retained after successful enrollment")
	}
}

func TestCloudURLRequiresHTTPSOutsideLoopback(t *testing.T) {
	if _, err := NewClient("http://example.com", nil); err == nil {
		t.Fatal("expected non-loopback HTTP to be rejected")
	}
	if _, err := NewClient("https://example.com", nil); err != nil {
		t.Fatal(err)
	}
}
