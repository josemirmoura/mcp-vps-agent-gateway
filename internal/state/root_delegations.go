package state

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type RootDelegation struct {
	ID         string     `json:"delegation_id"`
	Subject    string     `json:"subject"`
	Root       string     `json:"root"`
	Access     string     `json:"access"`
	ApprovalID string     `json:"approval_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

func normalizeRootAccess(access string) (string, error) {
	access = strings.ToLower(strings.TrimSpace(access))
	switch access {
	case "read", "work", "compose":
		return access, nil
	default:
		return "", errors.New("root access must be read, work, or compose")
	}
}

func (s *Store) IssueRootDelegation(ctx context.Context, subject, root, access, approvalID string, ttl time.Duration) (RootDelegation, error) {
	if subject == "" || root == "" {
		return RootDelegation{}, errors.New("subject and root are required")
	}
	var err error
	access, err = normalizeRootAccess(access)
	if err != nil {
		return RootDelegation{}, err
	}
	if ttl < 0 {
		return RootDelegation{}, errors.New("ttl cannot be negative")
	}
	id, err := randomID("root_")
	if err != nil {
		return RootDelegation{}, err
	}
	now := time.Now()
	var expires any
	var expiresAt *time.Time
	if ttl > 0 {
		exp := now.Add(ttl)
		expires = exp.UnixNano()
		expiresAt = &exp
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RootDelegation{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE root_delegations SET revoked_at=?
		  WHERE subject=? AND root=? AND revoked_at IS NULL`,
		now.UnixNano(), subject, root); err != nil {
		return RootDelegation{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO root_delegations(delegation_id,subject,root,access,approval_id,created_at,expires_at)
		  VALUES(?,?,?,?,?,?,?)`,
		id, subject, root, access, approvalID, now.UnixNano(), expires); err != nil {
		return RootDelegation{}, err
	}
	if err := tx.Commit(); err != nil {
		return RootDelegation{}, err
	}
	return RootDelegation{
		ID: id, Subject: subject, Root: root, Access: access, ApprovalID: approvalID,
		CreatedAt: now, ExpiresAt: expiresAt,
	}, nil
}

func (s *Store) ListActiveRootDelegations(ctx context.Context, subject string) ([]RootDelegation, error) {
	if subject == "" {
		return nil, errors.New("subject is required")
	}
	now := time.Now().UnixNano()
	rows, err := s.db.QueryContext(ctx,
		`SELECT delegation_id,root,access,approval_id,created_at,expires_at
		  FROM root_delegations
		  WHERE subject=? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at>?)
		  ORDER BY root, created_at`,
		subject, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RootDelegation
	for rows.Next() {
		var d RootDelegation
		var approval sql.NullString
		var created int64
		var expires sql.NullInt64
		if err := rows.Scan(&d.ID, &d.Root, &d.Access, &approval, &created, &expires); err != nil {
			return nil, err
		}
		d.Subject = subject
		d.ApprovalID = approval.String
		d.CreatedAt = time.Unix(0, created)
		if expires.Valid {
			exp := time.Unix(0, expires.Int64)
			d.ExpiresAt = &exp
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) RevokeRootDelegation(ctx context.Context, subject, root string) (int64, error) {
	if subject == "" || root == "" {
		return 0, errors.New("subject and root are required")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE root_delegations SET revoked_at=?
		  WHERE subject=? AND root=? AND revoked_at IS NULL`,
		time.Now().UnixNano(), subject, root)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
