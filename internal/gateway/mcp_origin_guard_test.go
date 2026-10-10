package gateway

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestGateCOriginAuthorityPreventsDNSRebinding(t *testing.T) {
    // The configured resource is the authority, NOT the request's Host.
    // A remote browser can set Host and Origin to the attacker's DNS name
    // when rebinding it to 127.0.0.1; matching those two fields is unsafe.
    for _, configuration := range []struct {
        name, resource string
    }{
        {name: "public-resource-origin", resource: "https://mcp.example.invalid/mcp"},
        {name: "disposable-loopback", resource: ""},
    } {
        t.Run(configuration.name, func(t *testing.T) {
            probe := &oauthResourceProbeExecutor{}
            server := httptest.NewServer(Handler(probe, AuthConfig{
                Mode: "none", StaticSubject: "test-disposable",
                ResourceIdentifier: configuration.resource,
            }))
            defer server.Close()
            localAuthority := strings.TrimPrefix(server.URL, "http://")
            body := map[string]any{
                "jsonrpc": "2.0", "id": 1, "method": "tools/call",
                "params": map[string]any{
                    "name": "system.info", "arguments": map[string]any{},
                    "_meta": map[string]any{
                        "io.modelcontextprotocol/protocolVersion": "2026-07-28",
                        "io.modelcontextprotocol/clientInfo": map[string]string{
                            "name": "origin-guard-test", "version": "1",
                        },
                        "io.modelcontextprotocol/clientCapabilities": map[string]any{},
                    },
                },
            }
            raw, err := json.Marshal(body)
            if err != nil { t.Fatal(err) }
            send := func(host string, origins []string, want int) {
                t.Helper()
                before, _ := probe.snapshot()
                req, err := http.NewRequest(http.MethodPost, server.URL+"/mcp", bytes.NewReader(raw))
                if err != nil { t.Fatal(err) }
                if host != "" { req.Host = host }
                req.Header.Set("Content-Type", "application/json")
                req.Header.Set("Accept", "application/json, text/event-stream")
                req.Header.Set("MCP-Protocol-Version", "2026-07-28")
                req.Header.Set("Mcp-Method", "tools/call")
                req.Header.Set("Mcp-Name", "system.info")
                for _, origin := range origins { req.Header.Add("Origin", origin) }
                response, err := server.Client().Do(req)
                if err != nil { t.Fatal(err) }
                defer response.Body.Close()
                if response.StatusCode != want {
                    t.Fatalf("HTTP %d; expected %d (host=%s, origins=%v)", response.StatusCode, want, host, origins)
                }
                after, _ := probe.snapshot()
                if want == http.StatusForbidden && before != after {
                    t.Fatal("untrusted Origin reached the real tool executor")
                }
                if want == http.StatusOK && after != before+1 {
                    t.Fatalf("trusted request did not reach the tool executor, before=%d after=%d", before, after)
                }
            }
            validOrigin := "http://" + localAuthority
            if configuration.resource != "" { validOrigin = "https://mcp.example.invalid" }
            send("", nil, http.StatusOK) // backend MCP clients omit Origin
            send("", []string{validOrigin}, http.StatusOK)
            for _, origin := range []string{
                "https://attacker.invalid",
                "http://mcp.example.invalid", // cross-scheme public URL
                "https://mcp.example.invalid/", // Origin must not carry path
                "https://attacker.invalid/path",
                "null",
                "not-a-url",
                "https://user@attacker.invalid",
                "",
                "https://mcp.example.invalid, https://attacker.invalid",
            } {
                // Loopback mode also rejects any non-local or malformed origin.
                send("", []string{origin}, http.StatusForbidden)
            }
            send("", []string{validOrigin, "https://attacker.invalid"}, http.StatusForbidden)
            // DNS rebinding: Host and Origin both appear to come from attacker.
            send("attacker.invalid", []string{"http://attacker.invalid"}, http.StatusForbidden)
            send("attacker.invalid", []string{"https://attacker.invalid"}, http.StatusForbidden)
        })
    }
}

func TestGateCOriginParserAcceptsValidHostnamesWithT(t *testing.T) {
    // Regression: a mistaken string escape can treat the ASCII letter "t"
    // as a forbidden host byte and reject localhost/trusted domains.
    for _, origin := range []string{
        "http://localhost:8080",
        "https://test.example.invalid",
        "https://trusted.example.invalid:8443",
        "http://127.0.0.1:18994",
        "http://[::1]:18994",
    } {
        if got := parseMCPOrigin(origin); got == nil {
            t.Fatalf("valid Origin unexpectedly rejected: %q", origin)
        }
    }
    for _, origin := range []string{
        "null",
        "https://trusted.example.invalid/path",
        "https://trusted.example.invalid?x=1",
        "https://name@trusted.example.invalid",
        "https://test.example.invalid, https://attacker.invalid",
        "https://trusted.example.invalid ",
    } {
        if got := parseMCPOrigin(origin); got != nil {
            t.Fatalf("invalid Origin unexpectedly accepted: %q", origin)
        }
    }
}
