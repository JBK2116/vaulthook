package worker

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/JBK2116/vaulthook/internal/providers/github"
	"github.com/JBK2116/vaulthook/internal/providers/stripe"
)

const (
	// RetryIntervalSeconds is how long the worker waits before retrying
	// a failed webhook delivery.
	RetryIntervalSeconds = 30

	// workerInterval is how often the worker wakes to poll for webhooks.
	workerInterval = 30 * time.Second

	// replayInterval is how often the worker wakes to process replay events.
	replayInterval = 2 * time.Second

	// getEventsTimeout bounds the database fetch for the next batch of webhooks.
	getEventsTimeout = 5 * time.Second

	// forwardEventTimeout bounds forwarding a batch of webhooks to their destinations.
	forwardEventTimeout = 10 * time.Second

	// updateEventsTimeout bounds persisting delivery results to the database.
	updateEventsTimeout = 5 * time.Second

	// httpClientTimeout is the overall timeout for each forwarded HTTP request.
	httpClientTimeout = 10 * time.Second

	// maxIdleConns caps the number of idle keep-alive connections the client retains.
	maxIdleConns = 100

	// idleConnTimeout is how long an idle keep-alive connection is retained.
	idleConnTimeout = 90 * time.Second
)

// Worker struct is responsible for processing all webhook events that are
// stored in the database.
type Worker struct {
	sse    *events.Service
	repo   Repository
	logger *zerolog.Logger
	client *http.Client
}

var (
	ErrNoHooksToWork = errors.New("[Worker] no webhooks to work at the moment")
	ErrRateLimited   = errors.New("[Worker] rate limited")
)

// newWorker returns a pointer to a Worker backed by the provided values.
func newWorker(svc *events.Service, repo Repository, logger *zerolog.Logger) *Worker {
	return &Worker{
		sse:    svc,
		repo:   repo,
		logger: logger,
		client: &http.Client{
			Timeout: httpClientTimeout,
			Transport: &http.Transport{
				MaxIdleConns:    maxIdleConns,
				IdleConnTimeout: idleConnTimeout,
			},
		},
	}
}

// start kicks off a loop that causes the worker to run in the background.
func (w *Worker) start(ctx context.Context, signal <-chan struct{}) {
	ticker := time.NewTicker(workerInterval)
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

// startRetry kicks off a loop that causes the worker to run in the background
// following the configured retry interval.
func (w *Worker) startRetry(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(RetryIntervalSeconds) * time.Second)
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

// startReplay kicks off a loop that causes the worker to run in the background
// following a short interval for replay events.
func (w *Worker) startReplay(ctx context.Context) {
	ticker := time.NewTicker(replayInterval)
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
		// get the next batch of webhooks for processing
		getCtx, cancelGet := context.WithTimeout(ctx, getEventsTimeout)
		hooks, err := w.getNext(getCtx)
		cancelGet()
		if err != nil {
			if errors.Is(err, ErrNoHooksToWork) {
				break
			}
			w.logger.Error().Stack().Err(err).Msg("[Worker] error retrieving next webhook for processing")
			break
		}
		// forwarding attempt (updates is valid for use even if error is not nil)
		fwdCtx, cancelFwd := context.WithTimeout(ctx, forwardEventTimeout)
		updates := w.forwardEvent(fwdCtx, hooks)
		cancelFwd()
		// update the webhooks accordingly after the forwarding attempt
		updCtx, cancelUpd := context.WithTimeout(ctx, updateEventsTimeout)
		updatedHooks, err := w.updateEvent(updCtx, updates)
		cancelUpd()
		if err != nil {
			w.logger.Error().Stack().Err(err).Msg("[Worker] error occurred when updating webhooks")
			continue
		}
		// send the updated webhooks to the frontend
		for i := range updatedHooks {
			w.send(&updatedHooks[i])
		}
	}
}

// getNext retrieves the next appropriate batch of webhook events required for processing.
func (w *Worker) getNext(ctx context.Context) ([]model.Webhook, error) {
	evt, err := w.repo.GetEvents(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoHooksToWork
		}
		w.logger.Error().Stack().Err(err).Msg("[Worker] database error when getting next event")
		return nil, err
	}
	return evt, nil
}

