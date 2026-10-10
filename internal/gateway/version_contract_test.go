package gateway

import (
    "context"
    "encoding/json"
    "net/http/httptest"
    "strings"
    "testing"

    portico "github.com/josemirmoura/mcp-vps-agent-gateway"
    "github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

type versionContractExecutor struct{}

func (versionContractExecutor) Call(_ context.Context, req wire.Request) (wire.Response, error) {
    if req.Tool != "system.info" {
        return wire.ErrorResponse(req.ID, "unexpected_tool", req.Tool), nil
    }
    // The Broker is deliberately unaware of the Gateway's packaged release.
    // The Gateway injects its own version and must not trust an upstream value.
    raw, _ := json.Marshal(map[string]any{
        "hostname": "version-smoke",
        "gateway_version": "v9.9.9-spoofed",
    })
    return wire.Response{ID:req.ID, OK:true, Result:raw}, nil
}

func TestMCPInitializeAndSystemInfoAdvertiseSamePackagedRelease(t *testing.T) {
    ts := httptest.NewServer(Handler(versionContractExecutor{}, AuthConfig{
        Mode:"none", StaticSubject:"version-contract-client",
    }))
    defer ts.Close()

    client := mcp.NewClient(&mcp.Implementation{
        Name:"version-contract-test", Version:"v0.0.0",
    }, nil)
    ctx := context.Background()
    session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint:ts.URL+"/mcp"}, nil)
    if err != nil { t.Fatal(err) }
    defer session.Close()

    init := session.InitializeResult()
    if init == nil || init.ServerInfo == nil {
        t.Fatal("MCP initialize did not include serverInfo")
    }
    if init.ServerInfo.Name != "portico-mcp" {
        t.Fatalf("unexpected server identity: %+v", init.ServerInfo)
    }
    if got,want:=init.ServerInfo.Version,portico.Version();got != want {
        t.Fatalf("MCP handshake announced %q but embedded VERSION is %q",got,want)
    }
    if strings.Contains(init.ServerInfo.Version,"v9.9.9-spoofed") {
        t.Fatal("server version must not come from Broker or MCP client")
    }

    result,err:=session.CallTool(ctx,&mcp.CallToolParams{
        Name:"system.info", Arguments:map[string]any{},
    })
    if err!=nil || result.IsError {
        t.Fatalf("system.info failed: err=%v result=%+v",err,result)
    }
    payload,err:=json.Marshal(result.StructuredContent)
    if err!=nil{t.Fatal(err)}
    var info struct {
        GatewayVersion string `json:"gateway_version"`
    }
    if err:=json.Unmarshal(payload,&info);err!=nil{t.Fatal(err)}
    if info.GatewayVersion != init.ServerInfo.Version {
        t.Fatalf("MCP serverInfo.version=%q, system.info.gateway_version=%q",init.ServerInfo.Version,info.GatewayVersion)
    }
    if info.GatewayVersion=="v9.9.9-spoofed"{
        t.Fatal("system.info must not trust Broker-supplied gateway_version")
    }
}
