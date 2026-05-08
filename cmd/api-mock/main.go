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
 * STRIPE SIMULATOR CONTROL API
 * ----------------------------
 * Success:  curl -X POST http://localhost:8081/control -d '{"mode":"success"}'
 * 500 Err:  curl -X POST http://localhost:8081/control -d '{"mode":"500"}'
 * 429 Rate: curl -X POST http://localhost:8081/control -d '{"mode":"429"}'
 * Slow:     curl -X POST http://localhost:8081/control -d '{"mode":"slow"}'
 * Drop TCP: curl -X POST http://localhost:8081/control -d '{"mode":"drop"}'
 * Recovery: curl -X POST http://localhost:8081/control -d '{"mode":"recovery", "recovery_fail_count": 100}'
 *
 * Check Stats: curl http://localhost:8081/stats
 */

type Mode string

const (
	ModeSuccess       Mode = "success"
	Mode500           Mode = "500"
	Mode503           Mode = "503"
	Mode429           Mode = "429"
	Mode400           Mode = "400"
	Mode401           Mode = "401"
	Mode404           Mode = "404"
	ModeTimeout       Mode = "timeout"
	ModeSlow          Mode = "slow"
	ModeDrop          Mode = "drop"
	ModeFlaky         Mode = "flaky"
	ModeRecovery      Mode = "recovery"
	ModeRandomFailure Mode = "random_failure"
)

var (
	mu             sync.RWMutex
	currentMode    = ModeSuccess
	rng            = rand.New(rand.NewSource(time.Now().UnixNano()))
	flakyFailRate  = 0.5
	randomFailRate = 0.3

	// Recovery state
	recoveryFailCount = 10
	recoveryCounter   = 0

	// Realistic Outage Simulation
	outageEnd time.Time

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

func stripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Add Network Jitter (0-150ms) to every request for realism
	time.Sleep(time.Duration(rng.Intn(150)) * time.Millisecond)

	mu.Lock()
	mode := currentMode

	// Handle Recovery logic
	shouldFail := false
	if mode == ModeRecovery {
		shouldFail = recoveryCounter < recoveryFailCount
		if shouldFail {
			recoveryCounter++
		}
	} else if mode == ModeFlaky {
		shouldFail = rng.Float64() < flakyFailRate
	} else if mode == ModeRandomFailure {
		shouldFail = rng.Float64() < randomFailRate
	}
	mu.Unlock()

	log.Printf("[stripe] mode=%-14s %s %s", mode, r.Method, r.URL.Path)

	switch mode {
	case ModeSuccess:
		sendJSON(w, http.StatusOK, map[string]bool{"received": true})

	case Mode500:
		stats.record(false)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)

	case Mode503:
		stats.record(false)
		// Realism: 503s often come from Nginx/Load Balancers as HTML
		w.Header().Set("Retry-After", fmt.Sprintf("%d", 5+rng.Intn(25)))
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, "<html><body><h1>503 Service Temporarily Unavailable</h1></body></html>")

	case Mode429:
		stats.record(false)
		w.Header().Set("Retry-After", fmt.Sprintf("%d", 5+rng.Intn(10)))
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "0")
		sendJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})

	case Mode400:
		stats.record(false)
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_request", "message": "invalid payload schema"})

	case ModeTimeout:
		stats.record(false)
		select {} // Block forever to trigger worker client timeout

	case ModeSlow:
		// Trickle response: Send headers, then trickle body bytes to test ReadDeadlines
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

	case ModeDrop:
		stats.record(false)
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, _ := hj.Hijack()
			conn.Close() // Hard TCP reset
		}

	case ModeRecovery:
		if shouldFail {
			stats.record(false)
			w.Header().Set("Retry-After", "5")
			http.Error(w, `{"error":"recovering"}`, http.StatusServiceUnavailable)
		} else {
			sendJSON(w, http.StatusOK, map[string]any{"received": true, "recovered": true})
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
			"mode":                string(currentMode),
			"recovery_fail_count": recoveryFailCount,
		})
		return
	}

	var body struct {
		Mode              Mode `json:"mode"`
		RecoveryFailCount int  `json:"recovery_fail_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}

	mu.Lock()
	currentMode = body.Mode
	recoveryCounter = 0
	if body.RecoveryFailCount > 0 {
		recoveryFailCount = body.RecoveryFailCount
	}
	mu.Unlock()
	stats.reset()
	log.Printf("!!! MODE CHANGED TO: %s !!!", body.Mode)
	w.WriteHeader(http.StatusOK)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/webhooks/stripe", stripeWebhookHandler)
	mux.HandleFunc("/control", controlHandler)
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		json.NewEncoder(w).Encode(stats)
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
