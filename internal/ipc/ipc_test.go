package ipc

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

type countingHandler struct{ calls atomic.Int64 }

func (h *countingHandler) Handle(_ context.Context, req wire.Request) wire.Response {
	h.calls.Add(1)
	return wire.Response{ID: req.ID, OK: true}
}

func TestIPCRejectsOversizedMalformedRequest(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "broker.sock")
	h := &countingHandler{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- NewServer(socket, h).Serve(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(socket); err == nil { break }
		if time.Now().After(deadline) { t.Fatal("socket did not appear") }
		time.Sleep(10 * time.Millisecond)
	}

	conn, err := net.Dial("unix", socket)
	if err != nil { t.Fatal(err) }
	payload := strings.Repeat("x", (2<<20)+4096)
	_, _ = conn.Write([]byte(payload))
	_ = conn.Close()
	time.Sleep(50 * time.Millisecond)
	if got := h.calls.Load(); got != 0 {
		t.Fatalf("oversized malformed request reached handler: %d", got)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second): t.Fatal("IPC server did not stop")
	}
}
