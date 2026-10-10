package gateway

import (
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"
)

// Exercise the REAL production Streamable HTTP Handler. No Broker, test_* tool,
// privilege escalation, or production credentials are available to this test.
// The executor counts calls so malformed transport metadata cannot be bypassed.
func TestGateCStatelessHTTPRejectsHeaderSpoofingBeforeExecutor(t *testing.T) {
    probe := &oauthResourceProbeExecutor{}
    server := httptest.NewServer(Handler(probe, AuthConfig{
        Mode: "none", StaticSubject: "disposable-gate-c",
    }))
    defer server.Close()

    validBody := func() map[string]any {
        return map[string]any{
            "jsonrpc": "2.0", "id": 101, "method": "tools/call",
            "params": map[string]any{
                "name": "system.info",
                "arguments": map[string]any{},
                "_meta": map[string]any{
                    "io.modelcontextprotocol/protocolVersion": "2026-07-28",
                    "io.modelcontextprotocol/clientInfo": map[string]string{
                        "name": "gate-c-test", "version": "1",
                    },
                    "io.modelcontextprotocol/clientCapabilities": map[string]any{},
                },
            },
        }
    }
    send := func(body map[string]any, overrides map[string]string) (*http.Response, map[string]json.RawMessage) {
        t.Helper()
        encoded, err := json.Marshal(body)
        if err != nil { t.Fatal(err) }
        request, err := http.NewRequest(http.MethodPost, server.URL+"/mcp", bytes.NewReader(encoded))
        if err != nil { t.Fatal(err) }
        request.Header.Set("Content-Type", "application/json")
        request.Header.Set("Accept", "application/json, text/event-stream")
        request.Header.Set("MCP-Protocol-Version", "2026-07-28")
        request.Header.Set("Mcp-Method", "tools/call")
        request.Header.Set("Mcp-Name", "system.info")
        for k, v := range overrides {
            if v == "" { request.Header.Del(k) } else { request.Header.Set(k, v) }
        }
        response, err := server.Client().Do(request)
        if err != nil { t.Fatal(err) }
        defer response.Body.Close()
        raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
        if err != nil { t.Fatal(err) }
        var decoded map[string]json.RawMessage
        _ = json.Unmarshal(raw, &decoded)
        return response, decoded
    }
    for _, test := range []struct {
        name string
        mutate func(map[string]any)
        headers map[string]string
        code int
    }{
        {
            name: "forged method header cannot route different body",
            headers: map[string]string{"Mcp-Method": "tools/list"},
            code: -32020,
        },
        {
            name: "forged name header cannot route another tool",
            headers: map[string]string{"Mcp-Name": "file.read_test"},
            code: -32020,
        },
        {
            name: "mismatched protocol version metadata",
            headers: map[string]string{"MCP-Protocol-Version": "2025-11-25"},
            code: -32020,
        },
        {
            name: "missing body metadata",
            mutate: func(body map[string]any) {
                delete(body["params"].(map[string]any), "_meta")
            },
        },
    } {
        t.Run(test.name, func(t *testing.T) {
            before, _ := probe.snapshot()
            body := validBody()
            if test.mutate != nil { test.mutate(body) }
            response, rpc := send(body, test.headers)
            if response.StatusCode != http.StatusBadRequest {
                t.Fatalf("HTTP %d: wanted 400 for invalid transport metadata", response.StatusCode)
            }
            if test.code != 0 {
                var decoded struct { Code int `json:"code"` }
                if err := json.Unmarshal(rpc["error"], &decoded); err != nil {
                    t.Fatalf("missing JSON-RPC transport error: %v", err)
                }
                if decoded.Code != test.code {
                    t.Fatalf("JSON-RPC code %d, want %d", decoded.Code, test.code)
                }
            }
            if after, _ := probe.snapshot(); before != after {
                t.Fatal("malformed header reached the executor")
            }
        })
    }

    t.Run("malicious browser origin is denied before executor", func(t *testing.T) {
        before, _ := probe.snapshot()
        response, _ := send(validBody(), map[string]string{"Origin": "https://attacker.invalid"})
        if response.StatusCode != http.StatusForbidden {
            t.Fatalf("invalid Origin returned HTTP %d, expected 403", response.StatusCode)
        }
        if after, _ := probe.snapshot(); before != after {
            t.Fatal("invalid Origin reached the executor")
        }
    })
    t.Run("independent requests need no MCP session", func(t *testing.T) {
        before, _ := probe.snapshot()
        for i := 0; i < 2; i++ {
            response, rpc := send(validBody(), map[string]string{
                "Mcp-Session-Id": "obsolete-legacy-session",
            })
            if response.StatusCode != http.StatusOK || len(rpc["result"]) == 0 {
                t.Fatalf("stateless request %d failed (HTTP %d)", i+1, response.StatusCode)
            }
            if response.Header.Get("Mcp-Session-Id") != "" {
                t.Fatal("2026 stateless Handler minted an obsolete session ID")
            }
        }
        if after, _ := probe.snapshot(); after != before+2 {
            t.Fatalf("expected 2 independent calls, received %d", after-before)
        }
    })
}
