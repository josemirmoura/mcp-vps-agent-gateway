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
  --surface: #f5f7fa;
  --text: #17191d;
  --muted: #596270;
  --border: #d6dce5;
  --primary: #0b57d0;
  --primary-text: #ffffff;
  --secondary: #ffffff;
  --secondary-text: #24272d;
  --warning-bg: #fff4d6;
  --warning-text: #5d3a00;
  --warning-border: #d79800;
  --success: #176b3a;
  --focus: #3b82f6;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #17191e;
    --surface: #22252c;
    --text: #f4f6f8;
    --muted: #c2c8d0;
    --border: #464d58;
    --primary: #8bc1ff;
    --primary-text: #071321;
    --secondary: #282c34;
    --secondary-text: #f4f6f8;
    --warning-bg: #3a2b0e;
    --warning-text: #ffe2a8;
    --warning-border: #b98520;
    --success: #7dd99f;
    --focus: #9bcaff;
  }
}
* { box-sizing: border-box; }
html, body { min-width: 0; }
body {
  margin: 0;
  padding: 10px;
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  background: transparent;
  color: var(--text);
}
.card {
  display: grid;
  gap: 12px;
  width: 100%;
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--bg);
  color: var(--text);
}
h2 {
  font-size: 16px;
  line-height: 1.3;
  margin: 0;
  color: var(--text);
}
p { margin: 0; line-height: 1.45; color: var(--text); }
.details {
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  display: grid;
  gap: 8px;
  font-size: 13px;
  min-width: 0;
}
.row {
  display: grid;
  grid-template-columns: minmax(72px, 88px) minmax(0, 1fr);
  gap: 8px;
  align-items: start;
}
.label { color: var(--text); }
.value {
  min-width: 0;
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  overflow-wrap: anywhere;
  word-break: break-word;
}
.warning {
  display: none;
  padding: 10px 12px;
  border: 1px solid var(--warning-border);
  border-radius: 10px;
  background: var(--warning-bg);
  color: var(--warning-text);
  font-size: 13px;
  line-height: 1.45;
}
.warning.visible { display: block; }
.warning strong { color: var(--warning-text); }
.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
button {
  min-height: 38px;
  border: 1px solid var(--border);
  border-radius: 9px;
  padding: 8px 12px;
  font: inherit;
  font-weight: 650;
  cursor: pointer;
  transition: filter .12s ease, opacity .12s ease, transform .06s ease;
}
button:hover:not(:disabled) { filter: brightness(1.05); }
button:active:not(:disabled) { transform: translateY(1px); }
button:focus-visible { outline: 3px solid var(--focus); outline-offset: 2px; }
button:disabled { opacity: .48; cursor: default; }
.primary {
  background: var(--primary);
  color: var(--primary-text);
  border-color: var(--primary);
}
.secondary {
  background: var(--secondary);
  color: var(--secondary-text);
  border-color: var(--border);
}
.status {
  font-size: 13px;
  line-height: 1.4;
  min-height: 20px;
  color: var(--text);
}
.status.success { color: var(--success); font-weight: 650; }
.note {
  font-size: 12px;
  line-height: 1.4;
  color: var(--muted);
}
@media (max-width: 380px) {
  body { padding: 6px; }
  .card { padding: 12px; }
  .row { grid-template-columns: 1fr; gap: 2px; }
  .actions { display: grid; grid-template-columns: 1fr; }
  button { width: 100%; }
}
</style>
</head>
<body>
<main class="card">
  <h2 id="title">Autorizar acesso à VPS</h2>
  <p id="intro">O Portico MCP pediu acesso a uma pasta. Confira o escopo antes de autorizar.</p>

  <section class="details" aria-label="Detalhes da autorização">
    <div class="row"><strong class="label" id="rootLabel">Pasta</strong><span class="value" id="root">carregando…</span></div>
    <div class="row"><strong class="label" id="accessLabel">Acesso</strong><span id="access">—</span></div>
    <div class="row"><strong class="label" id="durationLabel">Duração</strong><span id="duration">—</span></div>
  </section>

  <div class="warning" id="ceilingWarning" role="alert"></div>

  <div class="actions">
    <button class="primary" id="approve" disabled>Autorizar</button>
    <button class="secondary" id="deny" disabled>Negar</button>
  </div>

  <div class="status" id="status" role="status" aria-live="polite">Preparando aprovação segura…</div>
  <p class="note" id="note">A autorização usa um token de uso único entregue somente a este card. O modelo não recebe esse token.</p>
