package gateway

import (
    "context"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Synthetic upstream test_* tools must never enter the normal Gateway catalog.
// The independent lab binary registers fixtures ONLY with an explicit build tag.
func TestProductionMCPServerDoesNotExposeConformanceFixtures(t *testing.T) {
    root := t.TempDir()
    manager, err := securefs.New([]string{root}, []string{root}, 1024)
    if err != nil {
        t.Fatal(err)
    }
    server := httptest.NewServer(Handler(LocalExecutor{FS: manager}, AuthConfig{
        Mode: "none", StaticSubject: "fixture-boundary-test",
    }))
    defer server.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    client := mcp.NewClient(&mcp.Implementation{Name: "fixture-boundary-test", Version: "1"}, nil)
    session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp"}, nil)
    if err != nil {
        t.Fatal(err)
    }
    defer session.Close()

    listed, err := session.ListTools(ctx, &mcp.ListToolsParams{})
    if err != nil {
        t.Fatal(err)
    }
    if len(listed.Tools) == 0 {
        t.Fatal("production Gateway unexpectedly advertised no tools")
    }
    for _, tool := range listed.Tools {
        if strings.HasPrefix(tool.Name, "test_") {
            t.Fatalf("synthetic conformance fixture %q leaked into production tools/list", tool.Name)
        }
    }
    prompts, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
    if err != nil {
        t.Fatal(err)
    }
    for _, prompt := range prompts.Prompts {
        if strings.HasPrefix(prompt.Name, "test_") {
            t.Fatalf("synthetic prompt %q leaked into production prompts/list", prompt.Name)
        }
    }
    resources, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
    if err != nil {
        t.Fatal(err)
    }
    for _, resource := range resources.Resources {
        if strings.HasPrefix(resource.URI, "test://") {
            t.Fatalf("synthetic resource %q leaked into production resources/list", resource.URI)
        }
    }
    templates, err := session.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{})
    if err != nil {
        t.Fatal(err)
    }
    for _, template := range templates.ResourceTemplates {
        if strings.HasPrefix(template.URITemplate, "test://") {
            t.Fatalf("synthetic template %q leaked into production resource templates", template.URITemplate)
        }
    }
}
