package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearer struct{ token string }

func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(req)
}

func main() {
	endpoint := flag.String("endpoint", "http://127.0.0.1:8080/mcp", "MCP endpoint")
	token := flag.String("token", os.Getenv("VPS_AGENT_MCP_TOKEN"), "Bearer token")
	tool := flag.String("tool", "", "tool name")
	argsRaw := flag.String("args", "{}", "JSON tool arguments")
	expectError := flag.Bool("expect-error", false, "succeed only when the tool returns an error")
	timeout := flag.Duration("timeout", 30*time.Second, "call timeout")
	flag.Parse()
	if *tool == "" {
		fmt.Fprintln(os.Stderr, "--tool is required")
		os.Exit(2)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(*argsRaw), &args); err != nil {
		fmt.Fprintln(os.Stderr, "invalid --args JSON:", err)
		os.Exit(2)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "vps-agent-mcp-call", Version: "v1"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: *endpoint}
	if *token != "" {
		transport.HTTPClient = &http.Client{Transport: bearer{token: *token}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer session.Close()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: *tool, Arguments: args})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out := map[string]any{
		"tool": *tool, "is_error": result.IsError,
		"structured_content": result.StructuredContent,
		"content": result.Content,
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(raw))
	if *expectError {
		if !result.IsError {
			os.Exit(1)
		}
		return
	}
	if result.IsError {
		os.Exit(1)
	}
}
