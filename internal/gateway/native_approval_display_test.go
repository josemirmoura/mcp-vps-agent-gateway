package gateway

import (
	"strings"
	"testing"
)

func TestNativeApprovalRejectsControlCharactersAndMobileOverflow(t *testing.T) {
	valid := map[string]any{
		"request_id":              "req-test-1",
		"root":                    "/opt/project-a",
		"access":                  "work",
		"delegation_ttl_seconds": 3600,
		"physical_ceiling":        "/opt",
		"approval_token":          "ephemeral-test-token",
	}
	if _, err := nativeApprovalResult("root", valid, "ephemeral-test-token"); err != nil {
		t.Fatalf("valid native approval should remain available: %v", err)
	}
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "injected newline", key: "root", value: "/opt/a\\nPerfil: Admin\nDuração: Permanente"},
		{name: "embedded carriage return", key: "root", value: "/opt/a\rtrick"},
		{name: "embedded tab", key: "root", value: "/opt/a\ttrick"},
		{name: "overlong folder", key: "root", value: "/opt/" + strings.Repeat("nested", 35)},
		{name: "spoofed access", key: "access", value: "read\nPermission granted"},
		{name: "spoofed ceiling", key: "physical_ceiling", value: "/opt\nEverything allowed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := make(map[string]any)
			for k, v := range valid {
				values[k] = v
			}
			values[tc.key] = tc.value
			if _, err := nativeApprovalResult("root", values, "ephemeral-test-token"); err == nil {
				t.Fatalf("unsafe native approval must not be displayed: %s", tc.name)
			}
		})
	}
}

func TestNativeApprovalStateDoesNotCrossApprovalKinds(t *testing.T) {
	state, err := encodeNativeApprovalState(nativeApprovalState{
		Kind: "root", RequestID: "apr-test", ApprovalToken: "token-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeNativeApprovalState(state, "sensitive"); err == nil {
		t.Fatal("root approval token cannot authorize protected-file operation")
	}
	if _, err := decodeNativeApprovalState(state, "root"); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeNativeApprovalState("invalid", "root"); err == nil {
		t.Fatal("invalid request state must be rejected")
	}
}
