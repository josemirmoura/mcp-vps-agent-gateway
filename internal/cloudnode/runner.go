package cloudnode

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

type BrokerExecutor interface {
	Call(context.Context, wire.Request) (wire.Response, error)
}

type Runner struct {
	Client            *Client
	Identity          Identity
	Broker            BrokerExecutor
	BrokerSubject     string
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	RenewInterval     time.Duration
	BrokerTimeout     time.Duration
}

func (r Runner) Run(ctx context.Context) error {
	if r.Client == nil || r.Broker == nil {
		return errors.New("Cloud client and Broker executor are required")
	}
	if err := ValidateIdentity(r.Identity); err != nil {
		return err
	}
	if r.BrokerSubject == "" {
		return errors.New("local Broker subject is required")
	}
	if r.HeartbeatInterval <= 0 {
		r.HeartbeatInterval = 30 * time.Second
	}
	if r.PollInterval <= 0 {
		r.PollInterval = 2 * time.Second
	}
	if r.RenewInterval <= 0 {
		r.RenewInterval = 10 * time.Second
	}
	if r.BrokerTimeout <= 0 {
		r.BrokerTimeout = 15 * time.Minute
	}

	go r.heartbeatLoop(ctx)

	backoff := time.Second
	for ctx.Err() == nil {
		task, ok, err := r.Client.LeaseTask(ctx, r.Identity)
		if err != nil {
			slog.WarnContext(ctx, "portico_cloud_task_lease_failed", "error", err)
			if err := sleepContext(ctx, backoff); err != nil {
				break
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second

		if !ok {
			if err := sleepContext(ctx, r.PollInterval); err != nil {
				break
			}
			continue
		}

		if err := r.processTask(ctx, task); err != nil {
			slog.ErrorContext(ctx, "portico_cloud_task_failed",
				"task_id", task.ID,
				"operation", task.Operation,
				"error", err,
			)
		}
	}

	if ctx.Err() != nil {
		return nil
	}
	return errors.New("Cloud node loop stopped unexpectedly")
}

func (r Runner) heartbeatLoop(ctx context.Context) {
	send := func() {
		if err := r.Client.Heartbeat(ctx, r.Identity); err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "portico_cloud_heartbeat_failed", "error", err)
		}
	}
	send()

	ticker := time.NewTicker(r.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		}
	}
}

func (r Runner) processTask(ctx context.Context, task Task) error {
	if task.ID == "" || task.LeaseID == "" || task.Operation == "" {
		return errors.New("leased task is missing required execution fields")
	}
	if task.DestinationNodeID != r.Identity.NodeID {
		return errors.New("leased task targets a different node")
	}
	if len(task.Input) == 0 {
		task.Input = json.RawMessage("{}")
	}
	if !json.Valid(task.Input) {
		return r.completeExecutionError(ctx, task, "Cloud task input is invalid JSON")
	}

	brokerRequest := wire.Request{
		ID:           task.ID,
		Subject:      r.BrokerSubject,
		Tool:         task.Operation,
		Resource:     task.Resource,
		Action:       task.Action,
		InvocationID: task.ID,
		GrantID:      task.GrantID,
		Args:         task.Input,
	}

	renewCtx, stopRenewal := context.WithCancel(ctx)
	defer stopRenewal()
	renewFatal := make(chan error, 1)
	go r.renewLoop(renewCtx, task, renewFatal)

	brokerCtx, cancelBroker := context.WithTimeout(ctx, r.BrokerTimeout)
	defer cancelBroker()

	type brokerResult struct {
		response wire.Response
		err      error
	}
	brokerDone := make(chan brokerResult, 1)
	go func() {
		response, err := r.Broker.Call(brokerCtx, brokerRequest)
		brokerDone <- brokerResult{response: response, err: err}
	}()

	var result brokerResult
	select {
	case <-ctx.Done():
		return nil
	case err := <-renewFatal:
		cancelBroker()
		return err
	case result = <-brokerDone:
	}

	var completion Completion
	completion.LeaseID = task.LeaseID
	if result.err != nil {
		completion.Result = json.RawMessage("null")
		completion.Error = boundedError(result.err.Error())
	} else {
		raw, err := json.Marshal(result.response)
		if err != nil {
			completion.Result = json.RawMessage("null")
			completion.Error = boundedError("failed to encode Broker response: " + err.Error())
		} else {
			completion.Result = raw
		}
	}

	if err := r.completeWithRetry(ctx, task.ID, completion, renewFatal); err != nil {
		return err
	}
	stopRenewal()

	slog.InfoContext(ctx, "portico_cloud_task_completed",
		"task_id", task.ID,
		"operation", task.Operation,
		"broker_ok", result.err == nil && result.response.OK,
	)
	return nil
}

func (r Runner) renewLoop(ctx context.Context, task Task, fatal chan<- error) {
	ticker := time.NewTicker(r.RenewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := r.Client.RenewTaskLease(ctx, r.Identity, task.ID, task.LeaseID)
			if err == nil {
				continue
			}
			if IsStatus(err, http.StatusUnauthorized) || IsStatus(err, http.StatusConflict) {
				select {
				case fatal <- err:
				default:
				}
				return
			}
			if ctx.Err() == nil {
				slog.WarnContext(ctx, "portico_cloud_task_renew_failed",
					"task_id", task.ID,
					"error", err,
				)
			}
		}
	}
}

func (r Runner) completeWithRetry(
	ctx context.Context,
	taskID string,
	completion Completion,
	renewFatal <-chan error,
) error {
	backoff := time.Second
	for {
		err := r.Client.CompleteTask(ctx, r.Identity, taskID, completion)
		if err == nil {
			return nil
		}
		if IsStatus(err, http.StatusUnauthorized) || IsStatus(err, http.StatusConflict) {
			return err
		}

		select {
		case fatalErr := <-renewFatal:
			return fatalErr
		default:
		}

		slog.WarnContext(ctx, "portico_cloud_task_completion_retry",
			"task_id", taskID,
			"error", err,
			"retry_in", backoff,
		)
		if err := sleepContext(ctx, backoff); err != nil {
			return err
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (r Runner) completeExecutionError(ctx context.Context, task Task, message string) error {
	completion := Completion{
		LeaseID: task.LeaseID,
		Result:  json.RawMessage("null"),
		Error:   boundedError(message),
	}
	return r.Client.CompleteTask(ctx, r.Identity, task.ID, completion)
}

func boundedError(message string) string {
	const max = 4096
	if len(message) <= max {
		return message
	}
	return message[:max]
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
