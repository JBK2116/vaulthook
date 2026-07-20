package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

/**
 * WEBHOOK RECEIVER SIMULATOR CONTROL API
 * Success:      curl -X POST http://localhost:8081/control -d '{"mode":"success"}'
 * Outage:       curl -X POST http://localhost:8081/control -d '{"mode":"outage", "fail_count": 2000}'
 * Slow Recover: curl -X POST http://localhost:8081/control -d '{"mode":"slow_recover", "fail_count": 500}'
 * Fatal:        curl -X POST http://localhost:8081/control -d '{"mode":"fatal", "fail_count": 50000}'
 * Chaos:        curl -X POST http://localhost:8081/control -d '{"mode":"chaos", "fail_count": 1000}'
 * Flaky:        curl -X POST http://localhost:8081/control -d '{"mode":"flaky", "fail_rate": 0.2}'
 *
 * Check Stats:  curl http://localhost:8081/stats
 * Check Mode:   curl http://localhost:8081/control
 *
 * MODE DESCRIPTIONS
 * success      — Baseline. Every request returns 200 immediately.
 *
 * outage       — Simulates a full destination crash/restart. Returns 503 with
 *                Retry-After for fail_count requests, then heals to 200.
 *                Covers: 500, 503, recovery scenarios.
 *
 * slow_recover — Destination is alive but degraded. Trickles response bytes
 *                over several seconds for fail_count requests (tests read
 *                deadlines mid-stream), then heals to normal 200.
 *                Covers: slow destinations, partial network degradation.
 *
 * fatal        — Destination permanently rejects requests with random 4xx codes.
 *                fail_count is set extremely high (default 50000) so retry
 *                exhaustion is guaranteed. Simulates misconfigured destination,
 *                revoked auth, or deleted endpoint.
 *                Covers: 400, 401, 404 exhaustion scenarios.
 *
 * chaos        — Hard TCP reset (connection drop) for fail_count requests,
 *                then heals to 200. Tests whether the worker correctly detects
 *                connection-level errors vs timeouts.
 *                Covers: network partitions, hard process kills.
 *
 * flaky        — Random failure rate throughout the entire test. Never heals
 *                to full success — some percentage always fails. Tests that
 *                retry logic doesn't over-aggressively dead-letter events that
 *                eventually succeed.
 *                Covers: unreliable destinations, intermittent network issues.
 */

type Mode string

const (
	ModeSuccess     Mode = "success"
	ModeOutage      Mode = "outage"
	ModeSlowRecover Mode = "slow_recover"
	ModeFatal       Mode = "fatal"
	ModeChaos       Mode = "chaos"
	ModeFlaky       Mode = "flaky"
)

// 4xx codes fatal mode rotates through
var fatalCodes = []int{
	http.StatusBadRequest,
	http.StatusUnauthorized,
	http.StatusNotFound,
}

var fatalMessages = map[int]string{
	http.StatusBadRequest:   `{"error":"bad_request","message":"invalid payload schema"}`,
	http.StatusUnauthorized: `{"error":"unauthorized","message":"invalid or revoked credentials"}`,
	http.StatusNotFound:     `{"error":"not_found","message":"endpoint does not exist"}`,
}

var (
	mu          sync.RWMutex
	currentMode = ModeSuccess
	rng         = rand.New(rand.NewSource(time.Now().UnixNano()))

	// Shared heal counter — how many failures before mode heals to 200
	failCount   = 0
	failCounter = 0

	// Flaky — percentage of requests that fail
	flakyFailRate = 0.5

	stats = &ModeStats{}
)

type ModeStats struct {
	mu        sync.Mutex
	Requests  int `json:"requests"`
	Successes int `json:"successes"`
	Failures  int `json:"failures"`
}

func (s *ModeStats) record(success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Requests++
	if success {
		s.Successes++
	} else {
		s.Failures++
	}
}

func (s *ModeStats) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Requests, s.Successes, s.Failures = 0, 0, 0
}

// shouldFail checks the shared counter and returns true if still in failure window.
// Must be called with mu held.
func shouldFail() bool {
	if failCounter < failCount {
		failCounter++
		return true
	}
	return false
}

func uniWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Realistic network jitter on every request
	time.Sleep(time.Duration(rng.Intn(150)) * time.Millisecond)

	mu.Lock()
	mode := currentMode
	failing := false
	if mode == ModeOutage || mode == ModeSlowRecover || mode == ModeFatal || mode == ModeChaos {
		failing = shouldFail()
	} else if mode == ModeFlaky {
		failing = rng.Float64() < flakyFailRate
	}
	mu.Unlock()

	log.Printf("[uniWebhookHandler] mode=%-14s failing=%-5v %s %s", mode, failing, r.Method, r.URL.Path)

	switch mode {
	case ModeSuccess:
		sendJSON(w, http.StatusOK, map[string]bool{"received": true})

	case ModeOutage:
		if failing {
			stats.record(false)
			w.Header().Set("Retry-After", fmt.Sprintf("%d", 5+rng.Intn(25)))
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "<html><body><h1>503 Service Temporarily Unavailable</h1></body></html>")
		} else {
			sendJSON(w, http.StatusOK, map[string]any{"received": true, "recovered": true})
		}

	case ModeSlowRecover:
		if failing {
			// Trickle body bytes — tests read deadline mid-stream
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body := `{"received":true,"note":"trickle_complete"}`
			for _, char := range body {
				fmt.Fprintf(w, "%c", char)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				time.Sleep(300 * time.Millisecond)
			}
			stats.record(true)
		} else {
			sendJSON(w, http.StatusOK, map[string]bool{"received": true})
		}

	case ModeFatal:
		if failing {
			// Rotate through 4xx codes, covers bad_request, unauthorized, not_found
			// fail_count default is 50000 so retry exhaustion is guaranteed
			code := fatalCodes[rng.Intn(len(fatalCodes))]
			stats.record(false)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			fmt.Fprintln(w, fatalMessages[code])
		} else {
			// Heals eventually but so slowly it doesn't matter in practice
			sendJSON(w, http.StatusOK, map[string]bool{"received": true})
		}

	case ModeChaos:
		if failing {
			stats.record(false)
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
		} else {
			sendJSON(w, http.StatusOK, map[string]any{"received": true, "recovered": true})
		}

	case ModeFlaky:
		if failing {
			stats.record(false)
			code := fatalCodes[rng.Intn(len(fatalCodes))]
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			fmt.Fprintln(w, fatalMessages[code])
		} else {
			sendJSON(w, http.StatusOK, map[string]bool{"received": true})
		}

	default:
		sendJSON(w, http.StatusOK, map[string]bool{"received": true})
	}
}

func sendJSON(w http.ResponseWriter, code int, payload any) {
	stats.record(code < 300)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func controlHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		mu.RLock()
		defer mu.RUnlock()
		json.NewEncoder(w).Encode(map[string]any{
			"mode":          string(currentMode),
			"fail_count":    failCount,
			"fail_progress": fmt.Sprintf("%d/%d", failCounter, failCount),
		})
		return
	}

	var body struct {
		Mode      Mode    `json:"mode"`
		FailCount int     `json:"fail_count"`
		FailRate  float64 `json:"fail_rate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}

	mu.Lock()
	currentMode = body.Mode
	failCounter = 0

	switch body.Mode {
	case ModeFatal:
		// Default to 50000 if not specified, guarantees exhaustion
		if body.FailCount > 0 {
			failCount = body.FailCount
		} else {
			failCount = 50000
		}
	default:
		if body.FailCount > 0 {
			failCount = body.FailCount
		}
	}

	if body.FailRate > 0 {
		flakyFailRate = body.FailRate
	}
	mu.Unlock()

	stats.reset()
	log.Printf("!!! MODE CHANGED TO: %s (fail_count=%d) !!!", body.Mode, failCount)
	w.WriteHeader(http.StatusOK)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/webhooks", uniWebhookHandler)
	mux.HandleFunc("/control", controlHandler)
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		mode := currentMode
		fc := failCount
		fct := failCounter
		mu.RUnlock()
		stats.mu.Lock()
		defer stats.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{
			"mode":          string(mode),
			"fail_progress": fmt.Sprintf("%d/%d", fct, fc),
			"requests":      stats.Requests,
			"successes":     stats.Successes,
			"failures":      stats.Failures,
		})
	})

	srv := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Println("Simulator running on :8081...")
	srv.ListenAndServe()
}
