package gateway

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const rootApprovalWidgetURI = "ui://vps-agent/root-approval-v1.html"

const rootApprovalWidgetHTML = `<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
:root {
  color-scheme: light dark;
  --bg: #ffffff;
  --surface: #f6f7f9;
  --surface-strong: #eef1f5;
  --text: #17191d;
  --muted: #68717d;
  --border: #dfe3e8;
  --primary: #111827;
  --primary-hover: #273244;
  --primary-text: #ffffff;
  --secondary: #ffffff;
  --secondary-text: #252a31;
  --accent: #2563eb;
  --accent-soft: #eff6ff;
  --warning-bg: #fff8e6;
  --warning-text: #714b00;
  --warning-border: #e8bf55;
  --danger-bg: #fff1f2;
  --danger-text: #881337;
  --danger-border: #f3a7b7;
  --success: #167647;
  --focus: #60a5fa;
  --shadow: 0 18px 45px rgba(18, 24, 33, .10);
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #181b20;
    --surface: #21252c;
    --surface-strong: #292e37;
    --text: #f4f6f8;
    --muted: #aeb6c2;
    --border: #3c434d;
    --primary: #f3f4f6;
    --primary-hover: #ffffff;
    --primary-text: #111827;
    --secondary: #22262d;
    --secondary-text: #f4f6f8;
    --accent: #8ab4ff;
    --accent-soft: #17233a;
    --warning-bg: #33280f;
    --warning-text: #ffe5a5;
    --warning-border: #8f7028;
    --danger-bg: #351921;
    --danger-text: #ffc2cf;
    --danger-border: #864052;
    --success: #7bdca6;
    --focus: #9bc7ff;
    --shadow: 0 18px 45px rgba(0, 0, 0, .28);
  }
}
* { box-sizing: border-box; }
html, body { min-width: 0; }
body {
  margin: 0;
  padding: 8px;
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  background: transparent;
  color: var(--text);
}
.card {
  width: min(100%, 720px);
  margin: 0 auto;
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--bg);
  color: var(--text);
  box-shadow: var(--shadow);
}
.header {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr);
  gap: 12px;
  align-items: start;
}
.icon {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border-radius: 12px;
  background: var(--accent-soft);
  color: var(--accent);
}
.icon svg { width: 22px; height: 22px; fill: none; stroke: var(--accent); stroke-width: 1.8; }
.eyebrow {
  margin: 0 0 3px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 750;
  letter-spacing: .08em;
  text-transform: uppercase;
}
h2 {
  margin: 0;
  color: var(--text);
  font-size: 18px;
  line-height: 1.25;
  letter-spacing: -.015em;
}
.intro {
  margin: 5px 0 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.45;
}
.scope {
  margin-top: 15px;
  padding: 12px 13px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}
.scope-label {
  display: block;
  margin-bottom: 5px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .04em;
  text-transform: uppercase;
}
.scope-value {
  display: block;
  color: var(--text);
  font: 650 13px/1.4 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  overflow-wrap: anywhere;
  word-break: break-word;
}
.details {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 9px;
  margin-top: 9px;
}
.detail {
  min-width: 0;
  padding: 11px 12px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}
.detail-label {
  display: block;
  margin-bottom: 4px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: .04em;
}
.detail-value {
  color: var(--text);
  font-size: 13px;
  font-weight: 700;
}
.detail-help {
  display: block;
  margin-top: 3px;
  color: var(--muted);
  font-size: 11px;
  line-height: 1.35;
}
.notice {
  display: none;
  margin-top: 10px;
  padding: 11px 12px;
  border: 1px solid var(--warning-border);
  border-radius: 12px;
  background: var(--warning-bg);
  color: var(--warning-text);
  font-size: 12px;
  line-height: 1.45;
}
.notice.visible { display: block; }
.notice.sensitive {
  border-color: var(--danger-border);
  background: var(--danger-bg);
  color: var(--danger-text);
}
.notice strong { color: inherit; }
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
  margin-top: 15px;
}
button {
  min-height: 40px;
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 9px 15px;
  font: inherit;
  font-size: 13px;
  font-weight: 750;
  cursor: pointer;
  transition: background .12s ease, opacity .12s ease, transform .06s ease;
}
button:active:not(:disabled) { transform: translateY(1px); }
button:focus-visible { outline: 3px solid var(--focus); outline-offset: 2px; }
button:disabled { opacity: .45; cursor: default; }
.primary {
  background: var(--primary);
  color: var(--primary-text);
  border-color: var(--primary);
}
.primary:hover:not(:disabled) { background: var(--primary-hover); }
.secondary {
  background: var(--secondary);
  color: var(--secondary-text);
}
.secondary:hover:not(:disabled) { background: var(--surface-strong); }
.footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 12px;
  padding-top: 11px;
  border-top: 1px solid var(--border);
}
.status {
  min-width: 0;
  color: var(--muted);
  font-size: 11px;
  line-height: 1.35;
}
.status.success { color: var(--success); font-weight: 700; }
.note {
  flex: 0 0 auto;
  color: var(--muted);
  font-size: 10px;
  line-height: 1.3;
  text-align: right;
}
@media (max-width: 520px) {
  body { padding: 4px; }
  .card { padding: 15px; border-radius: 15px; }
  .details { grid-template-columns: 1fr; }
  .actions { display: grid; grid-template-columns: 1fr; }
  button { width: 100%; }
  .footer { display: grid; }
  .note { text-align: left; }
}
</style>
</head>
<body>
<main class="card">
  <header class="header">
    <div class="icon" aria-hidden="true">
      <svg viewBox="0 0 24 24"><path d="M12 3 5 6v5c0 4.7 2.9 8.3 7 10 4.1-1.7 7-5.3 7-10V6l-7-3Z"></path><path d="M9.5 12.2 11.2 14l3.7-4"></path></svg>
    </div>
    <div>
      <p class="eyebrow" id="eyebrow">Portico MCP · Permissão</p>
      <h2 id="title">Revisar acesso</h2>
      <p class="intro" id="intro">Confira o que será liberado antes de autorizar.</p>
    </div>
  </header>

  <section class="scope" aria-label="Escopo solicitado">
    <span class="scope-label" id="scopeLabel">Pasta</span>
    <span class="scope-value" id="scopeValue">carregando…</span>
  </section>

  <section class="details" aria-label="Detalhes da autorização">
    <div class="detail">
      <span class="detail-label" id="accessLabel">Acesso</span>
      <span class="detail-value" id="accessValue">—</span>
      <span class="detail-help" id="accessHelp"></span>
    </div>
    <div class="detail">
      <span class="detail-label" id="durationLabel">Duração</span>
      <span class="detail-value" id="durationValue">—</span>
      <span class="detail-help" id="durationHelp"></span>
    </div>
  </section>

  <div class="notice" id="notice" role="alert"></div>

  <div class="actions">
    <button class="secondary" id="deny" disabled>Agora não</button>
    <button class="primary" id="approve" disabled>Autorizar acesso</button>
  </div>

  <footer class="footer">
    <div class="status" id="status" role="status" aria-live="polite">Preparando aprovação segura…</div>
    <div class="note" id="note">Você pode revogar depois.</div>
  </footer>
</main>
<script>
(() => {
  const scopeLabelEl = document.getElementById("scopeLabel");
  const scopeValueEl = document.getElementById("scopeValue");
  const accessValueEl = document.getElementById("accessValue");
  const accessHelpEl = document.getElementById("accessHelp");
  const durationValueEl = document.getElementById("durationValue");
  const durationHelpEl = document.getElementById("durationHelp");
  const statusEl = document.getElementById("status");
  const approveEl = document.getElementById("approve");
  const denyEl = document.getElementById("deny");
  const noticeEl = document.getElementById("notice");

  let rpcId = 0;
  const pending = new Map();
  let request = null;
  let approvalToken = "";
  let busy = false;
  let decided = false;

  const browserLang = String(window.openai?.locale || navigator.language || "en").toLowerCase();
  const pt = browserLang.startsWith("pt");

  const copy = pt ? {
    eyebrowRoot: "Portico MCP · Permissão de pasta",
    eyebrowSensitive: "Portico MCP · Arquivo protegido",
    titleRoot: "Revisar acesso ao projeto",
    titleSensitive: "Liberar arquivo protegido?",
    introRoot: "O Portico quer trabalhar nesta parte da VPS. Você continua no controle do escopo.",
    introSensitive: "Este arquivo continua trancado mesmo quando a pasta do projeto já está autorizada.",
    folder: "Pasta",
    protectedFile: "Arquivo protegido",
    access: "Acesso",
    duration: "Duração",
    permanent: "Permanente",
    permanentHelp: "Fica ativo até você revogar.",
    temporaryHelp: "Expira automaticamente.",
    read: "Somente leitura",
    readHelp: "Pode ver o conteúdo, sem alterar.",
    work: "Trabalho",
    workHelp: "Pode ler, criar, editar e usar shell confinado.",
    compose: "Trabalho + Compose",
    composeHelp: "Inclui operações Compose já permitidas pela política.",
    sensitiveReadHelp: "Pode ler somente este arquivo protegido.",
    sensitiveWorkHelp: "Pode ler e alterar somente este arquivo protegido.",
    authorize: "Autorizar acesso",
    authorizeSensitive: "Autorizar temporariamente",
    authorizeAll: "Autorizar todo ",
    deny: "Agora não",
    waiting: "Pronto para sua decisão.",
    preparing: "Preparando aprovação segura…",
    approving: "Aplicando autorização…",
    denying: "Mantendo acesso bloqueado…",
    approved: "Autorização aplicada.",
    denied: "Acesso mantido bloqueado.",
    failed: "Não foi possível concluir: ",
    bridgeFailed: "O cliente não inicializou o card MCP Apps.",
    note: "Você pode revogar depois.",
    ceilingWarning: (root) => "<strong>Escopo amplo:</strong> autorizar <code>" + escapeHTML(root) + "</code> libera o perfil solicitado para todas as pastas atuais e futuras abaixo desse teto. Faça isso apenas se quiser conscientemente esse alcance.",
    sensitiveWarning: "<strong>Camada extra de proteção:</strong> esta permissão vale somente para este arquivo e é temporária. A autorização normal da pasta não abre arquivos protegidos."
  } : {
    eyebrowRoot: "Portico MCP · Folder permission",
    eyebrowSensitive: "Portico MCP · Protected file",
    titleRoot: "Review project access",
    titleSensitive: "Unlock protected file?",
    introRoot: "Portico wants to work in this part of the VPS. You remain in control of the scope.",
    introSensitive: "This file stays locked even when the project folder is already authorized.",
    folder: "Folder",
    protectedFile: "Protected file",
    access: "Access",
    duration: "Duration",
    permanent: "Permanent",
    permanentHelp: "Remains active until you revoke it.",
    temporaryHelp: "Expires automatically.",
    read: "Read only",
    readHelp: "Can view content without changing it.",
    work: "Work",
    workHelp: "Can read, create, edit and use confined shell.",
    compose: "Work + Compose",
    composeHelp: "Includes Compose operations already allowed by policy.",
    sensitiveReadHelp: "Can read only this protected file.",
    sensitiveWorkHelp: "Can read and modify only this protected file.",
    authorize: "Authorize access",
    authorizeSensitive: "Authorize temporarily",
    authorizeAll: "Authorize all of ",
    deny: "Not now",
    waiting: "Ready for your decision.",
    preparing: "Preparing secure approval…",
    approving: "Applying authorization…",
    denying: "Keeping access locked…",
    approved: "Authorization applied.",
    denied: "Access remains locked.",
    failed: "Could not complete: ",
    bridgeFailed: "The client did not initialize the MCP Apps card.",
    note: "You can revoke this later.",
    ceilingWarning: (root) => "<strong>Broad scope:</strong> authorizing <code>" + escapeHTML(root) + "</code> applies the requested profile to every current and future folder below this ceiling. Do this only if you intentionally want that reach.",
    sensitiveWarning: "<strong>Extra protection layer:</strong> this permission applies only to this file and is temporary. Normal folder authorization never unlocks protected files."
  };

  document.getElementById("accessLabel").textContent = copy.access;
  document.getElementById("durationLabel").textContent = copy.duration;
  document.getElementById("note").textContent = copy.note;
  denyEl.textContent = copy.deny;

  function escapeHTML(value) {
    return String(value).replace(/[&<>"']/g, ch => ({
      "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
    }[ch]));
  }

  function rpcNotify(method, params) {
    window.parent.postMessage({ jsonrpc: "2.0", method, params }, "*");
  }

  function rpcRequest(method, params) {
    const id = ++rpcId;
    window.parent.postMessage({ jsonrpc: "2.0", id, method, params }, "*");
    return new Promise((resolve, reject) => pending.set(id, { resolve, reject }));
  }

  function findToken(meta) {
    if (!meta || typeof meta !== "object") return "";
    if (typeof meta["vps-agent/approvalToken"] === "string") return meta["vps-agent/approvalToken"];
    for (const key of ["mcp_tool_result", "call_tool_result"]) {
      const nested = meta[key];
      if (nested && typeof nested === "object") {
        const found = findToken(nested._meta || nested);
        if (found) return found;
      }
    }
    return "";
  }

  function hydrate(result) {
    if (!result || typeof result !== "object") return;
    const structured = result.structuredContent || result.structured_content || null;
    if (structured && structured.request_id) request = structured;
    const found = findToken(result._meta);
    if (found) approvalToken = found;

    const oa = window.openai;
    if (!request && oa?.toolOutput?.request_id) request = oa.toolOutput;
    if (!approvalToken && oa?.toolResponseMetadata) approvalToken = findToken(oa.toolResponseMetadata);
    render();
  }

  function accessCopy(kind, access) {
    if (access === "read") return [copy.read, kind === "sensitive" ? copy.sensitiveReadHelp : copy.readHelp];
    if (access === "work") return [copy.work, kind === "sensitive" ? copy.sensitiveWorkHelp : copy.workHelp];
    if (access === "compose") return [copy.compose, copy.composeHelp];
    return [access || "—", ""];
  }

  function render() {
    const kind = request?.kind === "sensitive" ? "sensitive" : "root";
    const sensitive = kind === "sensitive";
    const target = sensitive ? (request?.path || "") : (request?.root || "");
    const ceilingWide = !sensitive && Boolean(request?.ceiling_wide);

    document.getElementById("eyebrow").textContent = sensitive ? copy.eyebrowSensitive : copy.eyebrowRoot;
    document.getElementById("title").textContent = sensitive ? copy.titleSensitive : copy.titleRoot;
    document.getElementById("intro").textContent = sensitive ? copy.introSensitive : copy.introRoot;
    scopeLabelEl.textContent = sensitive ? copy.protectedFile : copy.folder;

    if (request) {
      scopeValueEl.textContent = target || "—";
      const [accessTitle, accessHelp] = accessCopy(kind, request.access);
      accessValueEl.textContent = accessTitle;
      accessHelpEl.textContent = accessHelp;

      const ttl = Number(request.delegation_ttl_seconds || 0);
      durationValueEl.textContent = ttl > 0 ? formatDuration(ttl) : copy.permanent;
      durationHelpEl.textContent = ttl > 0 ? copy.temporaryHelp : copy.permanentHelp;

      noticeEl.classList.remove("visible", "sensitive");
      noticeEl.textContent = "";
      if (sensitive) {
        noticeEl.innerHTML = copy.sensitiveWarning;
        noticeEl.classList.add("visible", "sensitive");
        approveEl.textContent = copy.authorizeSensitive;
      } else if (ceilingWide) {
        const ceiling = request.physical_ceiling || target;
        noticeEl.innerHTML = copy.ceilingWarning(ceiling);
        noticeEl.classList.add("visible");
        approveEl.textContent = copy.authorizeAll + ceiling;
      } else {
        approveEl.textContent = copy.authorize;
      }
    }

    const ready = Boolean(request?.request_id && approvalToken && !busy && !decided);
    approveEl.disabled = !ready;
    denyEl.disabled = !ready;
    if (!decided && !busy) {
      statusEl.classList.remove("success");
      statusEl.textContent = ready ? copy.waiting : copy.preparing;
    }
  }

  function formatDuration(seconds) {
    if (seconds % 3600 === 0) return (seconds / 3600) + " h";
    if (seconds % 60 === 0) return (seconds / 60) + " min";
    return seconds + " s";
  }

  async function sendModelContext(decision, response) {
    const kind = request?.kind === "sensitive" ? "sensitive" : "root";
    const target = kind === "sensitive" ? (request?.path || "") : (request?.root || "");
    const access = request?.access || "";
    const approved = decision === "approve";
    try {
      await rpcRequest("ui/update-model-context", {
        structuredContent: {
          vpsAgentAccess: {
            request_id: request?.request_id,
            kind,
            target,
            access,
            decision: approved ? "approved" : "denied",
            result: response?.structuredContent || null
          }
        }
      });
    } catch (_) {}

    const text = pt
      ? (approved
          ? "Autorizei o acesso " + access + " a " + target + ". Continue a tarefa original usando apenas esse escopo."
          : "Neguei o acesso a " + target + ". Continue sem ampliar esse escopo.")
      : (approved
          ? "I authorized " + access + " access to " + target + ". Continue the original task using only that scope."
          : "I denied access to " + target + ". Continue without expanding that scope.");
    try {
      await rpcRequest("ui/message", {
        role: "user",
        content: [{ type: "text", text }]
      });
    } catch (_) {
      try {
        await window.openai?.sendFollowUpMessage?.({ prompt: text, scrollToBottom: true });
      } catch (_) {}
    }
  }

  async function decide(decision) {
    if (busy || decided || !request?.request_id || !approvalToken) return;
    busy = true;
    statusEl.classList.remove("success");
    statusEl.textContent = decision === "approve" ? copy.approving : copy.denying;
    render();
    try {
      const toolName = request?.kind === "sensitive"
        ? "permissions.confirm_sensitive_access"
        : "permissions.confirm_root_access";
      const response = await rpcRequest("tools/call", {
        name: toolName,
        arguments: {
          request_id: request.request_id,
          approval_token: approvalToken,
          decision
        }
      });
      if (response?.isError) throw new Error("server refused confirmation");
      decided = true;
      approvalToken = "";
      statusEl.textContent = decision === "approve" ? copy.approved : copy.denied;
      if (decision === "approve") statusEl.classList.add("success");
      await sendModelContext(decision, response);
    } catch (error) {
      statusEl.textContent = copy.failed + (error?.message || String(error));
    } finally {
      busy = false;
      render();
    }
  }

  window.addEventListener("message", (event) => {
    if (event.source !== window.parent) return;
    const message = event.data;
    if (!message || message.jsonrpc !== "2.0") return;

    if (message.id !== undefined && pending.has(message.id)) {
      const waiter = pending.get(message.id);
      pending.delete(message.id);
      if (message.error) waiter.reject(message.error);
      else waiter.resolve(message.result);
      return;
    }

    if (message.method === "ui/notifications/tool-result") hydrate(message.params);
  }, { passive: true });

  approveEl.addEventListener("click", () => decide("approve"));
  denyEl.addEventListener("click", () => decide("deny"));

  const bridgeReady = rpcRequest("ui/initialize", {
    appInfo: { name: "portico-mcp-access-approval", version: "1.2.0" },
    appCapabilities: {},
    protocolVersion: "2026-01-26"
  }).then(() => {
    rpcNotify("ui/notifications/initialized", {});
    hydrate({
      structuredContent: window.openai?.toolOutput,
      _meta: window.openai?.toolResponseMetadata
    });
  }).catch((error) => {
    statusEl.textContent = copy.bridgeFailed;
    console.error(error);
  });

  void bridgeReady;
  render();
})();
</script>
</body>
</html>`

func registerRootApprovalWidget(server *mcp.Server) {
	server.AddResource(&mcp.Resource{
		Name:        "portico-mcp-access-approval",
		Title:       "Portico MCP access approval",
		Description: "Interactive card for reviewing and deciding a pending Portico MCP authorization request.",
		URI:         rootApprovalWidgetURI,
		MIMEType:    "text/html;profile=mcp-app",
	}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      rootApprovalWidgetURI,
				MIMEType: "text/html;profile=mcp-app",
				Text:     rootApprovalWidgetHTML,
				Meta: mcp.Meta{
					"ui": map[string]any{
						"prefersBorder": false,
						"csp": map[string]any{
							"connectDomains":  []string{},
							"resourceDomains": []string{},
						},
					},
					"openai/widgetDescription": "Review exactly what Portico MCP is requesting, then authorize or keep it locked.",
					"openai/widgetPrefersBorder": false,
				},
			}},
		}, nil
	})
}
