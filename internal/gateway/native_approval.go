package gateway

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const nativeApprovalInputKey = "portico_approval"

type nativeApprovalState struct {
	Version       int    `json:"v"`
	Kind          string `json:"kind"`
	RequestID     string `json:"request_id"`
	ApprovalToken string `json:"approval_token"`
}

func supportsNativeElicitation(req *mcp.CallToolRequest) bool {
	if req == nil {
		return false
	}
	caps := req.ClientCapabilities()
	return caps != nil && caps.Elicitation != nil
}

func encodeNativeApprovalState(state nativeApprovalState) (string, error) {
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Kind == "" || state.RequestID == "" || state.ApprovalToken == "" {
		return "", errors.New("incomplete approval state")
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeNativeApprovalState(raw, expectedKind string) (nativeApprovalState, error) {
	var state nativeApprovalState
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return state, errors.New("invalid approval request state")
	}
	if err := json.Unmarshal(payload, &state); err != nil {
		return state, errors.New("invalid approval request state")
	}
	if state.Version != 1 || state.Kind != expectedKind || state.RequestID == "" || state.ApprovalToken == "" {
		return state, errors.New("approval request state does not match this operation")
	}
	return state, nil
}

func nativeApprovalResult(kind string, brokerResult map[string]any, approvalToken string) (*mcp.CallToolResult, error) {
	requestID, _ := brokerResult["request_id"].(string)
	state, err := encodeNativeApprovalState(nativeApprovalState{
		Version:       1,
		Kind:          kind,
		RequestID:     requestID,
		ApprovalToken: approvalToken,
	})
	if err != nil {
		return nil, err
	}
	message, err := validatedApprovalMessage(kind, brokerResult)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		InputRequests: mcp.InputRequestMap{
			nativeApprovalInputKey: &mcp.ElicitParams{
				Message: message,
				RequestedSchema: &jsonschema.Schema{
					Type: "object",
				},
			},
		},
		RequestState: state,
	}, nil
}

func nativeApprovalDecision(req *mcp.CallToolRequest, expectedKind string) (nativeApprovalState, string, bool, error) {
	if req == nil || len(req.Params.InputResponses) == 0 {
		return nativeApprovalState{}, "", false, nil
	}
	state, err := decodeNativeApprovalState(req.Params.RequestState, expectedKind)
	if err != nil {
		return nativeApprovalState{}, "", true, err
	}
	raw, ok := req.Params.InputResponses[nativeApprovalInputKey]
	if !ok {
		return nativeApprovalState{}, "", true, errors.New("native approval response is missing")
	}
	result, ok := raw.(*mcp.ElicitResult)
	if !ok || result == nil {
		return nativeApprovalState{}, "", true, errors.New("native approval response has an unexpected type")
	}
	switch result.Action {
	case "accept":
		return state, "approve", true, nil
	case "decline", "cancel":
		return state, "deny", true, nil
	default:
		return nativeApprovalState{}, "", true, fmt.Errorf("unsupported approval action %q", result.Action)
	}
}

// validatedApprovalMessage never asks a user to approve a scope that cannot
// be displayed legibly, completely, and without attacker-controlled line breaks.
// Existing broker authority checks remain the final source of truth.
func validatedApprovalMessage(kind string, values map[string]any) (string, error) {
	if kind != "root" && kind != "sensitive" {
		return "", errors.New("unsupported native approval kind")
	}
	targetKey := "root"
	if kind == "sensitive" {
		targetKey = "path"
	}
	ceilingWide, _ := values["ceiling_wide"].(bool)
	for _, key := range []string{targetKey, "access", "physical_ceiling"} {
		value := stringValue(values[key])
		if value == "—" || strings.TrimSpace(value) == "" {
			if key == "physical_ceiling" && kind == "sensitive" {
				continue
			}
			if key == "physical_ceiling" && !ceilingWide {
				continue
			}
			return "", fmt.Errorf("cannot display missing approval field %s", key)
		}
		for _, r := range value {
			if r < 32 || r == 127 {
				return "", errors.New("approval target contains control characters; request a safe path")
			}
		}
	}
	message := approvalMessage(kind, values)
	if len(message) > 220 || strings.Count(message, "\n") > 5 {
		return "", errors.New("approval request exceeds the native mobile display limit; use a shorter path")
	}
	return message, nil
}

func approvalMessage(kind string, values map[string]any) string {
	pt := strings.EqualFold(strings.TrimSpace(os.Getenv("VPS_AGENT_LANG")), "pt-BR")
	access := stringValue(values["access"])
	duration := durationValue(values["delegation_ttl_seconds"], pt)

	if kind == "sensitive" {
		target := stringValue(values["path"])
		if pt {
			return fmt.Sprintf(
				"Autorizar arquivo protegido?\nArquivo: %s\nPerfil: %s\nDuração: %s\nSomente este arquivo será liberado.",
				target, accessLabel(access, true), duration,
			)
		}
		return fmt.Sprintf(
			"Authorize protected file?\nFile: %s\nProfile: %s\nDuration: %s\nOnly this file will be unlocked.",
			target, accessLabel(access, false), duration,
		)
	}

	target := stringValue(values["root"])
	ceilingWide, _ := values["ceiling_wide"].(bool)
	ceiling := stringValue(values["physical_ceiling"])
	if pt {
		message := fmt.Sprintf(
			"Autorizar Pórtico?\nPasta: %s\nPerfil: %s\nDuração: %s",
			target, accessLabel(access, true), duration,
		)
		if ceilingWide {
			message += fmt.Sprintf("\nATENÇÃO: inclui todas as pastas atuais e futuras sob %s.", ceiling)
		}
		return message
	}
	message := fmt.Sprintf(
		"Authorize Portico?\nFolder: %s\nProfile: %s\nDuration: %s",
		target, accessLabel(access, false), duration,
	)
	if ceilingWide {
		message += fmt.Sprintf("\nWARNING: includes every current and future folder under %s.", ceiling)
	}
	return message
}

func accessLabel(access string, pt bool) string {
	switch access {
	case "read":
		if pt {
			return "Leitura"
		}
		return "Read"
	case "work":
		if pt {
			return "Trabalho (leitura/escrita + shell confinado)"
		}
		return "Work (read/write + confined shell)"
	case "compose":
		if pt {
			return "Trabalho + Compose"
		}
		return "Work + Compose"
	default:
		return access
	}
}

func durationValue(value any, pt bool) string {
	seconds := int64Value(value)
	if seconds <= 0 {
		if pt {
			return "Permanente, até revogação"
		}
		return "Permanent, until revoked"
	}
	if seconds%3600 == 0 {
		return fmt.Sprintf("%d h", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("%d min", seconds/60)
	}
	if pt {
		return fmt.Sprintf("%d s", seconds)
	}
	return fmt.Sprintf("%d sec", seconds)
}

func stringValue(value any) string {
	if value == nil {
		return "—"
	}
	if s, ok := value.(string); ok && s != "" {
		return s
	}
	return fmt.Sprint(value)
}

func int64Value(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}
