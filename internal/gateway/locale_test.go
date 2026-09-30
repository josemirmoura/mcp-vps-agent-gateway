package gateway

import (
	"strings"
	"testing"
)

func TestOfficialGatewayLocales(t *testing.T) {
	tests := []struct {
		lang          string
		instruction   string
		approvalTitle string
	}{
		{"en", "You are connected to Portico MCP", "Authorize Portico MCP access?"},
		{"pt-BR", "Você está conectado ao Portico MCP", "Autorizar acesso do Portico MCP?"},
		{"es", "Estás conectado a Portico MCP", "¿Autorizar acceso de Portico MCP?"},
		{"de", "Sie sind mit Portico MCP verbunden", "Portico-MCP-Zugriff autorisieren?"},
		{"fr", "Vous êtes connecté à Portico MCP", "Autoriser l'accès de Portico MCP ?"},
		{"ja", "Portico MCP に接続されています", "Portico MCP のアクセスを許可しますか？"},
		{"id", "Anda terhubung ke Portico MCP", "Izinkan akses Portico MCP?"},
	}

	for _, tc := range tests {
		t.Run(tc.lang, func(t *testing.T) {
			t.Setenv("VPS_AGENT_LANG", tc.lang)
			if got := serverInstructions(); !strings.Contains(got, tc.instruction) {
				t.Fatalf("server instructions for %s are not localized: %q", tc.lang, got)
			}
			got := approvalMessage("root", map[string]any{
				"root":                   "/opt/project-a",
				"access":                 "work",
				"delegation_ttl_seconds": 3600,
				"physical_ceiling":       "/opt",
				"ceiling_wide":           false,
			})
			if !strings.Contains(got, tc.approvalTitle) ||
				!strings.Contains(got, "/opt/project-a") ||
				!strings.Contains(got, ".env") {
				t.Fatalf("approval copy for %s lost required context: %q", tc.lang, got)
			}
		})
	}
}

func TestGatewayLocaleNormalization(t *testing.T) {
	tests := map[string]string{
		"en_US.UTF-8": "en",
		"pt_BR.UTF-8": "pt-BR",
		"es_MX.UTF-8": "es",
		"de_DE.UTF-8": "de",
		"fr_FR.UTF-8": "fr",
		"ja_JP.UTF-8": "ja",
		"id_ID.UTF-8": "id",
		"xx_YY.UTF-8": "en",
	}
	for raw, want := range tests {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("VPS_AGENT_LANG", raw)
			if got := gatewayLang(); got != want {
				t.Fatalf("gatewayLang(%q)=%q want %q", raw, got, want)
			}
		})
	}
}
