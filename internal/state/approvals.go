package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

const MaxPendingApprovalsPerSubject = 64
const MaxPendingApprovals = 256

var ErrApprovalQueueFull = errors.New("pending approval capacity reached; consult or cancel existing requests")

type Approval struct {
	Fingerprint  string `json:"Fingerprint,omitempty"`
	NodeID       string `json:"NodeID,omitempty"`
	ID           string
	Subject      string
	Capabilities []string
	TTL          time.Duration
	Status       string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	Kind         string
	Resource     string
	Access       string
}

func (s *Store) CreateApproval(ctx context.Context, subject string, capabilities []string, ttl, requestLifetime time.Duration) (Approval, error) {
	if ttl <= 0 {
		return Approval{}, errors.New("positive ttl is required")
	}
	return s.createApproval(ctx, subject, capabilities, ttl, requestLifetime, "capability", "", "")
}

func (s *Store) CreateRootApproval(ctx context.Context, subject, root, access string, ttl, requestLifetime time.Duration) (Approval, error) {
	if root == "" || access == "" {
		return Approval{}, errors.New("root and access are required")
	}
	if ttl < 0 {
		return Approval{}, errors.New("ttl cannot be negative")
	}
	return s.createApproval(ctx, subject, nil, ttl, requestLifetime, "root", root, access)
}

func (s *Store) createApproval(ctx context.Context, subject string, capabilities []string, ttl, requestLifetime time.Duration, kind, resource, access string) (Approval, error) {
	if subject == "" || requestLifetime <= 0 {
		return Approval{}, errors.New("subject and positive request lifetime are required")
	}
	caps := canonicalCaps(capabilities)
	if kind == "capability" && len(caps) == 0 {
		return Approval{}, errors.New("capabilities are required")
	}
	if kind != "capability" && kind != "root" {
		return Approval{}, errors.New("unsupported approval kind")
	}
	id, err := randomID("apr_")
	if err != nil {
		return Approval{}, err
	}
	now := time.Now()
	exp := now.Add(requestLifetime)
	raw, _ := json.Marshal(caps)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO approvals(request_id,subject,capabilities,ttl_ns,status,created_at,expires_at,kind,resource,access)
		 SELECT ?,?,?,?, 'pending', ?, ?, ?, ?, ?
		 WHERE (SELECT COUNT(*) FROM approvals WHERE status='pending' AND expires_at>? AND subject=?) < ?
		 AND (SELECT COUNT(*) FROM approvals WHERE status='pending' AND expires_at>?) < ?`,
		id, subject, string(raw), ttl.Nanoseconds(), now.UnixNano(), exp.UnixNano(), kind, resource, access,
		now.UnixNano(), subject, MaxPendingApprovalsPerSubject, now.UnixNano(), MaxPendingApprovals)
	if err != nil {
		return Approval{}, err
	}
	if count, err := res.RowsAffected(); err != nil {
		return Approval{}, err
	} else if count != 1 {
		return Approval{}, ErrApprovalQueueFull
	}
	return Approval{
		ID: id, Subject: subject, Capabilities: caps, TTL: ttl, Status: "pending",
		CreatedAt: now, ExpiresAt: exp, Kind: kind, Resource: resource, Access: access,
	}, nil
}

func (s *Store) GetApproval(ctx context.Context, id string) (Approval, error) {
	var subject, rawCaps, status, kind string
	var resource, access sql.NullString
	var ttlNS, created, expires int64
	err := s.db.QueryRowContext(ctx,
		`SELECT subject,capabilities,ttl_ns,status,created_at,expires_at,kind,resource,access FROM approvals WHERE request_id=?`, id).
		Scan(&subject, &rawCaps, &ttlNS, &status, &created, &expires, &kind, &resource, &access)
	if err != nil {
		return Approval{}, err
	}
	var caps []string
	if err := json.Unmarshal([]byte(rawCaps), &caps); err != nil {
		return Approval{}, err
	}
	return Approval{
		ID: id, Subject: subject, Capabilities: caps, TTL: time.Duration(ttlNS),
		Status: status, CreatedAt: time.Unix(0, created), ExpiresAt: time.Unix(0, expires),
		Kind: kind, Resource: resource.String, Access: access.String,
	}, nil
}

func (s *Store) DecideApproval(ctx context.Context, id, decision string) (Approval, error) {
	if decision != "approved" && decision != "denied" {
		return Approval{}, errors.New("decision must be approved or denied")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Approval{}, err
	}
	defer tx.Rollback()

	var subject, rawCaps, status, kind string
	var resource, access sql.NullString
	var ttlNS, created, expires int64
	err = tx.QueryRowContext(ctx,
		`SELECT subject,capabilities,ttl_ns,status,created_at,expires_at,kind,resource,access FROM approvals WHERE request_id=?`, id).
		Scan(&subject, &rawCaps, &ttlNS, &status, &created, &expires, &kind, &resource, &access)
	if err != nil {
		return Approval{}, err
	}
	if status != "pending" {
		return Approval{}, errors.New("approval request is not pending")
	}
	now := time.Now()
	if now.UnixNano() >= expires {
		if _, err := tx.ExecContext(ctx, `UPDATE approvals SET status='expired',decided_at=? WHERE request_id=?`, now.UnixNano(), id); err != nil {
			return Approval{}, err
		}
		if err := tx.Commit(); err != nil {
			return Approval{}, err
		}
		return Approval{}, errors.New("approval request expired")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE approvals SET status=?,decided_at=? WHERE request_id=?`, decision, now.UnixNano(), id); err != nil {
		return Approval{}, err
	}
	if err := tx.Commit(); err != nil {
		return Approval{}, err
	}
	var caps []string
	if err := json.Unmarshal([]byte(rawCaps), &caps); err != nil {
		return Approval{}, err
	}
	return Approval{
		ID: id, Subject: subject, Capabilities: caps, TTL: time.Duration(ttlNS),
		Status: decision, CreatedAt: time.Unix(0, created), ExpiresAt: time.Unix(0, expires),
		Kind: kind, Resource: resource.String, Access: access.String,
	}, nil
}

func (s *Store) ListPendingApprovals(ctx context.Context) ([]Approval, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT request_id,subject,capabilities,ttl_ns,status,created_at,expires_at,kind,resource,access
		  FROM approvals WHERE status='pending' AND expires_at>? ORDER BY created_at`, time.Now().UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Approval
	for rows.Next() {
		var a Approval
		var rawCaps string
		var resource, access sql.NullString
		var ttlNS, created, expires int64
		if err := rows.Scan(&a.ID, &a.Subject, &rawCaps, &ttlNS, &a.Status, &created, &expires, &a.Kind, &resource, &access); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(rawCaps), &a.Capabilities); err != nil {
			return nil, err
		}
		a.TTL = time.Duration(ttlNS)
		a.CreatedAt = time.Unix(0, created)
		a.ExpiresAt = time.Unix(0, expires)
		a.Resource = resource.String
		a.Access = access.String
		out = append(out, a)
	}
	return out, rows.Err()
}

var _ = sql.ErrNoRows
