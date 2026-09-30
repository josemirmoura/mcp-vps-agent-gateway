package gateway

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPublicToolCatalogHasDeliberateAnnotations(t *testing.T) {
	ts := httptest.NewServer(Handler(LocalExecutor{}, AuthConfig{Mode: "none", StaticSubject: "test-user"}))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "annotation-test", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) < 50 {
		t.Fatalf("unexpectedly small public tool catalog: %d", len(result.Tools))
	}

	seen := map[string]*mcp.Tool{}
	for _, tool := range result.Tools {
		if tool.Annotations == nil {
			t.Errorf("tool %q has no MCP annotations", tool.Name)
			continue
		}
		if tool.Annotations.DestructiveHint == nil {
			t.Errorf("tool %q has no deliberate destructiveHint", tool.Name)
		}
		if tool.Annotations.OpenWorldHint == nil {
			t.Errorf("tool %q has no deliberate openWorldHint", tool.Name)
		}
		seen[tool.Name] = tool
	}

	assertAnnotation := func(name string, readOnly, destructive, idempotent, openWorld bool) {
		t.Helper()
		tool := seen[name]
		if tool == nil {
			t.Fatalf("representative tool %q missing from public catalog", name)
		}
		a := tool.Annotations
		if a.ReadOnlyHint != readOnly ||
			a.DestructiveHint == nil || *a.DestructiveHint != destructive ||
			a.IdempotentHint != idempotent ||
			a.OpenWorldHint == nil || *a.OpenWorldHint != openWorld {
			t.Fatalf(
				"tool %q annotations = readOnly:%v destructive:%v idempotent:%v openWorld:%v; want %v/%v/%v/%v",
				name,
				a.ReadOnlyHint, boolValue(a.DestructiveHint), a.IdempotentHint, boolValue(a.OpenWorldHint),
				readOnly, destructive, idempotent, openWorld,
			)
		}
	}

	assertAnnotation("system.info", true, false, true, false)
	assertAnnotation("file.remove", false, true, false, false)
	assertAnnotation("network.check", true, false, true, true)
	assertAnnotation("shell.exec", false, true, false, true)
	assertAnnotation("compose.pull", false, true, false, true)
	assertAnnotation("permissions.request_root_access", false, false, true, false)
	assertAnnotation("permissions.discover_scope", true, false, true, false)
	assertAnnotation("permissions.request_sensitive_access", false, false, true, false)
}

func TestToolAnnotationClassifierRejectsUnknownTool(t *testing.T) {
	if got, ok := annotationsForTool("portico.future_unclassified_tool"); ok || got != nil {
		t.Fatalf("unknown tool was silently classified: %+v", got)
	}
}

func boolValue(v *bool) bool {
	return v != nil && *v
}
