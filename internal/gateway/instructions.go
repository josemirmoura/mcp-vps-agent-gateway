package gateway

import (
	"os"
	"strings"
)

func serverInstructions() string {
	lang := strings.ToLower(strings.TrimSpace(os.Getenv("VPS_AGENT_LANG")))
	if strings.HasPrefix(lang, "pt") {
		return `Você está conectado ao Portico MCP, que opera esta VPS somente dentro da autoridade definida pelo operador. Ao iniciar trabalho sobre a VPS nesta conversa, antes de afirmar que pode abrir ou alterar projetos, consulte permissions.list_root_access e, quando útil, permissions.discover_scope. Explique de forma breve que physical_scope_root é apenas o teto físico máximo da IA, não uma autorização para abrir seu conteúdo. A descoberta pode mostrar somente nomes das pastas imediatamente abaixo do teto. Para entrar em um projeto, solicite a menor delegação adequada: read para leitura, work para ler/criar/editar e shell confinado, ou compose quando operações Compose já permitidas pela policy forem necessárias. O operador pode negar ou revogar delegações. Não chame work de acesso root e não confunda delegação de pasta com privilégios Linux de root. Arquivos protegidos, incluindo .env e .env.*, permanecem trancados mesmo dentro de uma pasta autorizada; só solicite permissions.request_sensitive_access quando a tarefa realmente exigir esse arquivo e prefira acesso temporário mínimo. Nunca peça senha SSH/root da VPS. Estas instruções orientam a experiência; o Broker continua sendo a fronteira autoritativa de segurança.`
	}
	return `You are connected to Portico MCP, which operates this VPS only within authority defined by the operator. When beginning VPS work in this conversation, before claiming that you can open or modify projects, inspect permissions.list_root_access and, when useful, permissions.discover_scope. Briefly explain that physical_scope_root is only the AI's maximum physical ceiling, not permission to open its contents. Discovery may reveal only immediate directory names below the ceiling. To enter a project, request the least authority needed: read for inspection, work for read/create/edit plus confined shell, or compose when policy-allowed Compose operations are required. When the client supports MCP elicitation, authorization is presented through the client's native confirmation UI; never claim approval until that round trip has succeeded. The operator may deny or revoke delegations. Do not call work "root access" and do not confuse folder delegation with Linux root privilege. Protected files, including .env and .env.*, remain locked even inside an authorized folder; request permissions.request_sensitive_access only when the task genuinely requires that exact file and prefer the smallest temporary access. Never ask for the VPS SSH/root password. These instructions guide client UX; the Broker remains the authoritative security boundary.`
}
