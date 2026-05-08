package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// Worker struct is responsible for processing all webhook events that are stored in the database.
type Worker struct {
	sse    *events.EventService
	repo   WorkerRepository
	logger *zerolog.Logger
}

var (
	ErrNoHooksToWork = errors.New("no webhooks to work at the moment")
	ErrRateLimited   = errors.New("rate limited")
)

// newWorker returns a pointer to a Worker backed by the provided values.
func newWorker(svc *events.EventService, repo WorkerRepository, logger *zerolog.Logger) *Worker {
	return &Worker{
		sse:    svc,
		repo:   repo,
		logger: logger,
	}
}

// start kicks off a loop that causes the worker to run in the background.
func (w *Worker) start(ctx context.Context, signal <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-signal:
			w.run(ctx)
		case <-ticker.C:
			w.run(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// startRetry kicks off a loop that causes the worker to run in the background following an interval.
func (w *Worker) startRetry(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(config.Envs.RetryIntervalSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.run(ctx)
		case <-ctx.Done():
			return
		}
	}

}

// run kicks off the Worker to begin working on webhooks.
func (w *Worker) run(ctx context.Context) {
	for {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*8)
		// get the next webhook for processing
		hook, err := w.getNext(timeoutCtx)
		if err != nil {
			cancel()
			if errors.Is(err, ErrNoHooksToWork) {
				break
			}
			w.logger.Error().Stack().Err(err).Msg("error retrieving next webhook for processing")
			break
		}
		// forwarding attempt (updates is valid for use even if error is not nil)
		updates, err := w.forwardEvent(timeoutCtx, hook)
		if err != nil {
			w.logger.Error().Stack().Err(err).Msg("error occurred when forwarding webhook")
		}
		// update the webhook accordingly after the forwarding attempt
		hook, err = w.updateEvent(timeoutCtx, updates)
		if err != nil {
			w.logger.Error().Stack().Err(err).Msg("error occurred when updating webhook")
			cancel()
			continue
		}
		// send the updated webhook to the frontend
		w.send(hook)
		cancel()
	}
}

// getNext retrieves the next appropriate webhook event required for processing.
func (w *Worker) getNext(ctx context.Context) (*model.Webhook, error) {
	evt, err := w.repo.GetEvent(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoHooksToWork
		}
		w.logger.Error().Stack().Err(err).Msg("worker experienced database error when getting next event")
		return nil, err
	}
	return evt, nil
}

// forwardEvent attempts to forward the webhook event to it's destination url.
func (w *Worker) forwardEvent(ctx context.Context, hook *model.Webhook) (updateWebhook, error) {
	// alert the frontend that processing has started for this webhook
	w.sse.Send(*hook)
	var updates updateWebhook
	updates.id = hook.ID
	// get the providers destination url
	payload := bytes.NewReader(hook.Payload)
	url, err := w.repo.GetDestinationURL(ctx, hook.ProviderID)
	if err != nil {
		setDefaultUpdateValues(err.Error(), &updates)
		return updates, err
	}
	// configure the http request payload
	req, err := http.NewRequestWithContext(ctx, "POST", url, payload)
	if err != nil {
		setDefaultUpdateValues(err.Error(), &updates)
		return updates, err
	}
	// set provider specific values
	if hook.Provider == string(model.Stripe) {
		if err := setStripeHeaders(req, hook.Headers); err != nil {
			setDefaultUpdateValues(err.Error(), &updates)
			return updates, err
		}
	}
	// payload and headers are set
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		// err only contains transport level errors
		setDefaultUpdateValues(err.Error(), &updates)
		return updates, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	// handle the response
	code := res.StatusCode
	switch {
	case code >= 200 && code < 300:
		setSuccessUpdateValues(code, &updates)
	case code == 429, code == 503:
		setRateLimitedUpdateValues(code, ErrRateLimited.Error(), res.Header.Get("Retry-After"), &updates)
	case code >= 400 && code < 500:
		setFailureUpdateValues(code, res.Status, &updates)
	case code >= 500:
		setRetryableUpdateValues(code, res.Status, &updates)
	}
	return updates, nil
}

// UpdateEvent updates the received events data in the database.
func (w *Worker) updateEvent(ctx context.Context, updates updateWebhook) (*model.Webhook, error) {
	hook, err := w.repo.UpdateEvent(ctx, updates)
	if err != nil {
		return nil, err
	}
	return hook, nil
}

// Send pushes the received updated event to the frontend via the sse pipeline.
func (w *Worker) send(hook *model.Webhook) {
	w.sse.Send(*hook)
}

// setStripeHeaders inputs the appropriate stripe headers into the provided http request object
func setStripeHeaders(r *http.Request, headers []byte) error {
	var parsed map[string][]string
	allowed := map[string]struct{}{
		"Content-Type":     {},
		"Stripe-Signature": {},
		"User-Agent":       {},
		"Cache-Control":    {},
	}
	if err := json.Unmarshal(headers, &parsed); err != nil {
		return err
	}
	for k, val := range parsed {
		if _, ok := allowed[k]; ok {
			for _, v := range val {
				r.Header.Add(k, v)
			}
		}
	}
	return nil
}

// setDefaultUpdateValues configures the provided updateWebhook to standard values.
func setDefaultUpdateValues(err string, updates *updateWebhook) {
	nextRetry := (time.Now().Add(time.Duration(config.Envs.RetryIntervalSeconds) * time.Second))
	updates.deliveryStatus = model.DeliveryStatusFailed
	updates.lastError = &err
	updates.nextRetryAt = &nextRetry
	updates.responseCode = nil
}

// setSuccessUpdateValues configures the update for a successful delivery (2xx).
func setSuccessUpdateValues(code int, updates *updateWebhook) {
	updates.deliveryStatus = model.DeliveryStatusDelivered
	updates.responseCode = &code
	updates.lastError = nil
	updates.nextRetryAt = nil
}

// setFailureUpdateValues configures the update for non-retryable 4xx responses.
// These require operator intervention, retrying will not resolve them.
func setFailureUpdateValues(code int, err string, updates *updateWebhook) {
	updates.deliveryStatus = model.DeliveryStatusFailed
	updates.responseCode = &code
	updates.lastError = &err
	updates.nextRetryAt = nil
}

// setRetryableUpdateValues configures the update for transient 5xx responses.
// The worker will retry after the configured interval.
func setRetryableUpdateValues(code int, err string, updates *updateWebhook) {
	nextRetry := time.Now().Add(time.Duration(config.Envs.RetryIntervalSeconds) * time.Second)
	updates.deliveryStatus = model.DeliveryStatusFailed
	updates.responseCode = &code
	updates.lastError = &err
	updates.nextRetryAt = &nextRetry
}

// setRateLimitedUpdateValues configures the update for 429 responses.
// Honours the Retry-After header if present, otherwise falls back to configured interval.
func setRateLimitedUpdateValues(code int, err, retryAfter string, updates *updateWebhook) {
	var nextRetry time.Time
	if secs, parseErr := strconv.Atoi(retryAfter); parseErr == nil && secs > 0 {
		nextRetry = time.Now().Add(time.Duration(secs) * time.Second)
	} else {
		nextRetry = time.Now().Add(time.Duration(config.Envs.RetryIntervalSeconds) * time.Second)
	}
	updates.deliveryStatus = model.DeliveryStatusFailed
	updates.responseCode = &code
	updates.lastError = &err
	updates.nextRetryAt = &nextRetry
}
