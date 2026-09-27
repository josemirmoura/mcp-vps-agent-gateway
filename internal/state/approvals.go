package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type Approval struct {
	ID           string
	Subject      string
	Capabilities []string
	TTL          time.Duration
	Status       string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

func (s *Store) CreateApproval(ctx context.Context, subject string, capabilities []string, ttl, requestLifetime time.Duration) (Approval, error) {
	if subject == "" || ttl <= 0 || requestLifetime <= 0 {
		return Approval{}, errors.New("subject and positive durations are required")
	}
	caps := canonicalCaps(capabilities)
	if len(caps) == 0 {
		return Approval{}, errors.New("capabilities are required")
	}
	id, err := randomID("apr_")
	if err != nil {
		return Approval{}, err
	}
	now := time.Now()
	exp := now.Add(requestLifetime)
	raw, _ := json.Marshal(caps)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO approvals(request_id,subject,capabilities,ttl_ns,status,created_at,expires_at)
		 VALUES(?,?,?,?, 'pending', ?, ?)`,
		id, subject, string(raw), ttl.Nanoseconds(), now.UnixNano(), exp.UnixNano())
	if err != nil {
		return Approval{}, err
	}
	return Approval{ID: id, Subject: subject, Capabilities: caps, TTL: ttl, Status: "pending", CreatedAt: now, ExpiresAt: exp}, nil
}

func (s *Store) GetApproval(ctx context.Context, id string) (Approval, error) {
	var subject, rawCaps, status string
	var ttlNS, created, expires int64
	err := s.db.QueryRowContext(ctx,
		`SELECT subject,capabilities,ttl_ns,status,created_at,expires_at FROM approvals WHERE request_id=?`, id).
		Scan(&subject, &rawCaps, &ttlNS, &status, &created, &expires)
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

	var subject, rawCaps, status string
	var ttlNS, created, expires int64
	err = tx.QueryRowContext(ctx,
		`SELECT subject,capabilities,ttl_ns,status,created_at,expires_at FROM approvals WHERE request_id=?`, id).
		Scan(&subject, &rawCaps, &ttlNS, &status, &created, &expires)
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
	}, nil
}

func (s *Store) ListPendingApprovals(ctx context.Context) ([]Approval, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT request_id,subject,capabilities,ttl_ns,status,created_at,expires_at
		  FROM approvals WHERE status='pending' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Approval
	for rows.Next() {
		var a Approval
		var rawCaps string
		var ttlNS, created, expires int64
		if err := rows.Scan(&a.ID, &a.Subject, &rawCaps, &ttlNS, &a.Status, &created, &expires); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(rawCaps), &a.Capabilities); err != nil {
			return nil, err
		}
		a.TTL = time.Duration(ttlNS)
		a.CreatedAt = time.Unix(0, created)
		a.ExpiresAt = time.Unix(0, expires)
		out = append(out, a)
	}
	return out, rows.Err()
}

var _ = sql.ErrNoRows
