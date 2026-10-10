//go:build portico_conformance_fixture

// This process is ONLY built with -tags=portico_conformance_fixture.
// It runs a real Portico MCP server registration against synthetic upstream
// conformance probes. It is never linked into the production Gateway binary.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
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

// Lab-only state binding: even this loopback fixture does not accept a naked
// client-supplied phase. A random per-process HMAC key authenticates round state.
func labSignedState(key []byte, phase string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(phase))
	return phase + "." + hex.EncodeToString(mac.Sum(nil))
}

func labVerifyState(key []byte, raw string) (string, bool) {
	for _, phase := range []string{"round-1", "round-2", "basic", "sampling", "roots", "state", "multiple", "capabilities", "tampered", "prompt"} {
		if hmac.Equal([]byte(raw), []byte(labSignedState(key, phase))) {
			return phase, true
		}
	}
	return "", false
}

// The pinned official runner expects SEP-2322 input_required responses. The
// SDK's Go result type stores its discriminator privately; unmarshaling the
// well-formed wire representation is a public, validation-preserving path.
func labInputRequired(state string, key string, field string) (*mcp.CallToolResult, error) {
	message := "Provide synthetic conformance data"
	request := map[string]any{
		"resultType": "input_required",
		"requestState": state,
		"inputRequests": map[string]any{key: map[string]any{
			"method": "elicitation/create",
			"params": map[string]any{
				"message": message,
				"requestedSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{field: map[string]any{"type": "string"}},
					"required": []string{field},
				},
			},
		}},
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	var out mcp.CallToolResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if !out.NeedsInput() {
		return nil, errors.New("test fixture could not construct input_required result")
	}
	return &out, nil
}

func registerLabInputTools(server *mcp.Server) error {
	stateKey := make([]byte, 32)
	if _, err := rand.Read(stateKey); err != nil { return err }
	mcp.AddTool(server, &mcp.Tool{Name: "test_input_required_result_elicitation",
		Description: "TEST ONLY, basic elicitation over InputRequiredResult."},
		func(_ context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
			if len(req.Params.InputResponses) == 0 || req.Params.InputResponses["user_name"] == nil {
				out, err := labInputRequired(labSignedState(stateKey, "basic"), "user_name", "name")
				return out, struct{}{}, err
			}
			if phase, ok := labVerifyState(stateKey, req.Params.RequestState); !ok || phase != "basic" {
				return nil, struct{}{}, errors.New("invalid or tampered fixture requestState")
			}
			if _, ok := req.Params.InputResponses["user_name"]; !ok {
				return nil, struct{}{}, errors.New("missing user_name inputResponse")
			}
			return &mcp.CallToolResult{Content: textResult("Hello, Alice!")}, struct{}{}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "test_input_required_result_multi_round",
		Description: "TEST ONLY, two HMAC-bound elicitation rounds followed by completion."},
		func(_ context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
			if req.Params.RequestState == "" && len(req.Params.InputResponses) == 0 {
				out, err := labInputRequired(labSignedState(stateKey, "round-1"), "step1", "name")
				return out, struct{}{}, err
			}
			phase, ok := labVerifyState(stateKey, req.Params.RequestState)
			if !ok { return nil, struct{}{}, errors.New("invalid or tampered fixture requestState") }
			switch phase {
			case "round-1":
				if _, ok := req.Params.InputResponses["step1"]; !ok {
					return nil, struct{}{}, errors.New("missing step1 inputResponse")
				}
				out, err := labInputRequired(labSignedState(stateKey, "round-2"), "step2", "color")
				return out, struct{}{}, err
			case "round-2":
				if _, ok := req.Params.InputResponses["step2"]; !ok {
					return nil, struct{}{}, errors.New("missing step2 inputResponse")
				}
				return &mcp.CallToolResult{Content: textResult("Name Alice and favorite color blue")}, struct{}{}, nil
		default:
				return nil, struct{}{}, errors.New("unsupported fixture round")
			}
		})
	return nil
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
	if err := registerLabInputTools(server); err != nil { return nil, err }
	if err := registerOtherLabInputTools(server); err != nil { return nil, err }
	if err := registerLabInputPrompt(server); err != nil { return nil, err }

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
	mcp.AddTool(server, &mcp.Tool{Name: "test_tool_with_progress",
		Description: "Conformance fixture progress, never available to production clients."},
		func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
			token := req.Params.GetProgressToken()
			if token != nil {
				for _, progress := range []float64{0, 50, 100} {
					if err := req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
						ProgressToken: token, Progress: progress, Total: 100,
					}); err != nil {
						return nil, struct{}{}, err
					}
					time.Sleep(15 * time.Millisecond)
				}
			}
			return &mcp.CallToolResult{Content: textResult("Progress diagnostic executed")}, struct{}{}, nil
		})

	textPrompt := func(name, text string) {
		server.AddPrompt(&mcp.Prompt{Name: name, Description: "Conformance diagnostic prompt; test-only fixture."},
			func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
				return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{
					Role: "user", Content: &mcp.TextContent{Text: text},
				}}}, nil
			})
	}
	textPrompt("test_simple_prompt", "This is a simple prompt for testing.")
	server.AddPrompt(&mcp.Prompt{
		Name: "test_prompt_with_arguments", Description: "Two argument substitution diagnostic.",
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
		Name: "test_prompt_with_embedded_resource", Description: "Embedded resource diagnostic.",
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
	server.AddPrompt(&mcp.Prompt{Name: "test_prompt_with_image", Description: "Image diagnostic."},
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
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: false, MaxRequestBodyBytes: 1 << 20})
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
