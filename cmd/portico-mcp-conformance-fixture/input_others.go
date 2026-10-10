//go:build portico_conformance_fixture

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This is test input exclusively for upstream conformance. It must not be
// available from the production Gateway, including by runtime feature flag.
func labInputEnvelope(state string, inputs map[string]any) (*mcp.CallToolResult, error) {
	payload, err := json.Marshal(map[string]any{
		"resultType": "input_required", "requestState": state, "inputRequests": inputs,
	})
	if err != nil {
		return nil, err
	}
	var result mcp.CallToolResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	if !result.NeedsInput() {
		return nil, errors.New("fixture failed to produce InputRequiredResult")
	}
	return &result, nil
}

func labElicitInput(key, field string) map[string]any {
	return map[string]any{
		"method": "elicitation/create",
		"params": map[string]any{
			"message": fmt.Sprintf("Supply fixture-only %s", field),
			"requestedSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{field: map[string]any{"type": "string"}},
				"required": []string{field},
			},
		},
	}
}
func labSamplingInput() map[string]any {
	return map[string]any{
		"method": "sampling/createMessage",
		"params": map[string]any{
			"messages": []any{map[string]any{
				"role": "user", "content": map[string]any{"type": "text", "text": "Generate a greeting"},
			}},
			"maxTokens": 50,
		},
	}
}
func labRootsInput() map[string]any {
	return map[string]any{"method": "roots/list", "params": map[string]any{}}
}

func registerOtherLabInputTools(server *mcp.Server) error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil { return err }

	specs := []struct {
		name, phase string
		inputs map[string]any
	}{
		{"test_input_required_result_sampling", "sampling", map[string]any{"capital_question": labSamplingInput()}},
		{"test_input_required_result_list_roots", "roots", map[string]any{"client_roots": labRootsInput()}},
		{"test_input_required_result_request_state", "state", map[string]any{"confirm": labElicitInput("confirm", "ok")}},
		{"test_input_required_result_multiple_inputs", "multiple", map[string]any{
			"person": labElicitInput("person", "name"),
			"message": labSamplingInput(),
			"client_roots": labRootsInput(),
		}},
		{"test_input_required_result_capabilities", "capabilities", map[string]any{
			// The official capability test advertises sampling but not elicitation.
			"sampling": labSamplingInput(),
		}},
	}
	for _, spec := range specs {
		spec := spec
		mcp.AddTool(server, &mcp.Tool{
			Name: spec.name, Description: "TEST ONLY: HMAC-bound ephemeral input requests",
		}, func(_ context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
			if len(req.Params.InputResponses) == 0 && req.Params.RequestState == "" {
				out, err := labInputEnvelope(labSignedState(key, spec.phase), spec.inputs)
				return out, struct{}{}, err
			}
			if phase, ok := labVerifyState(key, req.Params.RequestState); !ok || phase != spec.phase {
				return nil, struct{}{}, errors.New("tampered or mismatched requestState")
			}
			for requestID := range spec.inputs {
				if _, ok := req.Params.InputResponses[requestID]; !ok {
					out, err := labInputEnvelope(labSignedState(key, spec.phase), spec.inputs)
					return out, struct{}{}, err
				}
			}
			text := "Synthetic input responses accepted"
			if spec.phase == "state" { text += " state-ok" }
			return &mcp.CallToolResult{Content: textResult(text)}, struct{}{}, nil
		})
	}
	// The "tampered" scenario explicitly requires a JSON-RPC *protocol error*,
	// not a complete tool response with isError=true.
	server.AddTool(&mcp.Tool{Name: "test_input_required_result_tampered_state",
		Description: "TEST ONLY: reject modified signed state", InputSchema: json.RawMessage(`{"type":"object"}`)},
		func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			phase := "tampered"
			if len(req.Params.InputResponses) == 0 && req.Params.RequestState == "" {
				return labInputEnvelope(labSignedState(key, phase), map[string]any{
					"confirm": labElicitInput("confirm", "ok"),
				})
			}
			if received, ok := labVerifyState(key, req.Params.RequestState); !ok || received != phase {
				return nil, errors.New("invalid signed requestState")
			}
			if _, ok := req.Params.InputResponses["confirm"]; !ok {
				return nil, errors.New("missing confirmation")
			}
			return &mcp.CallToolResult{Content: textResult("Signed state verified")}, nil
		})
	return nil
}
