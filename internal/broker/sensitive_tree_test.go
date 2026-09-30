package broker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

func TestProtectedFileBlocksRecursiveTreeRemoval(t *testing.T) {
	ctx := context.Background()
	b, root, store := sensitiveTestBroker(t)
	defer store.Close()

	project := filepath.Join(root, "tree")
	if err := os.MkdirAll(project, 0750); err != nil {
		t.Fatal(err)
	}
	protected := filepath.Join(project, ".env")
	if err := os.WriteFile(protected, []byte("placeholder"), 0600); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]any{"recursive": true})
	resp := b.Handle(ctx, wire.Request{
		ID: "tree-remove",
		Subject: "alice",
		Tool: "file.remove",
		Resource: project,
		InvocationID: "tree-remove-1",
		Args: args,
	})
	if resp.OK {
		t.Fatal("tree removal unexpectedly crossed the protected-file boundary")
	}
	if _, err := os.Stat(protected); err != nil {
		t.Fatalf("protected file changed after denied tree removal: %v", err)
	}
}

func TestProtectedFileBlocksDirectoryMove(t *testing.T) {
	ctx := context.Background()
	b, root, store := sensitiveTestBroker(t)
	defer store.Close()

	project := filepath.Join(root, "tree")
	if err := os.MkdirAll(project, 0750); err != nil {
		t.Fatal(err)
	}
	protected := filepath.Join(project, ".env")
	if err := os.WriteFile(protected, []byte("placeholder"), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "tree-moved")
	args, _ := json.Marshal(map[string]any{"destination": destination})
	resp := b.Handle(ctx, wire.Request{
		ID: "tree-move",
		Subject: "alice",
		Tool: "file.move",
		Resource: project,
		InvocationID: "tree-move-1",
		Args: args,
	})
	if resp.OK {
		t.Fatal("directory move unexpectedly crossed the protected-file boundary")
	}
	if _, err := os.Stat(protected); err != nil {
		t.Fatalf("protected file changed after denied directory move: %v", err)
	}
}
