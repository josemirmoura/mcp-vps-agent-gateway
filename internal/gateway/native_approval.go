package gateway

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strconv"
 "strings"
 "unicode"
 "unicode/utf8"
)

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
	access, _ := values["access"].(string)
	if (kind == "root" && access != "read" && access != "work" && access != "compose") ||
		(kind == "sensitive" && access != "read") {
		return "", errors.New("unsupported approval access profile")
	}
	for _, key := range []string{targetKey, "access", "physical_ceiling"} {
		value := stringValue(values[key])
		if value == "—" || strings.TrimSpace(value) == "" {
			if key == "physical_ceiling" && kind == "sensitive" {
				continue
			}
			return "", fmt.Errorf("cannot display missing approval field %s", key)
		}
		if !utf8.ValidString(value) {
			return "", errors.New("approval field contains invalid UTF-8")
		}
		for _, r := range value {
			if unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp) {
				return "", errors.New("approval field contains invisible or direction-control characters; use a safe path")
			}
		}
	}
	if kind == "root" && !ceilingWide {
		root, _ := values["root"].(string)
		ceiling, _ := values["physical_ceiling"].(string)
		if filepath.Clean(root) == filepath.Clean(ceiling) {
			return "", errors.New("ceiling-wide approval request must disclose its full scope warning")
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
			"Autorizar Pórtico?\nPasta: %s\nLimite: %s\nPerfil: %s\nDuração: %s",
			target, ceiling, accessLabel(access, true), duration,
		)
		if ceilingWide {
			message += fmt.Sprintf("\nATENÇÃO: inclui todas as pastas atuais e futuras sob %s.", ceiling)
		}
		return message
	}
	message := fmt.Sprintf(
		"Authorize Portico?\nFolder: %s\nCeiling: %s\nProfile: %s\nDuration: %s",
		target, ceiling, accessLabel(access, false), duration,
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
