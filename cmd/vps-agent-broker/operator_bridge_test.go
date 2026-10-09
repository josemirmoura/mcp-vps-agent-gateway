package main

import (
 "context"
 "encoding/json"
 "path/filepath"
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
 base := state.Approval{Status:"pending", Subject:"alice", TTL:time.Hour, ExpiresAt:time.Now().Add(time.Minute)}
 root := base
 root.Kind="root";root.Resource="/opt/project";root.Access="work"
 if !eligibleWebApproval(root,"alice") { t.Fatal("valid root approval rejected") }
 if eligibleWebApproval(root,"bob") { t.Fatal("different subject accepted") }
 permanent:=root;permanent.TTL=0
 if eligibleWebApproval(permanent,"alice") {t.Fatal("permanent grant accepted")}
 t.Setenv("VPS_AGENT_PHYSICAL_SCOPE_ROOT","/opt")
 wide:=root;wide.Resource="/opt"
 if eligibleWebApproval(wide,"alice") {t.Fatal("physical ceiling-wide grant accepted")}
 expired:=root;expired.ExpiresAt=time.Now().Add(-time.Second)
 if eligibleWebApproval(expired,"alice") { t.Fatal("expired request accepted") }
 broad:=base;broad.Kind="capability";broad.Capabilities=[]string{"shell.admin"}
 if eligibleWebApproval(broad,"alice") {t.Fatal("elevation capability exposed")}
 sensitive:=base;sensitive.Kind="capability";sensitive.Capabilities=[]string{"sensitive.read:2f6f70742f2e656e76"}
 if !eligibleWebApproval(sensitive,"alice") {t.Fatal("valid protected-file approval rejected")}
 bad:=sensitive;bad.Capabilities=[]string{"sensitive.read:616263"}
 if eligibleWebApproval(bad,"alice") {t.Fatal("relative protected path accepted")}
}

func TestOperatorBridgeSQLiteDenyOneShotAndHiddenElevation(t *testing.T) {
 ctx:=context.Background()
 store,err:=state.Open(filepath.Join(t.TempDir(),"operator.db"))
 if err!=nil {t.Fatal(err)}
 defer store.Close()
 approved,err:=store.CreateRootApproval(ctx,"alice","/opt/test-project","work",time.Hour,10*time.Minute)
 if err!=nil {t.Fatal(err)}
 _,err=store.CreateApproval(ctx,"alice",[]string{"shell.admin"},time.Hour,10*time.Minute)
 if err!=nil {t.Fatal(err)}
 _,err=store.CreateRootApproval(ctx,"bob","/opt/other","read",time.Hour,10*time.Minute)
 if err!=nil {t.Fatal(err)}
 b:=&operatorBridge{broker:&broker.Broker{State:store,AdminToken:"admin-secret",ExpectedSubject:"alice"},token:"web-secret"}
 req:=wire.Request{ID:"list",Tool:"admin.approval.list",AdminToken:"web-secret"}
 list:=b.Handle(ctx,req)
 if !list.OK {t.Fatalf("list denied: %+v",list.Error)}
 var rows []state.Approval
 if err:=json.Unmarshal(list.Result,&rows);err!=nil{t.Fatal(err)}
 if len(rows)!=1||rows[0].ID!=approved.ID {t.Fatalf("web exposed ineligible requests: %+v",rows)}
 body,_:=json.Marshal(map[string]any{"request_id":approved.ID})
 decision:=wire.Request{ID:"deny",Tool:"admin.approval.deny",AdminToken:"web-secret",Args:body}
 result:=b.Handle(ctx,decision)
 if !result.OK{t.Fatalf("denial rejected: %+v",result.Error)}
 dbrow,err:=store.GetApproval(ctx,approved.ID)
 if err!=nil||dbrow.Status!="denied"{t.Fatalf("state not denied: %+v %v",dbrow,err)}
 again:=b.Handle(ctx,decision)
 if again.OK {t.Fatal("second decision must be rejected")}
}
