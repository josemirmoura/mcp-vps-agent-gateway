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

// ErrBrokerStillRunning means the Broker did not acknowledge cancellation.
// Fail closed: the node must stop polling instead of risking overlapping
// local operations or re-dispatch of a task with uncertain host side effects.
var ErrBrokerStillRunning = errors.New("Broker operation still running after cancellation; stop Cloud node connector")

type Runner struct {
	Client            *Client
	Identity          Identity
	Broker            BrokerExecutor
	BrokerSubject     string
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	RenewInterval     time.Duration
	BrokerTimeout     time.Duration
	BrokerDrainTimeout time.Duration
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
	if r.BrokerDrainTimeout <= 0 {
		r.BrokerDrainTimeout = 5 * time.Second
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
			if errors.Is(err, ErrBrokerStillRunning) {
				// Unknown execution outcome: never acquire another Cloud
				// task in this connector process until operator review.
				return err
			}
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
	if err := validateTaskDeliveryWindow(task, time.Now().UTC()); err != nil {
		// The Cloud lease is no longer valid. Do not dispatch to the Broker
		// or attempt completion with an expired lease ID.
		return err
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

	// A task expiry is a fixed deadline. Delivery leases are renewable:
	// renewLoop must cancel the Broker on an unrenewed lease deadline rather
	// than permanently capping execution at the *initial* lease expiry.
	// Cancellation remains best effort after host-side work has started.
	brokerDeadline := time.Now().Add(r.BrokerTimeout)
	if task.ExpiresAt != nil && task.ExpiresAt.Before(brokerDeadline) {
		brokerDeadline = *task.ExpiresAt
	}
	brokerCtx, cancelBroker := context.WithDeadline(ctx, brokerDeadline)
	defer cancelBroker()

	type brokerResult struct {
		response wire.Response
		err      error
	}
	brokerDone := make(chan brokerResult, 1)
	go func() {
		if err := brokerCtx.Err(); err != nil {
			brokerDone <- brokerResult{err: err}
			return
		}
		response, err := r.Broker.Call(brokerCtx, brokerRequest)
		brokerDone <- brokerResult{response: response, err: err}
	}()

	// A cancellation request over IPC does not prove the Broker stopped.
	// Wait for its actual response before taking more tasks; if the Broker
	// ignores cancellation, terminate polling rather than overlap work.
	waitForBroker := func() error {
		grace := r.BrokerDrainTimeout
		if grace <= 0 {
			grace = 5 * time.Second
		}
		timer := time.NewTimer(grace)
		defer timer.Stop()
		select {
		case <-brokerDone:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return ErrBrokerStillRunning
		}
	}

	var result brokerResult
	select {
	case <-ctx.Done():
		cancelBroker()
		return nil
	case err := <-renewFatal:
		cancelBroker()
		if drainErr := waitForBroker(); drainErr != nil {
			return drainErr
		}
		return err
	case <-brokerCtx.Done():
		cancelBroker()
		if drainErr := waitForBroker(); drainErr != nil {
			return drainErr
		}
		return brokerCtx.Err()
	case result = <-brokerDone:
		// A completion racing the deadline must not be reported as success.
		if err := brokerCtx.Err(); err != nil {
			return err
		}
		select {
		case err := <-renewFatal:
			return err
		default:
		}
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
			if !result.response.OK {
				// A denied local Broker request must not become a successful Cloud
				// task merely because its signed completion transport succeeded.
				// Keep the error text stable and non-secret; detailed local
				// authorization evidence belongs to the Broker audit chain.
				completion.Error = "local Broker denied operation"
			}
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

// validateTaskDeliveryWindow checks Cloud delivery deadlines before any
// node-local action. A timed-out lease may be requeued by Cloud; the node must
// not act on it or use the stale lease to report a completion.
func validateTaskDeliveryWindow(task Task, now time.Time) error {
	if task.LeaseExpiresAt != nil && !now.Before(*task.LeaseExpiresAt) {
		return errors.New("Cloud task lease expired before Broker dispatch")
	}
	if task.ExpiresAt != nil && !now.Before(*task.ExpiresAt) {
		return errors.New("Cloud task expired before Broker dispatch")
	}
	return nil
}

func (r Runner) renewLoop(ctx context.Context, task Task, fatal chan<- error) {
	ticker := time.NewTicker(r.RenewInterval)
	defer ticker.Stop()

	reportFatal := func(err error) {
		select {
		case fatal <- err:
		default:
		}
	}

	// Preserve the current known lease deadline locally even if Cloud is
	// unreachable. A successful signed renewal replaces this deadline.
	var leaseTimer *time.Timer
	var leaseDeadline <-chan time.Time
	var confirmedDeadline time.Time
	defer func() {
		if leaseTimer != nil {
			leaseTimer.Stop()
		}
	}()
	setDeadline := func(expiresAt time.Time) bool {
		remaining := time.Until(expiresAt)
		if remaining <= 0 {
			reportFatal(errors.New("Cloud lease expired during Broker execution"))
			return false
		}
		confirmedDeadline = expiresAt
		if leaseTimer == nil {
			leaseTimer = time.NewTimer(remaining)
			leaseDeadline = leaseTimer.C
			return true
		}
		if !leaseTimer.Stop() {
			select {
			case <-leaseTimer.C:
			default:
			}
		}
		leaseTimer.Reset(remaining)
		return true
	}
	if task.LeaseExpiresAt != nil && !setDeadline(*task.LeaseExpiresAt) {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-leaseDeadline:
			reportFatal(errors.New("Cloud lease expired without renewal"))
			return
		case <-ticker.C:
			// The renewal HTTP call itself cannot stall past the last known
			// lease deadline while leaving the local Broker running.
			requestCtx := ctx
			var cancelRequest context.CancelFunc
			if !confirmedDeadline.IsZero() {
				requestCtx, cancelRequest = context.WithDeadline(ctx, confirmedDeadline)
			}
			expiresAt, err := r.Client.RenewTaskLease(requestCtx, r.Identity, task.ID, task.LeaseID)
			if cancelRequest != nil {
				cancelRequest()
			}
			if !confirmedDeadline.IsZero() && !time.Now().Before(confirmedDeadline) {
				reportFatal(errors.New("Cloud lease expired while renewal was in flight"))
				return
			}
			if err == nil {
				if !setDeadline(expiresAt) {
					return
				}
				continue
			}
			if IsStatus(err, http.StatusUnauthorized) || IsStatus(err, http.StatusConflict) {
				reportFatal(err)
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
