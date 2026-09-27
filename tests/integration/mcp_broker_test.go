package integration

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/gateway"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeServiceManager struct {
	mu       sync.Mutex
	restarts int
}

func (f *fakeServiceManager) Status(context.Context, string) (string, error) {
	return "active", nil
}

func (f *fakeServiceManager) List(context.Context) ([]broker.ServiceInfo, error) {
	return []broker.ServiceInfo{{Name: "vps-agent-test.service", Active: "active"}}, nil
}

func (f *fakeServiceManager) Logs(context.Context, string, int) (string, error) {
	return "fake log", nil
}

func (f *fakeServiceManager) Start(context.Context, string) (string, error) { return "active", nil }
func (f *fakeServiceManager) Stop(context.Context, string) (string, error) { return "inactive", nil }
func (f *fakeServiceManager) Reload(context.Context, string) (string, error) { return "active", nil }
func (f *fakeServiceManager) Enable(context.Context, string) (string, error) { return "enabled", nil }
func (f *fakeServiceManager) Disable(context.Context, string) (string, error) { return "disabled", nil }

func (f *fakeServiceManager) Restart(context.Context, string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restarts++
	return "active", nil
}

func (f *fakeServiceManager) RestartCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.restarts
}

func TestMCPToBrokerEndToEnd(t *testing.T) {
	root := t.TempDir()
	socket := filepath.Join(t.TempDir(), "broker.sock")

	store, err := state.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	fs, err := securefs.New([]string{root}, []string{root}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}

	services := &fakeServiceManager{}
	cfg := &policy.Config{
		Version: 1,
		Mode:    "scoped",
		Filesystem: policy.FilesystemPolicy{
			Read:  []string{root},
			Write: []string{root},
		},
		Services: policy.ResourcePolicy{
			Inspect: []string{"vps-agent-test.service"},
			Manage:  []string{"vps-agent-test.service"},
			Actions: []string{"status", "restart"},
		},
		Network:   policy.NetworkPolicy{Mode: "blocked"},
		Privilege: policy.PrivilegePolicy{Admin: "broker-only"},
		Replay:    policy.ReplayPolicy{RequireIdempotencyForSafeWrites: true},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	b := &broker.Broker{
		Policy:   cfg,
		FS:       fs,
		State:    store,
		Services: services,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- ipc.NewServer(socket, b).Serve(ctx)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := (ipc.Client{Socket: socket, Timeout: 100 * time.Millisecond}).Call(
			context.Background(),
			wireRequest("health-probe", "system.info", "", ""),
		); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	httpServer := httptest.NewServer(gateway.Handler(
		gateway.BrokerExecutor{Client: ipc.Client{Socket: socket, Timeout: 2 * time.Second}},
		gateway.AuthConfig{Mode: "none", StaticSubject: "alice"},
	))
	defer httpServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "integration-test", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "service.status",
		Arguments: map[string]any{"name": "vps-agent-test.service"},
	})
	if err != nil || result.IsError {
		t.Fatalf("status allowed call failed: err=%v result=%+v", err, result)
	}

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "service.restart",
		Arguments: map[string]any{"name": "vps-agent-test.service"},
	})
	if err != nil || result.IsError {
		t.Fatalf("restart allowed call failed: err=%v result=%+v", err, result)
	}
	if services.RestartCount() != 1 {
		t.Fatalf("restart count=%d, want 1", services.RestartCount())
	}

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "service.restart",
		Arguments: map[string]any{"name": "postgres.service"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("forbidden service restart was accepted")
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("IPC server shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("IPC server did not shut down")
	}
}

// Avoid importing internal/wire into the test's public intent; keep probe minimal.
func wireRequest(id, tool, resource, subject string) wire.Request {
	return wire.Request{ID: id, Tool: tool, Resource: resource, Subject: subject}
}