</main>
<script>
(() => {
  const rootEl = document.getElementById("root");
  const accessEl = document.getElementById("access");
  const durationEl = document.getElementById("duration");
  const statusEl = document.getElementById("status");
  const approveEl = document.getElementById("approve");
  const denyEl = document.getElementById("deny");
  const warningEl = document.getElementById("ceilingWarning");

  let rpcId = 0;
  const pending = new Map();
  let request = null;
  let approvalToken = "";
  let busy = false;
  let decided = false;

  const browserLang = String(window.openai?.locale || navigator.language || "en").toLowerCase();
  const pt = browserLang.startsWith("pt");

  const copy = pt ? {
    title: "Autorizar acesso à VPS",
    intro: "O Portico MCP pediu acesso a uma pasta. Confira o escopo antes de autorizar.",
    root: "Pasta",
    access: "Acesso",
    duration: "Duração",
    permanent: "Permanente, até revogação",
    authorize: "Autorizar",
    authorizeAll: "Autorizar todo ",
    deny: "Negar",
    waiting: "Aguardando sua decisão.",
    preparing: "Preparando aprovação segura…",
    approving: "Autorizando…",
    denying: "Negando…",
    approved: "Acesso autorizado.",
    denied: "Acesso negado.",
    failed: "Não foi possível concluir: ",
    bridgeFailed: "Este cliente não inicializou o card MCP Apps.",
    note: "A autorização usa um token de uso único entregue somente a este card. O modelo não recebe esse token.",
    warning: (root) => "<strong>Atenção:</strong> esta autorização cobre todo o teto físico <code>" + escapeHTML(root) + "</code>. O Portico poderá usar o perfil solicitado em qualquer pasta atual ou futura dentro desse caminho enquanto a autorização estiver ativa."
  } : {
    title: "Authorize VPS access",
    intro: "Portico MCP requested access to a folder. Review the scope before authorizing.",
    root: "Folder",
    access: "Access",
    duration: "Duration",
    permanent: "Permanent, until revoked",
    authorize: "Authorize",
    authorizeAll: "Authorize all of ",
    deny: "Deny",
    waiting: "Waiting for your decision.",
    preparing: "Preparing secure approval…",
    approving: "Authorizing…",
    denying: "Denying…",
    approved: "Access authorized.",
    denied: "Access denied.",
    failed: "Could not complete: ",
    bridgeFailed: "This client did not initialize the MCP Apps card.",
    note: "Authorization uses a one-time token delivered only to this card. The model never receives that token.",
    warning: (root) => "<strong>Warning:</strong> this authorization covers the entire physical ceiling <code>" + escapeHTML(root) + "</code>. Portico may use the requested profile in any current or future folder inside that path while the authorization remains active."
  };

  document.getElementById("title").textContent = copy.title;
  document.getElementById("intro").textContent = copy.intro;
  document.getElementById("rootLabel").textContent = copy.root;
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
    if (!approvalToken && oa?.toolResponseMetadata) {
      approvalToken = findToken(oa.toolResponseMetadata);
    }
    render();
  }

  function render() {
    const ceilingWide = Boolean(request?.ceiling_wide);
    if (request) {
      rootEl.textContent = request.root || "—";
      accessEl.textContent = request.access || "—";
      const ttl = Number(request.delegation_ttl_seconds || 0);
      durationEl.textContent = ttl > 0 ? formatDuration(ttl) : copy.permanent;

      if (ceilingWide) {
        const ceiling = request.physical_ceiling || request.root || "";
        warningEl.innerHTML = copy.warning(ceiling);
        warningEl.classList.add("visible");
        approveEl.textContent = copy.authorizeAll + ceiling;
      } else {
        warningEl.textContent = "";
        warningEl.classList.remove("visible");
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
    const root = request?.root || "";
    const access = request?.access || "";
    const approved = decision === "approve";
    try {
      await rpcRequest("ui/update-model-context", {
        structuredContent: {
          vpsAgentRootAccess: {
            request_id: request?.request_id,
            root,
            access,
            decision: approved ? "approved" : "denied",
            result: response?.structuredContent || null
          }
        }
      });
    } catch (_) {}

    const text = pt
      ? (approved
          ? "Autorizei o acesso " + access + " a " + root + ". Continue a tarefa original usando essa pasta."
          : "Neguei o acesso a " + root + ". Não use essa pasta e continue sem ampliar esse escopo.")
      : (approved
          ? "I authorized " + access + " access to " + root + ". Continue the original task using that folder."
          : "I denied access to " + root + ". Do not use that folder; continue without expanding scope.");
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
      const response = await rpcRequest("tools/call", {
        name: "permissions.confirm_root_access",
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

    if (message.method === "ui/notifications/tool-result") {
      hydrate(message.params);
    }
  }, { passive: true });

  approveEl.addEventListener("click", () => decide("approve"));
  denyEl.addEventListener("click", () => decide("deny"));

  const bridgeReady = rpcRequest("ui/initialize", {
    appInfo: { name: "portico-mcp-root-approval", version: "1.1.0" },
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
		Name:        "portico-mcp-root-approval",
		Title:       "Portico MCP root access approval",
		Description: "Interactive card for approving or denying a pending VPS root delegation.",
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
						"prefersBorder": true,
						"csp": map[string]any{
							"connectDomains":  []string{},
							"resourceDomains": []string{},
						},
					},
					"openai/widgetDescription": "Review and approve or deny one Portico MCP root delegation request.",
					"openai/widgetPrefersBorder": true,
				},
			}},
		}, nil
	})
}
