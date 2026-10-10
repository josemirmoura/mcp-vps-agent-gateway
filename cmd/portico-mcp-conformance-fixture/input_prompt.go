//go:build portico_conformance_fixture

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Validate the official prompts/get InputRequiredResult path on the very same
// MCP server the production Gateway registers, without adding a prompt to the
// production catalog. This file is only built under the explicit lab tag.
func registerLabInputPrompt(server *mcp.Server) error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil { return err }
	const phase = "prompt"
	server.AddPrompt(&mcp.Prompt{
		Name: "test_input_required_result_prompt",
		Description: "TEST ONLY: request ephemeral context on prompts/get",
	}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		if len(req.Params.InputResponses) == 0 && req.Params.RequestState == "" {
			wire, err := json.Marshal(map[string]any{
				"resultType": "input_required",
				"requestState": labSignedState(key, phase),
				"inputRequests": map[string]any{
					"prompt_context": labElicitInput("prompt_context", "context"),
				},
			})
			if err != nil { return nil, err }
			var result mcp.GetPromptResult
			if err := json.Unmarshal(wire, &result); err != nil { return nil, err }
			if !result.NeedsInput() { return nil, errors.New("prompts/get lab fixture missing resultType") }
			return &result, nil
		}
		if received, valid := labVerifyState(key, req.Params.RequestState); !valid || received != phase {
			return nil, errors.New("forged prompt requestState")
		}
		if _, ok := req.Params.InputResponses["prompt_context"]; !ok {
			return nil, errors.New("required prompt context not supplied")
		}
		return &mcp.GetPromptResult{
			Messages: []*mcp.PromptMessage{
				{Role: "user", Content: &mcp.TextContent{Text: "Context accepted for test-only prompt"}},
			},
		}, nil
	})
	return nil
}
