package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/sandbox"
	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/state"
)

type Manager struct {
	State  *state.Store
	Runner Runner
}

func (m *Manager) Start(ctx context.Context, subject, tool, resource, grantID string, spec sandbox.Spec) (state.JobRecord, error) {
	if m.State == nil || m.Runner == nil {
		return state.JobRecord{}, errors.New("job manager is not configured")
	}
	if spec.Unit == "" {
		return state.JobRecord{}, errors.New("job unit is required")
	}
	now := time.Now()
	deadline := now.Add(spec.Runtime)
	if grantID != "" {
		grant, err := m.State.GetGrant(ctx, grantID)
		if err != nil {
			return state.JobRecord{}, fmt.Errorf("load job grant: %w", err)
		}
		if grant.Subject != subject || grant.Revoked || !grant.ExpiresAt.After(now) {
			return state.JobRecord{}, errors.New("job grant is invalid, revoked, expired, or belongs to another subject")
		}
		if grant.ExpiresAt.Before(deadline) {
			deadline = grant.ExpiresAt
		}
		spec.Runtime = time.Until(deadline)
		if spec.Runtime <= 0 {
			return state.JobRecord{}, errors.New("job deadline has already expired")
		}
	}
	rec := state.JobRecord{
		ID: spec.Unit, Subject: subject, Tool: tool, Resource: resource,
		UnitName: spec.Unit, GrantID: grantID, State: "starting", Deadline: deadline,
	}
	if err := m.State.CreateJob(ctx, rec); err != nil {
		return state.JobRecord{}, err
	}
	started, err := m.Runner.Start(ctx, spec)
	if err != nil {
		_ = m.State.UpdateJobState(context.Background(), rec.ID, "failed", err.Error(), nil)
		return state.JobRecord{}, err
	}
	rec.State = started.State
	rec.Deadline = started.Deadline
	if err := m.State.UpdateJobState(ctx, rec.ID, rec.State, "", nil); err != nil {
		return state.JobRecord{}, fmt.Errorf("job started but state update failed: %w", err)
	}
	return m.State.GetJob(ctx, rec.ID)
}

func (m *Manager) Status(ctx context.Context, subject, id string) (state.JobRecord, error) {
	rec, err := m.State.GetJob(ctx, id)
	if err != nil {
		return state.JobRecord{}, err
	}
	if rec.Subject != subject {
		return state.JobRecord{}, errors.New("job belongs to another subject")
	}
	if rec.State == "done" || rec.State == "failed" || rec.State == "cancelled" {
		return rec, nil
	}
	status, err := m.Runner.Status(ctx, Job{ID: rec.ID, Unit: rec.UnitName, State: rec.State, Deadline: rec.Deadline})
	if err != nil {
		return rec, err
	}
	if status != "" && status != rec.State {
		rec.State = status
		_ = m.State.UpdateJobState(ctx, rec.ID, status, "", nil)
	}
	return rec, nil
}

func (m *Manager) Tail(ctx context.Context, subject, id string, lines int) (string, error) {
	rec, err := m.State.GetJob(ctx, id)
	if err != nil {
		return "", err
	}
	if rec.Subject != subject {
		return "", errors.New("job belongs to another subject")
	}
	return m.Runner.Tail(ctx, Job{ID: rec.ID, Unit: rec.UnitName, State: rec.State, Deadline: rec.Deadline}, lines)
}

func (m *Manager) Cancel(ctx context.Context, subject, id string) error {
	rec, err := m.State.GetJob(ctx, id)
	if err != nil {
		return err
	}
	if rec.Subject != subject {
		return errors.New("job belongs to another subject")
	}
	if err := m.Runner.Cancel(ctx, Job{ID: rec.ID, Unit: rec.UnitName, State: rec.State, Deadline: rec.Deadline}); err != nil {
		return err
	}
	return m.State.UpdateJobState(ctx, rec.ID, "cancelled", "cancelled by request", nil)
}
