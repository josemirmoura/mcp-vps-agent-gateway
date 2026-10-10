package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/gateway"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/ipc"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/policy"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/securefs"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
)

// Exercises one actual 2026 stateless MCP POST over OAuth token introspection,
// Gateway authentication, IPC, local Broker policy and disposable service manager.
// The normal Gateway implementation is used; no test_* method is published.
func TestGateCOAuthStatelessRealBrokerFailClosed(t *testing.T) {
	root := t.TempDir()
	socket := filepath.Join(t.TempDir(), "broker.sock")

	store, err := state.Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer store.Close()
	fs, err := securefs.New([]string{root}, []string{root}, 1<<20)
	if err != nil { t.Fatal(err) }
	services := &fakeServiceManager{}
	cfg := &policy.Config{
		Version: 1,
		Mode: "scoped",
		Filesystem: policy.FilesystemPolicy{Read: []string{root}, Write: []string{root}},
		Services: policy.ResourcePolicy{
			Inspect: []string{"vps-agent-test.service"},
			Manage: []string{"vps-agent-test.service"},
			Actions: []string{"status", "restart"},
		},
		Network: policy.NetworkPolicy{Mode: "blocked"},
		Privilege: policy.PrivilegePolicy{Admin: "broker-only"},
		Replay: policy.ReplayPolicy{RequireIdempotencyForSafeWrites: true},
	}
	if err := cfg.Validate(); err != nil { t.Fatal(err) }
	localBroker := &broker.Broker{Policy:cfg, FS:fs, State:store, Services:services}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- ipc.NewServer(socket, localBroker).Serve(ctx) }()
	brokerClient := ipc.Client{Socket:socket, Timeout:2*time.Second}
	ready := false
	for i:=0; i<100; i++ {
		if _, err := brokerClient.Call(context.Background(), wireRequest(
			"gate-c-broker-health", "system.info", "", "gate-c-disposable")); err == nil {
			ready = true
			break
		}
		time.Sleep(10*time.Millisecond)
	}
	if !ready { t.Fatal("disposable broker unavailable") }

	const issuer = "https://issuer.example.invalid"
	const audience = "gate-c-resource"
	const resource = "https://mcp.example.invalid/mcp"
	const clientID = "only-disposable-gate-c"
	const clientSecret = "test-only-password"
	introspector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		u, pw, ok := req.BasicAuth()
		if !ok || u != clientID || pw != clientSecret || req.Method != http.MethodPost ||
			req.URL.Path != "/introspect" || req.ParseForm() != nil {
			http.Error(w, "invalid introspection request", http.StatusUnauthorized)
			return
		}
		token := req.PostForm.Get("token")
		data := map[string]any{
			"active": token == "valid" || token == "wrong-audience" || token == "wrong-scope",
			"sub": "gate-c-operator",
			"scope": "portico.read",
			"iss": issuer,
			"aud": []string{audience},
			"exp": time.Now().Add(5*time.Minute).Unix(),
		}
		if token == "wrong-audience" { data["aud"] = []string{"other-resource"} }
		if token == "wrong-scope" { data["scope"] = "portico.read.extra" }
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(data)
	}))
	defer introspector.Close()
	verifier, err := gateway.NewIntegratedOIDCVerifier(
		introspector.URL+"/introspect", "", clientID, clientSecret, issuer, audience)
	if err != nil { t.Fatal(err) }

	server := httptest.NewServer(gateway.Handler(
		gateway.BrokerExecutor{Client:brokerClient},
		gateway.AuthConfig{
			Mode:"integrated",
			Verifier:verifier,
			RequiredScopes:[]string{"portico.read"},
			ResourceIdentifier:resource,
			ResourceMetadataURL:"https://mcp.example.invalid/.well-known/oauth-protected-resource",
			AuthorizationServers:[]string{issuer},
		},
	))
	defer server.Close()

	send := func(token, tool, serviceName, origin string, wantHTTP int, wantToolError bool) {
		t.Helper()
		body := map[string]any{
			"jsonrpc":"2.0", "id":7, "method":"tools/call",
			"params":map[string]any{
				"name":tool,
				"arguments":map[string]any{"name":serviceName},
				"_meta":map[string]any{
					"io.modelcontextprotocol/protocolVersion":"2026-07-28",
					"io.modelcontextprotocol/clientInfo":map[string]string{
						"name":"gate-c-unauthorized-boundary", "version":"1",
					},
					"io.modelcontextprotocol/clientCapabilities":map[string]any{},
				},
			},
		}
		b,err:=json.Marshal(body)
		if err!=nil { t.Fatal(err) }
		req,err:=http.NewRequest(http.MethodPost,server.URL+"/mcp",strings.NewReader(string(b)))
		if err!=nil { t.Fatal(err) }
		req.Header.Set("Accept","application/json, text/event-stream")
		req.Header.Set("Content-Type","application/json")
		req.Header.Set("MCP-Protocol-Version","2026-07-28")
		req.Header.Set("Mcp-Method","tools/call")
		req.Header.Set("Mcp-Name",tool)
		if token!="" { req.Header.Set("Authorization","Bearer "+token) }
		if origin!="" { req.Header.Set("Origin",origin) }
		resp,err:=server.Client().Do(req)
		if err!=nil { t.Fatal(err) }
		defer resp.Body.Close()
		if wantHTTP==0 {
			if resp.StatusCode!=http.StatusUnauthorized && resp.StatusCode!=http.StatusForbidden {
				t.Fatalf("insufficient-scope HTTP %d must be 401 or 403",resp.StatusCode)
			}
		} else if resp.StatusCode!=wantHTTP {
			t.Fatalf("tool=%q token category=%q HTTP %d (expected %d)",tool,token,resp.StatusCode,wantHTTP)
		}
		if resp.StatusCode!=http.StatusOK {
			_,_ = io.Copy(io.Discard,io.LimitReader(resp.Body,4096))
			return
		}
		var rpc struct {
			Result struct {IsError bool `json:"isError"`} `json:"result"`
			Error json.RawMessage `json:"error"`
		}
		if err:=json.NewDecoder(io.LimitReader(resp.Body,1<<20)).Decode(&rpc);err!=nil {
			t.Fatal(err)
		}
		if len(rpc.Error)!=0 || rpc.Result.IsError!=wantToolError {
			t.Fatalf("incorrect Broker result for %q: rpcError=%v isError=%v",
				tool,len(rpc.Error)!=0,rpc.Result.IsError)
		}
	}

	// Invalid authorization cannot reach privileged local IPC operations.
	before := services.RestartCount()
	send("","service.restart","vps-agent-test.service","",http.StatusUnauthorized,false)
	send("inactive","service.restart","vps-agent-test.service","",http.StatusUnauthorized,false)
	send("wrong-audience","service.restart","vps-agent-test.service","",http.StatusUnauthorized,false)
	send("wrong-scope","service.restart","vps-agent-test.service","",0,false)
	send("valid","service.restart","vps-agent-test.service",
		"https://attacker.invalid",http.StatusForbidden,false)
	if services.RestartCount()!=before { t.Fatal("unauthorized request restarted a service") }

	// Authorized OAuth subject can only perform Broker-allowed read.
	send("valid","service.status","vps-agent-test.service","",http.StatusOK,false)

	// Even a valid bearer cannot turn an out-of-policy service into authority.
	send("valid","service.restart","postgres.service","",http.StatusOK,true)
	if services.RestartCount()!=before { t.Fatal("out-of-policy service restarted") }

	cancel()
	select {
	case err:= <-done:
		if err!=nil { t.Fatalf("disposable Broker exit: %v",err) }
	case <-time.After(2*time.Second):
		t.Fatal("disposable Broker failed to shut down")
	}
}
