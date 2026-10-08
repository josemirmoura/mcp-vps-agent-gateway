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
		{name: "Unicode bidirectional override", key: "root", value: "/opt/payroll\u202eexe.liam"},
		{name: "Unicode bidirectional isolate", key: "physical_ceiling", value: "/opt\u2066"},
		{name: "Unicode paragraph separator", key: "root", value: "/opt/foo\u2029Folder: /"},
		{name: "Unicode line separator", key: "root", value: "/opt/foo\u2028Profile: Work"},
		{name: "Unicode zero-width joiner", key: "root", value: "/opt/fo\u200do"},
		{name: "Unicode C1 next-line control", key: "root", value: "/opt/abc\u0085def"},
		{name: "Unsupported access", key: "access", value: "admin"},
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


// TestNativeApprovalRequiresVisibleAuthorityContext verifies that users never
// authorize a root that is silently the whole physical scope ceiling.
func TestNativeApprovalRequiresVisibleAuthorityContext(t *testing.T) {
	t.Setenv("VPS_AGENT_LANG", "en")
	values := map[string]any{
		"request_id": "req-visible-context",
		"root": "/opt/projeto-á",
		"access": "read",
		"delegation_ttl_seconds": 600,
		"physical_ceiling": "/opt",
		"ceiling_wide": false,
	}
	if _, err := validatedApprovalMessage("root", values); err != nil {
		t.Fatalf("legitimate accented scope must remain displayable: %v", err)
	}
	for _, access := range []string{"read", "work", "compose"} {
		values["access"] = access
		if _, err := validatedApprovalMessage("root", values); err != nil {
			t.Fatalf("valid root profile %q unexpectedly rejected: %v", access, err)
		}
	}
	values["access"] = "read"
	values["physical_ceiling"] = ""
	if _, err := validatedApprovalMessage("root", values); err == nil {
		t.Fatal("native root scope must not conceal the physical ceiling")
	}
	values["physical_ceiling"] = "/opt"
	values["root"] = "/opt/"
	if _, err := validatedApprovalMessage("root", values); err == nil {
		t.Fatal("request for entire physical ceiling must show the full-scope warning")
	}
	values["ceiling_wide"] = true
	if message, err := validatedApprovalMessage("root", values); err != nil {
		t.Fatalf("warned ceiling-wide request must remain displayable: %v", err)
	} else if !strings.Contains(message, "every current and future folder") {
		t.Fatalf("broad scope warning not visible: %s", message)
	}
	values["root"] = "/opt"
	values["physical_ceiling"] = string([]byte{0xff})
	if _, err := validatedApprovalMessage("root", values); err == nil {
		t.Fatal("invalid UTF-8 approval scope must fail closed")
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
