//go:build portico_conformance_fixture

// This process is ONLY built with -tags=portico_conformance_fixture.
// It runs a real Portico MCP server registration against synthetic upstream
// conformance probes. It is never linked into the production Gateway binary.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/gateway"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const loopbackAddress = "127.0.0.1:18996"

func fakePNG() []byte {
	const tiny = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRV4AAAAASUVORK5CYII="
	decoded, err := base64.StdEncoding.DecodeString(tiny)
	if err != nil {
		panic(err)
	}
	return decoded
}

func textResult(value string) []mcp.Content {
	return []mcp.Content{&mcp.TextContent{Text: value}}
}

func syntheticTool(server *mcp.Server, name string, handler func() *mcp.CallToolResult) {
	mcp.AddTool(server, &mcp.Tool{
		Name: name, Description: "Disposable MCP conformance diagnostic; NEVER available in production.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
		return handler(), struct{}{}, nil
	})
}

func fixtureServer() (*mcp.Server, error) {
	root, err := os.MkdirTemp("", "portico-mcp-conformance-fixture-")
	if err != nil {
		return nil, err
	}
	fs, err := securefs.New([]string{root}, []string{root}, securefs.DefaultMaxBytes)
	if err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}
	// Crucially this calls the production Gateway MCP registration function.
	// We add only the upstream-requested synthetic tools to THIS instance.
	server := gateway.NewMCPServer(gateway.LocalExecutor{FS: fs})
	image := fakePNG()

	syntheticTool(server, "test_simple_text", func() *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: textResult("This is a simple text response for testing.")}
	})
	syntheticTool(server, "test_image_content", func() *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.ImageContent{Data: image, MIMEType: "image/png"}}}
	})
	syntheticTool(server, "test_audio_content", func() *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.AudioContent{Data: []byte("RIFF"), MIMEType: "audio/wav"}}}
	})
	embedded := &mcp.EmbeddedResource{Resource: &mcp.ResourceContents{
		URI: "test://embedded-resource", MIMEType: "text/plain", Text: "This is an embedded resource content.",
	}}
	syntheticTool(server, "test_embedded_resource", func() *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: []mcp.Content{embedded}}
	})
	syntheticTool(server, "test_multiple_content_types", func() *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: []mcp.Content{
			&mcp.TextContent{Text: "Multiple content types test:"},
			&mcp.ImageContent{Data: image, MIMEType: "image/png"},
			&mcp.EmbeddedResource{Resource: &mcp.ResourceContents{
				URI: "test://mixed-content-resource", MIMEType: "application/json",
				Text: `{"test":"data","value":123}`,
			}},
		}}
	})
	syntheticTool(server, "test_error_handling", func() *mcp.CallToolResult {
		return &mcp.CallToolResult{
			IsError: true,
			Content: textResult("This tool intentionally returns an error for testing"),
		}
	})
	syntheticTool(server, "test_tool_with_progress", func() *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: textResult("Progress diagnostic executed")}
	})

	textPrompt := func(name, text string) {
		server.AddPrompt(&mcp.Prompt{Name: name},
			func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
				return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{
					Role: "user", Content: &mcp.TextContent{Text: text},
				}}}, nil
			})
	}
	textPrompt("test_simple_prompt", "This is a simple prompt for testing.")
	server.AddPrompt(&mcp.Prompt{
		Name: "test_prompt_with_arguments",
		Arguments: []*mcp.PromptArgument{{Name: "arg1", Required: true}, {Name: "arg2", Required: true}},
	}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := req.Params.Arguments
		return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{
			Role: "user", Content: &mcp.TextContent{
				Text: fmt.Sprintf("Prompt with arguments: arg1='%s', arg2='%s'", args["arg1"], args["arg2"]),
			},
		}}}, nil
	})
	server.AddPrompt(&mcp.Prompt{
		Name: "test_prompt_with_embedded_resource",
		Arguments: []*mcp.PromptArgument{{Name: "resourceUri", Required: true}},
	}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.EmbeddedResource{Resource: &mcp.ResourceContents{
				URI: req.Params.Arguments["resourceUri"], MIMEType: "text/plain",
				Text: "Embedded resource content for testing.",
			}}},
			{Role: "user", Content: &mcp.TextContent{Text: "Please process the embedded resource above."}},
		}}, nil
	})
	server.AddPrompt(&mcp.Prompt{Name: "test_prompt_with_image"},
		func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{
				{Role: "user", Content: &mcp.ImageContent{Data: image, MIMEType: "image/png"}},
				{Role: "user", Content: &mcp.TextContent{Text: "Please analyze the image above."}},
			}}, nil
		})

	server.AddResource(&mcp.Resource{Name: "fixture-text", URI: "test://static-text", MIMEType: "text/plain"},
		func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
				URI: req.Params.URI, MIMEType: "text/plain",
				Text: "This is the content of the static text resource.",
			}}}, nil
		})
	server.AddResource(&mcp.Resource{Name: "fixture-image", URI: "test://static-binary", MIMEType: "image/png"},
		func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
				URI: req.Params.URI, MIMEType: "image/png", Blob: image,
			}}}, nil
		})
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name: "fixture-template", URITemplate: "test://template/{id}/data", MIMEType: "application/json",
	}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI
		id := "123" // the frozen conformance scenario uses this ID
		data, _ := json.Marshal(map[string]any{"id": id, "templateTest": true, "data": "Data for ID: " + id})
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI: uri, MIMEType: "application/json", Text: string(data),
		}}}, nil
	})

	return server, nil
}

func main() {
	server, err := fixtureServer()
	if err != nil {
		log.Fatal(err)
	}
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 1 << 20})
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	// Enforced loopback, no public URL, no OAuth bypass on the real Gateway.
	listener, err := net.Listen("tcp", loopbackAddress)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("TEST-ONLY MCP fixture on %s; no Broker socket or VPS access", loopbackAddress)
	log.Fatal(http.Serve(listener, mux))
}
