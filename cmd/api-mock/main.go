// This file serves a mock http server for testing the main application.
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

// Mode represents a simulated webhook destination behaviour.
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
	ModeRecovery      Mode = "recovery"       // Fails for first N requests, then succeeds — simulates a service coming back up
	ModeRandomFailure Mode = "random_failure" // Fails at a configurable probability
)

var (
	mu          sync.RWMutex
	currentMode = ModeSuccess

	rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	// Flaky: probability of failure (0.0–1.0), default 0.5
	flakyFailRate float64 = 0.5

	// Recovery: how many requests fail before switching to success permanently
	recoveryFailCount int = 10
	recoveryCounter   int = 0

	// RandomFailure: probability of failure (0.0–1.0), default 0.3
	randomFailRate float64 = 0.3

	// Stats
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
	s.Requests++
	if success {
		s.Successes++
	} else {
		s.Failures++
	}
	s.mu.Unlock()
}

func (s *ModeStats) reset() {
	s.mu.Lock()
	s.Requests = 0
	s.Successes = 0
	s.Failures = 0
	s.mu.Unlock()
}

func (s *ModeStats) snapshot() (int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Requests, s.Successes, s.Failures
}

// randomRetryAfter returns a realistic Retry-After value between 5–30s
func randomRetryAfter() string {
	return fmt.Sprintf("%d", 5+rng.Intn(26))
}

// Control handler
//
//	GET  /control              — get current mode + config
//	POST /control              — set mode and optional config
//
// Body options:
//
//	{"mode":"recovery","recovery_fail_count":5}
//	{"mode":"flaky","flaky_fail_rate":0.7}
//	{"mode":"random_failure","random_fail_rate":0.3}
func controlHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		resp := map[string]any{
			"mode":                string(currentMode),
			"flaky_fail_rate":     flakyFailRate,
			"recovery_fail_count": recoveryFailCount,
			"recovery_counter":    recoveryCounter,
			"random_fail_rate":    randomFailRate,
		}
		mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

	case http.MethodPost:
		var body struct {
			Mode              Mode    `json:"mode"`
			FlakyFailRate     float64 `json:"flaky_fail_rate"`
			RecoveryFailCount int     `json:"recovery_fail_count"`
			RandomFailRate    float64 `json:"random_fail_rate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Mode == "" {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		mu.Lock()
		currentMode = body.Mode
		recoveryCounter = 0
		if body.FlakyFailRate > 0 {
			flakyFailRate = body.FlakyFailRate
		}
		if body.RecoveryFailCount > 0 {
			recoveryFailCount = body.RecoveryFailCount
		}
		if body.RandomFailRate > 0 {
			randomFailRate = body.RandomFailRate
		}
		mu.Unlock()
		stats.reset()
		log.Printf("[control] mode switched to: %s", body.Mode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"mode": string(body.Mode)})

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Stats handler — GET /stats
func statsHandler(w http.ResponseWriter, r *http.Request) {
	req, succ, fail := stats.snapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{
		"requests":  req,
		"successes": succ,
		"failures":  fail,
	})
}

// Reset handler — POST /control/reset
func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	mu.Lock()
	recoveryCounter = 0
	mu.Unlock()
	stats.reset()
	log.Printf("[control] stats + counters reset")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"reset":true}`)
}

