package ipc

import (
	"context"
	"encoding/json"
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


func waitSocket(t *testing.T, socket string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(socket); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("socket did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestIPCPeerCredentialAllowAndDeny(t *testing.T) {
	t.Run("allow current uid", func(t *testing.T) {
		socket := filepath.Join(t.TempDir(), "broker.sock")
		h := &countingHandler{}
		srv := NewServer(socket, h)
		srv.AllowPeerUIDs(uint32(os.Getuid()))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- srv.Serve(ctx) }()
		waitSocket(t, socket)

		conn, err := net.Dial("unix", socket)
		if err != nil { t.Fatal(err) }
		if err := json.NewEncoder(conn).Encode(wire.Request{ID: "ok", Tool: "system.info"}); err != nil { t.Fatal(err) }
		var resp wire.Response
		if err := json.NewDecoder(conn).Decode(&resp); err != nil { t.Fatal(err) }
		_ = conn.Close()
		if !resp.OK || h.calls.Load() != 1 { t.Fatalf("resp=%+v calls=%d", resp, h.calls.Load()) }
		cancel()
		select { case <-done: case <-time.After(2*time.Second): t.Fatal("server did not stop") }
	})

	t.Run("deny unexpected uid", func(t *testing.T) {
		socket := filepath.Join(t.TempDir(), "broker.sock")
		h := &countingHandler{}
		srv := NewServer(socket, h)
		deniedUID := uint32(os.Getuid()) + 1
		if deniedUID == uint32(os.Getuid()) { deniedUID++ }
		srv.AllowPeerUIDs(deniedUID)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- srv.Serve(ctx) }()
		waitSocket(t, socket)

		conn, err := net.Dial("unix", socket)
		if err != nil { t.Fatal(err) }
		_ = json.NewEncoder(conn).Encode(wire.Request{ID: "deny", Tool: "system.info"})
		var resp wire.Response
		if err := json.NewDecoder(conn).Decode(&resp); err != nil { t.Fatal(err) }
		_ = conn.Close()
		if resp.OK || resp.Error == nil || resp.Error.Code != "unauthorized_peer" {
			t.Fatalf("expected unauthorized_peer, got %+v", resp)
		}
		if h.calls.Load() != 0 { t.Fatalf("unauthorized peer reached handler: %d", h.calls.Load()) }
		cancel()
		select { case <-done: case <-time.After(2*time.Second): t.Fatal("server did not stop") }
	})
}

