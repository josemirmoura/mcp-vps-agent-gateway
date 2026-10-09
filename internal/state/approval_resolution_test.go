package state

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestApprovalResolutionAtomicGrantAndAudit(t *testing.T) {
	ctx:=context.Background()
	s,err:=Open(filepath.Join(t.TempDir(),"state.db"));if err!=nil {t.Fatal(err)};defer s.Close()
	a,err:=s.CreateRootApproval(ctx,"alice","/opt/project","read",time.Minute,time.Minute);if err!=nil {t.Fatal(err)}
	// A failed audit write must roll back both the grant and decision.
	_,err=s.db.Exec(`CREATE TRIGGER reject_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'synthetic audit failure'); END`);if err!=nil{t.Fatal(err)}
	if _,err=s.ResolveApproval(ctx,a,"approved",AuditEvent{InstanceID:"node-a",Subject:"owner",Tool:"admin.approval.approve"});err==nil{t.Fatal("audit failure did not block grant")}
	row,_:=s.GetApproval(ctx,a.ID);roots,_:=s.ListActiveRootDelegations(ctx,"alice")
	if row.Status!="pending"||len(roots)!=0{t.Fatal("partial decision/grant escaped transaction")}
	if _,err=s.db.Exec(`DROP TRIGGER reject_audit`);err!=nil{t.Fatal(err)}
	result,err:=s.ResolveApproval(ctx,a,"approved",AuditEvent{InstanceID:"node-a",Subject:"owner",Tool:"admin.approval.approve"});if err!=nil{t.Fatal(err)}
	if result.Root==nil||result.Approval.Status!="approved"{t.Fatal("grant missing")}
	if status,err:=s.ApprovalAuthorityStatus(ctx,aWithStatus(a,"approved"));err!=nil||status!="approved"{t.Fatalf("status=%s err=%v",status,err)}
	if _,err=s.RevokeRootDelegation(ctx,"alice",a.Resource);err!=nil{t.Fatal(err)}
	if status,err:=s.ApprovalAuthorityStatus(ctx,aWithStatus(a,"approved"));err!=nil||status!="revoked"{t.Fatalf("revoked status=%s err=%v",status,err)}
	records,err:=s.ListAuditAfter(ctx,0,10);if err!=nil{t.Fatal(err)}
	if len(records)!=1||records[0].Event.ApprovalID!=a.ID||records[0].Event.Requester!="alice"||records[0].Event.Subject!="owner"{t.Fatalf("decision provenance missing: %+v",records)}
}

func aWithStatus(a Approval,status string) Approval { a.Status=status;return a }

func TestApprovalResolutionConcurrencyReplayExpiryAndSnapshot(t *testing.T){
	ctx:=context.Background();s,err:=Open(":memory:");if err!=nil{t.Fatal(err)};defer s.Close()
	a,err:=s.CreateRootApproval(ctx,"alice","/opt/project","read",time.Minute,time.Minute);if err!=nil{t.Fatal(err)}
	swapped:=a;swapped.Subject="bob"
	if _,err=s.ResolveApproval(ctx,swapped,"approved",AuditEvent{});err==nil{t.Fatal("swapped requester accepted")}
	if ApprovalFingerprint(a,"node-a")==ApprovalFingerprint(a,"node-b"){t.Fatal("fingerprint did not bind destination")}
	var winners atomic.Int32;var wg sync.WaitGroup
	for i:=0;i<12;i++{wg.Add(1);go func(i int){defer wg.Done();decision:="approved";if i%2==0{decision="denied"};if _,err:=s.ResolveApproval(ctx,a,decision,AuditEvent{Subject:"owner"});err==nil{winners.Add(1)}}(i)}
	wg.Wait();if winners.Load()!=1{t.Fatalf("%d concurrent decisions committed",winners.Load())}
	rows,_:=s.ListActiveRootDelegations(ctx,"alice");if len(rows)>1{t.Fatal("duplicate grant")}
	expired,err:=s.CreateRootApproval(ctx,"alice","/opt/other","read",time.Minute,time.Minute);if err!=nil{t.Fatal(err)}
	if _,err=s.db.Exec(`UPDATE approvals SET expires_at=? WHERE request_id=?`,time.Now().Add(-time.Second).UnixNano(),expired.ID);err!=nil{t.Fatal(err)}
	if _,err=s.ResolveApproval(ctx,expired,"approved",AuditEvent{});err==nil{t.Fatal("expired request accepted")}
}

func TestCapabilityDecisionStatusRevocationAndCancellation(t *testing.T){
	ctx:=context.Background();s,err:=Open(":memory:");if err!=nil{t.Fatal(err)};defer s.Close()
	a,err:=s.CreateApproval(ctx,"alice",[]string{"sensitive.read:2f6f70742f702f2e656e76"},time.Minute,time.Minute);if err!=nil{t.Fatal(err)}
	r,err:=s.ResolveApproval(ctx,a,"approved",AuditEvent{Subject:"owner"});if err!=nil{t.Fatal(err)}
	if r.Grant==nil{t.Fatal("capability grant missing")}
	if _,err=s.RevokeGrantsByCapability(ctx,"alice",a.Capabilities...);err!=nil{t.Fatal(err)}
	if status,err:=s.ApprovalAuthorityStatus(ctx,r.Approval);err!=nil||status!="revoked"{t.Fatalf("status=%s err=%v",status,err)}
	b,err:=s.CreateRootApproval(ctx,"alice","/opt/next","read",time.Minute,time.Minute);if err!=nil{t.Fatal(err)}
	r,err=s.ResolveApproval(ctx,b,"cancelled",AuditEvent{Subject:"alice"});if err!=nil||r.Root!=nil||r.Grant!=nil{t.Fatalf("cancel failed: %+v %v",r,err)}
}