// Webhook handler — POST /api/webhooks/stripe
func stripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	mu.Lock()
	mode := currentMode
	var shouldFail bool
	switch mode {
	case ModeFlaky:
		shouldFail = rng.Float64() < flakyFailRate
	case ModeRecovery:
		shouldFail = recoveryCounter < recoveryFailCount
		if shouldFail {
			recoveryCounter++
		}
	case ModeRandomFailure:
		shouldFail = rng.Float64() < randomFailRate
	}
	mu.Unlock()

	log.Printf("[stripe] mode=%-14s  %s %s", mode, r.Method, r.URL.Path)

	respond := func(success bool, fn func()) {
		stats.record(success)
		fn()
	}

	switch mode {
	case ModeSuccess:
		respond(true, func() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"received":true}`)
		})

	case Mode500:
		respond(false, func() {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		})

	case Mode503:
		respond(false, func() {
			w.Header().Set("Retry-After", randomRetryAfter())
			http.Error(w, `{"error":"service unavailable"}`, http.StatusServiceUnavailable)
		})

	case Mode429:
		respond(false, func() {
			w.Header().Set("Retry-After", randomRetryAfter())
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
		})

	case Mode400:
		respond(false, func() {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		})

	case Mode401:
		respond(false, func() {
			w.Header().Set("WWW-Authenticate", `Bearer realm="webhook"`)
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		})

	case Mode404:
		respond(false, func() {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		})

	case ModeTimeout:
		stats.record(false)
		select {}

	case ModeSlow:
		time.Sleep(8 * time.Second)
		respond(true, func() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"received":true,"note":"slow response"}`)
		})

	case ModeDrop:
		stats.record(false)
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack unsupported", http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
		log.Println("[stripe] connection dropped")

	case ModeFlaky:
		if shouldFail {
			respond(false, func() {
				http.Error(w, `{"error":"flaky — failed"}`, http.StatusInternalServerError)
			})
		} else {
			respond(true, func() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"received":true,"note":"flaky — succeeded"}`)
			})
		}

	case ModeRecovery:
		// Fails for the first N requests, then succeeds permanently
		if shouldFail {
			respond(false, func() {
				http.Error(w, `{"error":"service recovering"}`, http.StatusServiceUnavailable)
			})
		} else {
			respond(true, func() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"received":true,"note":"recovered"}`)
			})
		}

	case ModeRandomFailure:
		// Fails at a random probability — more realistic than flaky
		if shouldFail {
			respond(false, func() {
				http.Error(w, `{"error":"random failure"}`, http.StatusInternalServerError)
			})
		} else {
			respond(true, func() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"received":true}`)
			})
		}

	default:
		respond(true, func() {
			log.Printf("[stripe] unknown mode %q, falling back to 200", mode)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"received":true}`)
		})
	}
}

// Modes reference handler — GET /modes
func modesHandler(w http.ResponseWriter, r *http.Request) {
	modes := []map[string]string{
		{"mode": string(ModeSuccess), "description": "200 — happy path"},
		{"mode": string(Mode500), "description": "500 — destination crash; proxy should retry"},
		{"mode": string(Mode503), "description": "503 — unavailable; randomised Retry-After 5–30s"},
		{"mode": string(Mode429), "description": "429 — rate limited; randomised Retry-After 5–30s"},
		{"mode": string(Mode400), "description": "400 — bad payload; proxy should NOT retry"},
		{"mode": string(Mode401), "description": "401 — unauthorised; operator action required"},
		{"mode": string(Mode404), "description": "404 — wrong URL; retrying is pointless"},
		{"mode": string(ModeTimeout), "description": "no response — tests proxy read-deadline"},
		{"mode": string(ModeSlow), "description": "8s delay — tests proxy timeout threshold"},
		{"mode": string(ModeDrop), "description": "TCP drop — tests connection-reset handling"},
		{"mode": string(ModeFlaky), "description": "random probability failure — set flaky_fail_rate (default 0.5)"},
		{"mode": string(ModeRecovery), "description": "fails first N requests then succeeds — set recovery_fail_count (default 10)"},
		{"mode": string(ModeRandomFailure), "description": "random failure rate — set random_fail_rate (default 0.3)"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modes)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/webhooks/stripe", stripeWebhookHandler)
	mux.HandleFunc("/control", controlHandler)
	mux.HandleFunc("/control/reset", resetHandler)
	mux.HandleFunc("/stats", statsHandler)
	mux.HandleFunc("/modes", modesHandler)

	addr := ":8081"
	log.Printf("Webhook simulator listening on http://localhost%s/", addr)
	log.Printf("  POST /api/webhooks/stripe          — target endpoint")
	log.Printf("  GET  /control                      — get current mode + config")
	log.Printf("  POST /control                      — set mode + config")
	log.Printf("  POST /control/reset                — reset stats + counters")
	log.Printf("  GET  /stats                        — request/success/failure counts")
	log.Printf("  GET  /modes                        — list all modes")

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// CURLs
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"500"}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"503"}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"429"}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"400"}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"timeout"}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"slow"}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"drop"}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"flaky","flaky_fail_rate":0.7}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"recovery","recovery_fail_count":20}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"random_failure","random_fail_rate":0.4}'
// curl -X POST http://localhost:8081/control -H "Content-Type: application/json" -d '{"mode":"success"}'
// curl -X POST http://localhost:8081/control/reset
// curl http://localhost:8081/stats
