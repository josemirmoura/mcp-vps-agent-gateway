package main

import (
 "context"
 "testing"

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
