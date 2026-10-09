package gateway

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const approvalAppURI = "ui://portico/authorizations-v1.html"
const approvalAppMIME = "text/html;profile=mcp-app"
const approvalExtension = "io.modelcontextprotocol/ui"
const navigationInputKey = "portico_operator_navigation"

//go:embed approval_app.html
var approvalAppHTML string

type approvalClientSupport struct { Apps, Form, URL bool }

func approvalCapabilities(caps *mcp.ClientCapabilities) approvalClientSupport {
	var out approvalClientSupport
	if caps==nil { return out }
	if raw,ok:=caps.Extensions[approvalExtension]; ok {
		data,err:=json.Marshal(raw)
		var ui struct { MIME []string `json:"mimeTypes"` }
		if err==nil && json.Unmarshal(data,&ui)==nil {
			for _,mime:=range ui.MIME { if mime==approvalAppMIME { out.Apps=true } }
		}
	}
	if e:=caps.Elicitation;e!=nil {
		out.Form=e.Form!=nil || (e.Form==nil && e.URL==nil)
		out.URL=e.URL!=nil
	}
	return out
}

func appEnabled() bool { return os.Getenv("VPS_AGENT_MCP_APPS")=="1" }

func approvalTool(name,description string) *mcp.Tool {
	t:=annotatedTool(name,description)
	if appEnabled() && operatorApprovalURL("apr_resource_check")!="" {
		t.Meta=mcp.Meta{"ui":map[string]any{"resourceUri":approvalAppURI,"visibility":[]string{"model","app"}}}
	}
	return t
}

func registerApprovalApp(server *mcp.Server) {
	link:=operatorApprovalURL("apr_resource_check")
	if !appEnabled() || link=="" { return }
	u,_:=url.Parse(link)
	origin:=u.Scheme+"://"+u.Host
	config,_:=json.Marshal(map[string]any{"operatorOrigin":origin,"embedded":os.Getenv("PORTICO_OPERATOR_FRAME_ANCESTORS")!=""})
	page:=strings.Replace(approvalAppHTML,"__PORTICO_CONFIG__",string(config),1)
	server.AddResource(&mcp.Resource{URI:approvalAppURI,Name:"portico-authorizations",MIMEType:approvalAppMIME},
		func(context.Context,*mcp.ReadResourceRequest)(*mcp.ReadResourceResult,error){
			return &mcp.ReadResourceResult{Contents:[]*mcp.ResourceContents{{URI:approvalAppURI,MIMEType:approvalAppMIME,Text:page,
				Meta:mcp.Meta{"ui":map[string]any{"csp":map[string]any{"frameDomains":[]string{origin},"connectDomains":[]string{},"resourceDomains":[]string{}}}}}}},nil
		})
}

// Signed navigation state binds subject, target and expiry, without granting
// authority. A native accept can never call a Broker confirmation endpoint.
type operatorNavigationState struct {
	Version int `json:"v"`
	Kind string `json:"kind"`
	RequestID string `json:"request_id"`
	Target string `json:"target"`
	Subject string `json:"subject"`
	Expires int64 `json:"expires"`
}

func (s *Server) sealNavigation(st operatorNavigationState) string {
	if len(s.navigationKey)!=32 { return "" }
	raw,_:=json.Marshal(st)
	mac:=hmac.New(sha256.New,s.navigationKey);mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(raw)+"."+base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) navigationContinuation(ctx context.Context,req *mcp.CallToolRequest,kind,target string)(map[string]any,bool,error){
	if req==nil || req.Params==nil || len(req.Params.InputResponses)==0 { return nil,false,nil }
	parts:=strings.Split(req.Params.RequestState,".")
	if len(parts)!=2 || len(s.navigationKey)!=32 { return nil,true,errors.New("invalid navigation state") }
	raw,err:=base64.RawURLEncoding.DecodeString(parts[0])
	proof,proofErr:=base64.RawURLEncoding.DecodeString(parts[1])
	mac:=hmac.New(sha256.New,s.navigationKey);mac.Write(raw)
	var st operatorNavigationState
	if err!=nil || proofErr!=nil || !hmac.Equal(proof,mac.Sum(nil)) || json.Unmarshal(raw,&st)!=nil || st.Version!=2 || st.Kind!=kind || st.Target!=filepath.Clean(target) || st.Subject!=SubjectFromContext(ctx) || st.Expires<=time.Now().Unix() {
		return nil,true,errors.New("invalid navigation continuation; no operator decision accepted")
	}
	response,ok:=req.Params.InputResponses[navigationInputKey].(*mcp.ElicitResult)
	if !ok || response==nil || (response.Action!="accept" && response.Action!="decline" && response.Action!="cancel") {
		return nil,true,errors.New("invalid native navigation response")
	}
	args,_:=json.Marshal(map[string]any{"request_id":st.RequestID})
	var out map[string]any
	if err=s.call(ctx,"permissions.approval_status",st.RequestID,"status",args,&out,false,"");err!=nil { return out,true,err }
	if out["resource"]!=target { return nil,true,errors.New("request target does not match this continuation") }
	out["native_action"]=response.Action
	out["approval_channel"]="elicitation"
	out["approval_method"]="operator_navigation"
	out["approval_contract_version"]=1
	out["status_tool"]="permissions.approval_status"
	out["cancel_tool"]="permissions.cancel_approval"
	out["operator_cli"]="python3 scripts/operator-approvals.py --request "+st.RequestID
	out["operator_authentication_required"]=true
	out["message"]="A confirmação do cliente não concede permissões. A decisão autenticada pertence ao operador e ao Broker."
	if link:=operatorApprovalURL(st.RequestID);link!="" { out["operator_approval_url"]=link }
	return out,true,nil
}

