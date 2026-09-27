package state

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type JobRecord struct {
	ID        string
	Subject   string
	Tool      string
	Resource  string
	UnitName  string
	GrantID   string
	State     string
	Deadline  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	ExitCode  *int
	Detail    string
}

func (s *Store) CreateJob(ctx context.Context, j JobRecord) error {
	if j.ID == "" || j.Subject == "" || j.Tool == "" || j.State == "" || j.Deadline.IsZero() {
		return errors.New("job id, subject, tool, state and deadline are required")
	}
	now := time.Now()
	if j.CreatedAt.IsZero() {
		j.CreatedAt = now
	}
	j.UpdatedAt = now
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs(job_id,subject,tool,resource,unit_name,grant_id,state,deadline,created_at,updated_at,detail)
		  VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		j.ID, j.Subject, j.Tool, j.Resource, j.UnitName, j.GrantID, j.State,
		j.Deadline.UnixNano(), j.CreatedAt.UnixNano(), j.UpdatedAt.UnixNano(), j.Detail)
	return err
}

func (s *Store) UpdateJobState(ctx context.Context, id, stateName, detail string, exitCode *int) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET state=?, detail=?, exit_code=?, updated_at=? WHERE job_id=?`,
		stateName, detail, exitCode, time.Now().UnixNano(), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("job not found")
	}
	return nil
}

func (s *Store) GetJob(ctx context.Context, id string) (JobRecord, error) {
	var j JobRecord
	var deadline, created, updated int64
	var exit sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT job_id,subject,tool,resource,unit_name,grant_id,state,deadline,created_at,updated_at,exit_code,detail
		  FROM jobs WHERE job_id=?`, id).
		Scan(&j.ID, &j.Subject, &j.Tool, &j.Resource, &j.UnitName, &j.GrantID, &j.State,
			&deadline, &created, &updated, &exit, &j.Detail)
	if err != nil {
		return JobRecord{}, err
	}
	j.Deadline = time.Unix(0, deadline)
	j.CreatedAt = time.Unix(0, created)
	j.UpdatedAt = time.Unix(0, updated)
	if exit.Valid {
		v := int(exit.Int64)
		j.ExitCode = &v
	}
	return j, nil
}

func (s *Store) ListActiveJobs(ctx context.Context) ([]JobRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT job_id FROM jobs WHERE state IN ('starting','running','cancelling') ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]JobRecord, 0, len(ids))
	for _, id := range ids {
		j, err := s.GetJob(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, nil
}
