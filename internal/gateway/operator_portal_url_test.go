package gateway

import (
 "strings"
 "testing"
)
func TestOperatorApprovalURL(t *testing.T) {
 t.Setenv("VPS_AGENT_OPERATOR_PORTAL_URL","https://operator.example.test/operator")
 got:=operatorApprovalURL("apr_12345678")
 if got!="https://operator.example.test/operator?request=apr_12345678" {t.Fatalf("unexpected URL: %q",got)}
}
func TestOperatorApprovalURLRejectsUnsafeInputs(t *testing.T) {
 for _, root := range []string{"", "http://example.test/operator", "https://evil.test/@example.org?x=1","https://user:password@example.test/operator","javascript:alert(1)"} {
  t.Setenv("VPS_AGENT_OPERATOR_PORTAL_URL",root)
  if got:=operatorApprovalURL("apr_12345678");got!="" {t.Fatalf("accepted unsafe URL %q -> %q",root,got)}
 }
 t.Setenv("VPS_AGENT_OPERATOR_PORTAL_URL","https://operator.example.test/operator")
 if got:=operatorApprovalURL("apr_12345678%0a");got!="" {t.Fatal("accepted unsafe request id")}
 if got:=operatorApprovalURL(strings.Repeat("a",300));got!="" {t.Fatal("accepted long request id")}
}

func TestOperatorWebApprovalLinkBounds(t *testing.T){
 t.Setenv("VPS_AGENT_OPERATOR_PORTAL_URL","https://operator.example.test/operator")
 base:=map[string]any{"request_id":"apr_12345678","delegation_ttl_seconds":float64(900)}
 if got:=operatorWebApprovalLink(base); got=="" {t.Fatal("temporary web link rejected")}
 for _,change:=range []map[string]any{
  {"request_id":"apr_12345678","delegation_ttl_seconds":float64(0)},
  {"request_id":"apr_12345678","delegation_ttl_seconds":float64(-1)},
  {"request_id":"apr_12345678","delegation_ttl_seconds":float64(900),"ceiling_wide":true},
  {"request_id":"apr_12345678","delegation_ttl_seconds":"invalid"},
 } {
  if got:=operatorWebApprovalLink(change);got!=""{t.Fatalf("unsafe web approval link offered: %s",got)}
 }
}