func (s *Server) presentApproval(ctx context.Context,req *mcp.CallToolRequest,kind string,out map[string]any)(*mcp.CallToolResult,any,error){
	delete(out,"approval_token")
	out["approval_contract_version"]=1
	out["status_tool"]="permissions.approval_status"
	out["cancel_tool"]="permissions.cancel_approval"
	out["retry_after_seconds"]=3
	out["operator_authentication_required"]=true
	out["operator_approval_guide"]="docs/operator-approval-fallback.md"
	id,_:=out["request_id"].(string)
	out["operator_cli"]="python3 scripts/operator-approvals.py --request "+id
	link:=operatorWebApprovalLink(out)
	var caps *mcp.ClientCapabilities
	if req!=nil { caps=req.ClientCapabilities() }
	support:=approvalCapabilities(caps)
	out["client_capabilities"]=map[string]bool{"mcp_apps":support.Apps,"elicitation_form":support.Form,"elicitation_url":support.URL}
	out["approval_method"]="operator_fallback"
	out["approval_channel"]="cli"
	out["message"]="Pedido pendente. Use o terminal SSH autenticado para decidir. A IA não pode aprovar o próprio pedido."
	if link!="" {
		out["operator_approval_url"]=link
		out["approval_method"]="operator_web"
		out["approval_channel"]="https"
		out["message"]="Confira o pedido na Central HTTPS e confirme ou negue como operador autenticado."
	}
	if support.Apps && appEnabled() && link!="" {
		out["approval_method"]="mcp_apps"
		out["approval_channel"]="mcp_apps"
		return nil,out,nil
	}
	// Elicitation transports intent/navigation only. High-risk requests that the
	// lightweight portal cannot decide stay in the interactive SSH channel.
	if link!="" && (support.URL || support.Form) {
		target,_:=out["root"].(string)
		if kind=="sensitive" { target,_=out["path"].(string) }
		state:=s.sealNavigation(operatorNavigationState{Version:2,Kind:kind,RequestID:id,Target:target,Subject:SubjectFromContext(ctx),Expires:time.Now().Add(10*time.Minute).Unix()})
		if state=="" { return nil,out,nil }
		p:=&mcp.ElicitParams{Mode:"form",Message:"Revise o pedido na Central Pórtico: "+link+" . O aceite do cliente não concede acesso; a decisão exige autenticação do operador.",RequestedSchema:map[string]any{"type":"object"}}
		if support.URL { p.Mode="url";p.URL=link;p.ElicitationID=id;p.RequestedSchema=nil }
		return &mcp.CallToolResult{InputRequests:mcp.InputRequestMap{navigationInputKey:p},RequestState:state},nil,nil
	}
	return nil,out,nil
}

type approvalStatusInput struct { RequestID string `json:"request_id"` }

func registerApprovalStatus(server *mcp.Server,s *Server){
	for _,name:=range []string{"permissions.approval_status","permissions.cancel_approval"} {
		name:=name
		mcp.AddTool(server,annotatedTool(name,"Read or cancel only this authenticated subject's persistent approval request. Never approves a request."),
			func(ctx context.Context,_ *mcp.CallToolRequest,in approvalStatusInput)(*mcp.CallToolResult,map[string]any,error){
				var out map[string]any
				args,_:=json.Marshal(in)
				err:=s.call(ctx,name,in.RequestID,"",args,&out,name=="permissions.cancel_approval","")
				return nil,out,err
			})
	}
}

func approvalServerExtensions() map[string]any {
	if !appEnabled() || operatorApprovalURL("apr_resource_check")=="" { return nil }
	return map[string]any{approvalExtension:map[string]any{"mimeTypes":[]string{approvalAppMIME}}}
}
