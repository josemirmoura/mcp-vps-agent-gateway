package state

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(filename string) (*Store, error) {
	if filename != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(filename), 0o750); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("%s: %w", pragma, err)
		}
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS operations (
			invocation_id TEXT PRIMARY KEY,
			subject TEXT NOT NULL,
			tool TEXT NOT NULL,
			request_hash TEXT NOT NULL,
			state TEXT NOT NULL,
			response BLOB,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS grants (
			grant_id TEXT PRIMARY KEY,
			subject TEXT NOT NULL,
			capabilities TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			revoked_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS lock_generation (
			resource TEXT PRIMARY KEY,
			token INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS resource_locks (
			resource TEXT PRIMARY KEY,
			owner TEXT NOT NULL,
			token INTEGER NOT NULL,
			expires_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS approvals (
			request_id TEXT PRIMARY KEY,
			subject TEXT NOT NULL,
			capabilities TEXT NOT NULL,
			ttl_ns INTEGER NOT NULL,
			status TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			decided_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS jobs (
			job_id TEXT PRIMARY KEY,
			subject TEXT NOT NULL,
			tool TEXT NOT NULL,
			resource TEXT,
			unit_name TEXT,
			grant_id TEXT,
			state TEXT NOT NULL,
			deadline INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			exit_code INTEGER,
			detail TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			seq INTEGER PRIMARY KEY,
			event_json TEXT NOT NULL,
			prev_hash TEXT NOT NULL,
			hash TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

type OperationDecision string

const (
	OperationExecute   OperationDecision = "execute"
	OperationCached    OperationDecision = "cached"
	OperationConflict  OperationDecision = "conflict"
	OperationReconcile OperationDecision = "reconcile_required"
)

func (s *Store) BeginOperation(ctx context.Context, invocationID, subject, tool, requestHash string) (OperationDecision, []byte, error) {
	if invocationID == "" {
		return "", nil, errors.New("invocation id is required")
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO operations(invocation_id, subject, tool, request_hash, state, created_at, updated_at)
		 VALUES(?,?,?,?, 'pending', ?, ?)`,
		invocationID, subject, tool, requestHash, now, now)
	if err != nil {
		return "", nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return "", nil, err
	}
	if n == 1 {
		return OperationExecute, nil, nil
	}

	var existingHash, state string
	var response []byte
	if err := s.db.QueryRowContext(ctx,
		`SELECT request_hash, state, response FROM operations WHERE invocation_id=?`, invocationID).
		Scan(&existingHash, &state, &response); err != nil {
		return "", nil, err
	}
	if existingHash != requestHash {
		return OperationConflict, nil, nil
	}
	if state == "done" {
		return OperationCached, response, nil
	}
	return OperationReconcile, nil, nil
}

func (s *Store) AbortOperation(ctx context.Context, invocationID string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM operations WHERE invocation_id=? AND state='pending'`, invocationID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("operation is not pending")
	}
	return nil
}

func (s *Store) CompleteOperation(ctx context.Context, invocationID string, response []byte) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE operations SET state='done', response=?, updated_at=? WHERE invocation_id=? AND state='pending'`,
		response, time.Now().Unix(), invocationID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("operation is not pending")
	}
	return nil
}

type Lock struct {
	Resource  string
	Owner     string
	Token     int64
	ExpiresAt time.Time
}

func (s *Store) AcquireLock(ctx context.Context, resource, owner string, ttl time.Duration) (Lock, error) {
	if resource == "" || owner == "" {
		return Lock{}, errors.New("resource and owner are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Lock{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	var curOwner string
	var curToken, curExpires int64
	err = tx.QueryRowContext(ctx,
		`SELECT owner, token, expires_at FROM resource_locks WHERE resource=?`, resource).
		Scan(&curOwner, &curToken, &curExpires)
	switch {
	case err == nil && curExpires > now.UnixNano():
		return Lock{}, fmt.Errorf("resource locked by %s", curOwner)
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return Lock{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM resource_locks WHERE resource=?`, resource); err != nil {
		return Lock{}, err
	}

	var generation int64
	err = tx.QueryRowContext(ctx, `SELECT token FROM lock_generation WHERE resource=?`, resource).Scan(&generation)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Lock{}, err
	}
	generation++
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO lock_generation(resource, token) VALUES(?,?)
		 ON CONFLICT(resource) DO UPDATE SET token=excluded.token`,
		resource, generation); err != nil {
		return Lock{}, err
	}
	expires := now.Add(ttl)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO resource_locks(resource, owner, token, expires_at) VALUES(?,?,?,?)`,
		resource, owner, generation, expires.UnixNano()); err != nil {
		return Lock{}, err
	}
	if err := tx.Commit(); err != nil {
		return Lock{}, err
	}
	return Lock{Resource: resource, Owner: owner, Token: generation, ExpiresAt: expires}, nil
}

func (s *Store) ReleaseLock(ctx context.Context, l Lock) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM resource_locks WHERE resource=? AND owner=? AND token=?`,
		l.Resource, l.Owner, l.Token)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

type Grant struct {
	ID           string
	Subject      string
	Capabilities []string
	ExpiresAt    time.Time
	Revoked      bool
}

func randomID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b[:]), nil
}

func canonicalCaps(in []string) []string {
	set := make(map[string]struct{}, len(in))
	for _, c := range in {
		c = strings.TrimSpace(strings.ToLower(c))
		if c != "" {
			set[c] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func (s *Store) IssueGrant(ctx context.Context, subject string, capabilities []string, ttl time.Duration) (Grant, error) {
	if subject == "" {
		return Grant{}, errors.New("subject is required")
	}
	if ttl <= 0 {
		return Grant{}, errors.New("ttl must be positive")
	}
	caps := canonicalCaps(capabilities)
	if len(caps) == 0 {
		return Grant{}, errors.New("at least one capability is required")
	}
	id, err := randomID("gr_")
	if err != nil {
		return Grant{}, err
	}
	exp := time.Now().Add(ttl)
	raw, _ := json.Marshal(caps)
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO grants(grant_id, subject, capabilities, expires_at) VALUES(?,?,?,?)`,
		id, subject, string(raw), exp.UnixNano()); err != nil {
		return Grant{}, err
	}
	return Grant{ID: id, Subject: subject, Capabilities: caps, ExpiresAt: exp}, nil
}

