//go:build portico_conformance_fixture

// Diagnostic tools referenced by the official frozen server-stateless
// scenario. They exercise SDK transport/capability behavior exclusively
// in the disposable lab server and never ship in the production Gateway.
package main

import (
    "context"
    "encoding/json"

    "github.com/modelcontextprotocol/go-sdk/jsonrpc"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerStatelessLabDiagnostics(server *mcp.Server) {
    mcp.AddTool(server, &mcp.Tool{Name:"test_missing_capability",
        Description:"TEST ONLY: require an advertised sampling capability"},
        func(_ context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
            caps := req.ClientCapabilities()
            if caps != nil && caps.Sampling != nil {
                return &mcp.CallToolResult{
                    Content:textResult("Client declared sampling capability; allowed"),
                }, struct{}{}, nil
            }
            metadata := mcp.MissingRequiredClientCapabilityData{
                RequiredCapabilities: &mcp.ClientCapabilities{Sampling:&mcp.SamplingCapabilities{}},
            }
            encoded, err := json.Marshal(metadata)
            if err != nil {return nil,struct{}{},err}
            return nil,struct{}{}, &jsonrpc.Error{
                Code:mcp.CodeMissingRequiredClientCapabilities,
                Message:"sampling capability required but not declared",
                Data:encoded,
            }
        })

    mcp.AddTool(server, &mcp.Tool{Name:"test_streaming_elicitation",
        Description:"TEST ONLY: produce a result without unsupported in-stream requests"},
        func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
            return &mcp.CallToolResult{Content:textResult("Only an MCP result, never an independent request")},struct{}{},nil
        })

    mcp.AddTool(server, &mcp.Tool{Name:"test_logging_tool",
        Description:"TEST ONLY: emit a log only when client opt-in allows it"},
        func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
            // SDK enforces the absence of log notifications unless logLevel is
            // supplied in the protocol request's _meta.
            req.Session.Log(ctx,&mcp.LoggingMessageParams{
                Level:"info",Data:"test_logging_tool executed",
            })
            return &mcp.CallToolResult{Content:textResult("Diagnostic logging attempted")},struct{}{},nil
        })
}
