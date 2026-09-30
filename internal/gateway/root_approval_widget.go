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
:root { color-scheme: light dark; }
* { box-sizing: border-box; }
body {
  margin: 0;
  padding: 14px;
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  background: var(--color-background-primary, transparent);
  color: var(--color-text-primary, inherit);
}
.card { display: grid; gap: 12px; }
h2 { font-size: 16px; margin: 0; }
p { margin: 0; line-height: 1.45; }
.details {
  padding: 10px 12px;
  border: 1px solid var(--color-border-secondary, rgba(128,128,128,.35));
  border-radius: 10px;
  display: grid;
  gap: 7px;
  font-size: 13px;
}
.row { display: grid; grid-template-columns: 84px 1fr; gap: 8px; }
.value { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; overflow-wrap: anywhere; }
.actions { display: flex; gap: 8px; flex-wrap: wrap; }
button {
  border: 1px solid var(--color-border-primary, rgba(128,128,128,.45));
  border-radius: 9px;
  padding: 8px 12px;
  font: inherit;
  cursor: pointer;
}
button:disabled { opacity: .55; cursor: default; }
.primary {
  background: var(--color-background-inverse, currentColor);
  color: var(--color-text-inverse, Canvas);
}
.status { font-size: 13px; min-height: 20px; }
.note { font-size: 12px; opacity: .78; }
</style>
</head>
<body>
<main class="card">
  <h2>Autorizar acesso à VPS</h2>
  <p>O MCP pediu acesso a uma nova pasta. Confira o escopo antes de autorizar.</p>
  <section class="details">
    <div class="row"><strong>Pasta</strong><span class="value" id="root">carregando…</span></div>
    <div class="row"><strong>Acesso</strong><span id="access">—</span></div>
    <div class="row"><strong>Duração</strong><span id="duration">—</span></div>
  </section>
  <div class="actions">
    <button class="primary" id="approve" disabled>Autorizar</button>
    <button id="deny" disabled>Negar</button>
  </div>
  <div class="status" id="status">Preparando aprovação segura…</div>
  <p class="note">A autorização usa um token de uso único entregue somente a este card. O modelo não recebe esse token.</p>
</main>
<script>
(() => {
  const rootEl = document.getElementById("root");
  const accessEl = document.getElementById("access");
  const durationEl = document.getElementById("duration");
  const statusEl = document.getElementById("status");
  const approveEl = document.getElementById("approve");
  const denyEl = document.getElementById("deny");

  let rpcId = 0;
  const pending = new Map();
  let request = null;
  let approvalToken = "";
  let busy = false;
  let decided = false;

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
    if (request) {
      rootEl.textContent = request.root || "—";
      accessEl.textContent = request.access || "—";
      const ttl = Number(request.delegation_ttl_seconds || 0);
      durationEl.textContent = ttl > 0 ? formatDuration(ttl) : "Permanente, até revogação";
    }
    const ready = Boolean(request?.request_id && approvalToken && !busy && !decided);
    approveEl.disabled = !ready;
    denyEl.disabled = !ready;
    if (!decided && !busy) {
      statusEl.textContent = ready
        ? "Aguardando sua decisão."
        : "Preparando aprovação segura…";
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

    const text = approved
      ? "Autorizei o acesso " + access + " a " + root + ". Continue a tarefa original usando essa pasta."
      : "Neguei o acesso a " + root + ". Não use essa pasta e continue sem ampliar esse escopo.";
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
    statusEl.textContent = decision === "approve" ? "Autorizando…" : "Negando…";
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
      if (response?.isError) throw new Error("A confirmação foi recusada pelo servidor.");
      decided = true;
      approvalToken = "";
      statusEl.textContent = decision === "approve"
        ? "Acesso autorizado."
        : "Acesso negado.";
      await sendModelContext(decision, response);
    } catch (error) {
      statusEl.textContent = "Não foi possível concluir: " + (error?.message || String(error));
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
    appInfo: { name: "vps-agent-root-approval", version: "1.0.0" },
    appCapabilities: {},
    protocolVersion: "2026-01-26"
  }).then(() => {
    rpcNotify("ui/notifications/initialized", {});
    hydrate({
      structuredContent: window.openai?.toolOutput,
      _meta: window.openai?.toolResponseMetadata
    });
  }).catch((error) => {
    statusEl.textContent = "Este cliente não inicializou o card MCP Apps.";
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
		Name:        "vps-agent-root-approval",
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
					"openai/widgetDescription": "Review and approve or deny one VPS root delegation request.",
					"openai/widgetPrefersBorder": true,
				},
			}},
		}, nil
	})
}