// forwardEvent attempts to forward the webhook events to its destination URL.
func (w *Worker) forwardEvent(ctx context.Context, hooks []model.Webhook) []updateWebhook {
	ups := make([]updateWebhook, len(hooks))
	var wg sync.WaitGroup

	for i := range hooks {
		hook := hooks[i]
		wg.Add(1)
		go func(i int, hook model.Webhook) {
			defer wg.Done()

			// update values for batch insert into the database later
			var updates updateWebhook
			updates.id = hook.ID
			updates.forwardedTo = &hook.ForwardedTo
			updates.provName = model.ProviderName(hook.Provider)

			// update the destination if it has changed since the object was saved
			prov, err := providers.Cache.Get(ctx, model.ProviderName(hook.Provider))
			if err != nil {
				setDefaultUpdateValues(err.Error(), &updates)
				ups[i] = updates
				return
			}
			if prov.DestinationURL != hook.ForwardedTo {
				hook.ForwardedTo = prov.DestinationURL
			}

			// request only proceeds if the cache permits it
			ok, err := limiter.Allow(ctx, &prov)
			if err != nil {
				setDefaultUpdateValues(err.Error(), &updates)
				ups[i] = updates
				return
			}
			if !ok {
				setRateLimitedUpdateValues(http.StatusTooManyRequests, ErrRateLimited.Error(), "", &updates)
				ups[i] = updates
				return
			}

			// get the provider's destination URL
			payload := bytes.NewReader(hook.Payload)

			// configure the HTTP request payload
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, prov.DestinationURL, payload)
			if err != nil {
				setDefaultUpdateValues(err.Error(), &updates)
				ups[i] = updates
				return
			}

			// set provider-specific headers
			switch hook.Provider {
			case string(model.Stripe):
				if headerErr := stripe.SetForwardHeaders(req, hook.Headers); headerErr != nil {
					setDefaultUpdateValues(headerErr.Error(), &updates)
					ups[i] = updates
					return
				}
			case string(model.Github):
				if headerErr := github.SetForwardHeaders(req, hook.Headers); headerErr != nil {
					setDefaultUpdateValues(headerErr.Error(), &updates)
					ups[i] = updates
					return
				}
			}

			// payload and headers are set
			res, err := w.client.Do(req)
			if err != nil {
				// err only contains transport level errors
				setDefaultUpdateValues(err.Error(), &updates)
				ups[i] = updates
				return
			}
			defer func() {
				_ = res.Body.Close()
			}()

			// handle the response
			code := res.StatusCode
			switch {
			case code >= 200 && code < 300:
				setSuccessUpdateValues(code, &updates)
			case code == http.StatusTooManyRequests, code == http.StatusServiceUnavailable:
				setRateLimitedUpdateValues(code, ErrRateLimited.Error(), res.Header.Get("Retry-After"), &updates)
			case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
				setFailureUpdateValues(code, res.Status, &updates)
			case code >= http.StatusInternalServerError:
				setRetryableUpdateValues(code, res.Status, &updates)
			}

			ups[i] = updates
		}(i, hook)
	}

	wg.Wait()
	return ups
}

// updateEvent updates the received events' data in the database.
func (w *Worker) updateEvent(ctx context.Context, updates []updateWebhook) ([]model.Webhook, error) {
	return w.repo.UpdateEvents(ctx, updates)
}

// send pushes the received updated event to the frontend via the SSE pipeline.
func (w *Worker) send(hook *model.Webhook) {
	w.sse.Send(*hook)
}

// setDefaultUpdateValues configures the provided updateWebhook to standard
// failure values with a scheduled retry.
func setDefaultUpdateValues(err string, updates *updateWebhook) {
	nextRetry := time.Now().Add(time.Duration(RetryIntervalSeconds) * time.Second)
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
// These require operator intervention; retrying will not resolve them.
func setFailureUpdateValues(code int, err string, updates *updateWebhook) {
	updates.deliveryStatus = model.DeliveryStatusFailed
	updates.responseCode = &code
	updates.lastError = &err
	updates.nextRetryAt = nil
}

// setRetryableUpdateValues configures the update for transient 5xx responses.
// The worker will retry after the configured interval.
func setRetryableUpdateValues(code int, err string, updates *updateWebhook) {
	nextRetry := time.Now().Add(time.Duration(RetryIntervalSeconds) * time.Second)
	updates.deliveryStatus = model.DeliveryStatusFailed
	updates.responseCode = &code
	updates.lastError = &err
	updates.nextRetryAt = &nextRetry
}

// setRateLimitedUpdateValues configures the update for 429/503 responses.
// Honors the Retry-After header if present; otherwise falls back to the
// configured retry interval.
func setRateLimitedUpdateValues(code int, err, retryAfter string, updates *updateWebhook) {
	var nextRetry time.Time
	if secs, parseErr := strconv.Atoi(retryAfter); parseErr == nil && secs > 0 {
		nextRetry = time.Now().Add(time.Duration(secs) * time.Second)
	} else {
		nextRetry = time.Now().Add(time.Duration(RetryIntervalSeconds) * time.Second)
	}
	updates.deliveryStatus = model.DeliveryStatusFailed
	updates.responseCode = &code
	updates.lastError = &err
	updates.nextRetryAt = &nextRetry
}
