package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

// ApprovalFingerprint binds the displayed immutable scope to this destination.
// It is an integrity identifier, never an authentication credential.
func ApprovalFingerprint(a Approval, instanceID string) string {
	raw, _ := json.Marshal([]any{instanceID, a.ID, a.Subject, a.Kind, a.Resource,
		a.Access, a.Capabilities, a.TTL.Nanoseconds(), a.CreatedAt.UnixNano(), a.ExpiresAt.UnixNano()})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func approvalGrantID(id string) string {
	sum := sha256.Sum256([]byte("portico/approval-grant/v1/" + id))
	return "gr_" + hex.EncodeToString(sum[:16])
}

type ApprovalResolution struct {
	Approval Approval
	Root *RootDelegation
	Grant *Grant
}

// ResolveApproval commits the decision, grant and audit together. A crash or
// failed audit INSERT cannot leave an approved request without its grant.
// No external operation is performed while the transaction is held.
func (s *Store) ResolveApproval(ctx context.Context, expected Approval, decision string, ev AuditEvent) (ApprovalResolution, error) {
	var out ApprovalResolution
	if decision != "approved" && decision != "denied" && decision != "cancelled" {
		return out, errors.New("invalid approval decision")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return out, err }
	defer tx.Rollback()
	var a Approval
	var caps string
	var resource, access sql.NullString
	var ttl, created, expires int64
	err = tx.QueryRowContext(ctx, `SELECT request_id,subject,capabilities,ttl_ns,status,created_at,expires_at,kind,resource,access FROM approvals WHERE request_id=?`, expected.ID).
		Scan(&a.ID, &a.Subject, &caps, &ttl, &a.Status, &created, &expires, &a.Kind, &resource, &access)
	if err != nil { return out, err }
	if err = json.Unmarshal([]byte(caps), &a.Capabilities); err != nil { return out, err }
	a.TTL, a.CreatedAt, a.ExpiresAt = time.Duration(ttl), time.Unix(0, created), time.Unix(0, expires)
	a.Resource, a.Access = resource.String, access.String
	now := time.Now()
	if a.Status != "pending" || !now.Before(a.ExpiresAt) { return out, errors.New("request expired or already decided") }
	if ApprovalFingerprint(a, ev.InstanceID) != ApprovalFingerprint(expected, ev.InstanceID) { return out, errors.New("request scope changed") }
	res, err := tx.ExecContext(ctx, `UPDATE approvals SET status=?,decided_at=? WHERE request_id=? AND status='pending' AND expires_at>?`, decision, now.UnixNano(), a.ID, now.UnixNano())
	if err != nil { return out, err }
	if n, err := res.RowsAffected(); err != nil || n != 1 { return out, errors.New("request no longer pending") }
	a.Status = decision
	out.Approval = a
	if decision == "approved" && a.Kind == "root" {
		id, err := randomID("root_")
		if err != nil { return out, err }
		d := RootDelegation{ID:id, Subject:a.Subject, Root:a.Resource, Access:a.Access, ApprovalID:a.ID, CreatedAt:now}
		var exp any
		if a.TTL > 0 { stamp:=now.Add(a.TTL); d.ExpiresAt=&stamp; exp=stamp.UnixNano() }
		if _, err = normalizeRootAccess(a.Access); err != nil { return out, err }
		if _, err = tx.ExecContext(ctx, `UPDATE root_delegations SET revoked_at=? WHERE subject=? AND root=? AND revoked_at IS NULL`, now.UnixNano(), a.Subject, a.Resource); err != nil { return out, err }
		if _, err = tx.ExecContext(ctx, `INSERT INTO root_delegations(delegation_id,subject,root,access,approval_id,created_at,expires_at) VALUES(?,?,?,?,?,?,?)`, id,a.Subject,a.Resource,a.Access,a.ID,now.UnixNano(),exp); err != nil { return out, err }
		out.Root=&d
		ev.GrantID=id
	} else if decision == "approved" {
		if a.Kind != "capability" || a.TTL <= 0 || len(a.Capabilities)==0 { return out, errors.New("invalid capability grant") }
		g:=Grant{ID:approvalGrantID(a.ID), Subject:a.Subject, Capabilities:a.Capabilities, ExpiresAt:now.Add(a.TTL)}
		if _, err = tx.ExecContext(ctx, `INSERT INTO grants(grant_id,subject,capabilities,expires_at) VALUES(?,?,?,?)`, g.ID,g.Subject,caps,g.ExpiresAt.UnixNano()); err != nil { return out, err }
		out.Grant=&g
		ev.GrantID=g.ID
	}
	ev.Time=now.UTC().Format(time.RFC3339Nano)
	ev.ApprovalID, ev.Requester, ev.Decision = a.ID, a.Subject, decision
	raw, err := json.Marshal(ev)
	if err != nil { return out, err }
	var seq int64
	var prev string
	err = tx.QueryRowContext(ctx, `SELECT seq,hash FROM audit_events ORDER BY seq DESC LIMIT 1`).Scan(&seq,&prev)
	if errors.Is(err,sql.ErrNoRows) { seq,prev=0,"GENESIS" } else if err!=nil { return out,err }
	seq++
	if _,err=tx.ExecContext(ctx,`INSERT INTO audit_events(seq,event_json,prev_hash,hash) VALUES(?,?,?,?)`,seq,string(raw),prev,auditDigest(seq,prev,string(raw))); err!=nil { return out,err }
	if err=tx.Commit(); err!=nil { return out,err }
	return out,nil
}

// ApprovalAuthorityStatus distinguishes a historical decision from a currently
// usable grant. It is informational; subsequent operations reauthorize.
func (s *Store) ApprovalAuthorityStatus(ctx context.Context, a Approval) (string,error) {
	if a.Status=="pending" && !time.Now().Before(a.ExpiresAt) { return "expired",nil }
	if a.Status!="approved" { return a.Status,nil }
	if a.Kind=="root" {
		var expires, revoked sql.NullInt64
		err:=s.db.QueryRowContext(ctx,`SELECT expires_at,revoked_at FROM root_delegations WHERE approval_id=? ORDER BY created_at DESC LIMIT 1`,a.ID).Scan(&expires,&revoked)
		if err!=nil { return "unavailable",err }
		if revoked.Valid { return "revoked",nil }
		if expires.Valid && expires.Int64<=time.Now().UnixNano() { return "expired",nil }
		return "approved",nil
	}
	g,err:=s.GetGrant(ctx,approvalGrantID(a.ID))
	if err!=nil { return "unavailable",err }
	if g.Revoked { return "revoked",nil }
	if !time.Now().Before(g.ExpiresAt) { return "expired",nil }
	return "approved",nil
}