func (s *Store) GetGrant(ctx context.Context, grantID string) (Grant, error) {
	var g Grant
	var rawCaps string
	var expires int64
	var revoked sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT subject, capabilities, expires_at, revoked_at FROM grants WHERE grant_id=?`, grantID).
		Scan(&g.Subject, &rawCaps, &expires, &revoked)
	if err != nil {
		return Grant{}, err
	}
	g.ID = grantID
	g.ExpiresAt = time.Unix(0, expires)
	g.Revoked = revoked.Valid
	if err := json.Unmarshal([]byte(rawCaps), &g.Capabilities); err != nil {
		return Grant{}, err
	}
	return g, nil
}

func (s *Store) ValidateGrant(ctx context.Context, grantID, subject, capability string) (bool, error) {
	var storedSubject, rawCaps string
	var expires int64
	var revoked sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT subject, capabilities, expires_at, revoked_at FROM grants WHERE grant_id=?`, grantID).
		Scan(&storedSubject, &rawCaps, &expires, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if storedSubject != subject || revoked.Valid || time.Now().UnixNano() >= expires {
		return false, nil
	}
	var caps []string
	if err := json.Unmarshal([]byte(rawCaps), &caps); err != nil {
		return false, err
	}
	capability = strings.ToLower(strings.TrimSpace(capability))
	for _, c := range caps {
		if c == capability {
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) RevokeAll(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE grants SET revoked_at=? WHERE revoked_at IS NULL`, time.Now().UnixNano())
	return err
}

type AuditEvent struct {
	Time     string `json:"time"`
	Subject  string `json:"subject"`
	Tool     string `json:"tool"`
	Resource string `json:"resource,omitempty"`
	Decision string `json:"decision"`
	ActionID string `json:"action_id,omitempty"`
}

func auditDigest(seq int64, prev, eventJSON string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%d\n%s\n%s", seq, prev, eventJSON)))
	return hex.EncodeToString(h[:])
}

func (s *Store) AppendAudit(ctx context.Context, ev AuditEvent) (string, error) {
	if ev.Time == "" {
		ev.Time = time.Now().UTC().Format(time.RFC3339Nano)
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var seq int64
	var prev string
	err = tx.QueryRowContext(ctx, `SELECT seq, hash FROM audit_events ORDER BY seq DESC LIMIT 1`).Scan(&seq, &prev)
	if errors.Is(err, sql.ErrNoRows) {
		seq, prev = 0, "GENESIS"
	} else if err != nil {
		return "", err
	}
	seq++
	digest := auditDigest(seq, prev, string(raw))
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO audit_events(seq,event_json,prev_hash,hash) VALUES(?,?,?,?)`,
		seq, string(raw), prev, digest); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return digest, nil
}


type AuditRecord struct {
	Seq      int64      `json:"seq"`
	Event    AuditEvent `json:"event"`
	PrevHash string     `json:"prev_hash"`
	Hash     string     `json:"hash"`
}

func (s *Store) ListAuditAfter(ctx context.Context, afterSeq int64, limit int) ([]AuditRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT seq,event_json,prev_hash,hash FROM audit_events WHERE seq>? ORDER BY seq LIMIT ?`,
		afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditRecord
	for rows.Next() {
		var rec AuditRecord
		var raw string
		if err := rows.Scan(&rec.Seq, &raw, &rec.PrevHash, &rec.Hash); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &rec.Event); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

type AuditStatus struct {
	Valid    bool   `json:"valid"`
	Events   int64  `json:"events"`
	HeadHash string `json:"head_hash"`
}

func (s *Store) AuditStatus(ctx context.Context) (AuditStatus, error) {
	if err := s.VerifyAudit(ctx); err != nil {
		return AuditStatus{Valid: false}, err
	}
	var status AuditStatus
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(MAX(seq),0) FROM audit_events`).Scan(&status.Events, new(int64))
	if err != nil {
		return AuditStatus{}, err
	}
	if status.Events > 0 {
		if err := s.db.QueryRowContext(ctx, `SELECT hash FROM audit_events ORDER BY seq DESC LIMIT 1`).Scan(&status.HeadHash); err != nil {
			return AuditStatus{}, err
		}
	} else {
		status.HeadHash = "GENESIS"
	}
	status.Valid = true
	return status, nil
}

func (s *Store) VerifyAudit(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT seq,event_json,prev_hash,hash FROM audit_events ORDER BY seq`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var expectedSeq int64 = 1
	prev := "GENESIS"
	for rows.Next() {
		var seq int64
		var eventJSON, prevHash, digest string
		if err := rows.Scan(&seq, &eventJSON, &prevHash, &digest); err != nil {
			return err
		}
		if seq != expectedSeq || prevHash != prev {
			return fmt.Errorf("audit chain sequence mismatch at %d", seq)
		}
		if auditDigest(seq, prevHash, eventJSON) != digest {
			return fmt.Errorf("audit hash mismatch at %d", seq)
		}
		expectedSeq++
		prev = digest
	}
	return rows.Err()
}

func HashRequest(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
