package main

import (
 "context"
 "testing"
 "time"

 "github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"

 "github.com/josemirmoura/mcp-vps-agent-gateway/internal/broker"
 "github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)
func TestOperatorBridgeDeniesNonApprovalTools(t *testing.T) {
 b := &operatorBridge{broker: &broker.Broker{AdminToken:"root-admin-token"}, token:"scoped-operator-token"}
 for _, name := range []string{"system.info","admin.revoke_all","admin.audit.tail","permissions.request_root_access","shell.exec"} {
  response := b.Handle(context.Background(),wire.Request{ID:"t",Tool:name,AdminToken:"scoped-operator-token"})
  if response.OK {t.Fatalf("tool %q should be denied",name)}
 }
}
func TestOperatorBridgeRejectsMissingOrWrongToken(t *testing.T) {
 b:=&operatorBridge{broker:&broker.Broker{AdminToken:"root-admin-token"},token:"scoped-token"}
 for _,token:=range []string{"","scoped","root-admin-token"} {
  response:=b.Handle(context.Background(),wire.Request{ID:"t",Tool:"admin.approval.list",AdminToken:token})
  if response.OK {t.Fatal("invalid token was accepted")}
 }
}

func TestWebApprovalEligibilityRejectsElevationAndExpired(t *testing.T) {
 base := state.Approval{Status:"pending", Subject:"alice", ExpiresAt:time.Now().Add(time.Minute)}
 root := base
 root.Kind="root";root.Resource="/opt/project";root.Access="work"
 if !eligibleWebApproval(root,"alice") { t.Fatal("valid root approval rejected") }
 if eligibleWebApproval(root,"bob") { t.Fatal("different subject accepted") }
 expired:=root;expired.ExpiresAt=time.Now().Add(-time.Second)
 if eligibleWebApproval(expired,"alice") { t.Fatal("expired request accepted") }
 broad:=base;broad.Kind="capability";broad.Capabilities=[]string{"shell.admin"}
 if eligibleWebApproval(broad,"alice") {t.Fatal("elevation capability exposed")}
 sensitive:=base;sensitive.Kind="capability";sensitive.Capabilities=[]string{"sensitive.read:2f6f70742f2e656e76"}
 if !eligibleWebApproval(sensitive,"alice") {t.Fatal("valid protected-file approval rejected")}
 bad:=sensitive;bad.Capabilities=[]string{"sensitive.read:616263"}
 if eligibleWebApproval(bad,"alice") {t.Fatal("relative protected path accepted")}
}
