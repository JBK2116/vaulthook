// This file serves a mock http server for testing the main application.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Mode represents a simulated webhook destination behaviour.
type Mode string

const (
	// ModeSuccess — destination accepted the webhook. Proxy should mark delivery as succeeded.
	ModeSuccess Mode = "success"
	// Mode500 — destination crashed (unhandled exception, panic, etc.).
	// Proxy should retry with backoff; this is the most common transient failure.
	Mode500 Mode = "500"
	// Mode503 — destination is temporarily unavailable (deploy in progress, overloaded, etc.).
	// Proxy should retry; well-behaved proxies honour Retry-After if present.
	Mode503 Mode = "503"
	// Mode429 — destination is rate-limiting the proxy.
	// Proxy must back off; aggressive retrying here makes things worse.
	Mode429 Mode = "429"
	// Mode400 — destination explicitly rejected the payload (bad signature, schema mismatch, etc.).
	// Proxy should NOT retry; this is a permanent failure until the payload/config changes.
	Mode400 Mode = "400"
	// Mode401 — destination rejected the request as unauthorised (bad secret, expired token, etc.).
	// Proxy should NOT retry automatically; operator intervention required.
	Mode401 Mode = "401"
	// Mode404 — endpoint not found on the destination (misconfigured URL, deleted route, etc.).
	// Proxy should surface this prominently; retrying is pointless.
	Mode404 Mode = "404"
	// ModeTimeout — destination accepts the connection but never responds.
	// Tests whether the proxy enforces a read-deadline and marks the attempt as failed.
	ModeTimeout Mode = "timeout"
	// ModeSlow — destination responds, but only after a long delay (e.g. 8 s).
	// Tests proxy patience / configurable timeout thresholds.
	ModeSlow Mode = "slow"
	// ModeDrop — destination immediately closes the TCP connection without sending anything.
	// Tests proxy resilience to connection-reset errors (no HTTP response at all).
	ModeDrop Mode = "drop"
	// ModeFlaky — alternates success / 500 on every request.
	// Simulates an unstable destination; tests retry idempotency and deduplication.
	ModeFlaky Mode = "flaky"
)

var (
	mu          sync.RWMutex
	currentMode = ModeSuccess
	flakyToggle bool // tracks flaky alternation state
)

// Control handler — switch mode at runtime without restarting
//
//	POST /control   body: {"mode": "503"}
//	GET  /control   returns current mode
func controlHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		mode := currentMode
		mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"mode": string(mode)}); err != nil {
			log.Printf("error encoding control response: %v", err)
		}
	case http.MethodPost:
		var body struct {
			Mode Mode `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Mode == "" {
			http.Error(w, `{"error":"invalid body — send {\"mode\":\"<mode>\"}"}`, http.StatusBadRequest)
			return
		}
		mu.Lock()
		currentMode = body.Mode
		flakyToggle = false
		mu.Unlock()
		log.Printf("[control] mode switched to: %s", body.Mode)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"mode": string(body.Mode)}); err != nil {
			log.Printf("error encoding control response: %v", err)
		}
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// Webhook handler
//
//	POST /api/webhooks/stripe
func stripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	mu.Lock()
	mode := currentMode
	if mode == ModeFlaky {
		flakyToggle = !flakyToggle
	}
	toggle := flakyToggle
	mu.Unlock()
	log.Printf("[stripe] mode=%-10s  %s %s", mode, r.Method, r.URL.Path)
	switch mode {
	case ModeSuccess:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintln(w, `{"received":true}`); err != nil {
			log.Printf("error sending mode response: %v", err)
		}
	case Mode500:
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	case Mode503:
		w.Header().Set("Retry-After", "30")
		http.Error(w, `{"error":"service unavailable"}`, http.StatusServiceUnavailable)
	case Mode429:
		w.Header().Set("Retry-After", "60")
		http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
	case Mode400:
		http.Error(w, `{"error":"bad request — invalid payload or signature"}`, http.StatusBadRequest)
	case Mode401:
		w.Header().Set("WWW-Authenticate", `Bearer realm="webhook"`)
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	case Mode404:
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	case ModeTimeout:
		// Hold the connection open forever; the proxy must time out on its own.
		select {}
	case ModeSlow:
		// Respond after 8s set this just above/below the sender's configured timeout.
		time.Sleep(8 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintln(w, `{"received":true,"note":"slow response"}`); err != nil {
			log.Printf("error sending mode response: %v", err)
		}
	case ModeDrop:
		// Hijack and immediately close the TCP connection.
		// The sender receives a connection-reset / EOF with no HTTP response.
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack unsupported", http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		if err := conn.Close(); err != nil {
			log.Printf("error closing http connection: %v", err)
		}
		log.Println("[stripe] connection dropped (no HTTP response sent)")
	case ModeFlaky:
		// Alternates 200 / 500 on each successive request.
		// Use this to verify the proxy retries correctly and doesn't double-deliver on success.
		if toggle {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := fmt.Fprintln(w, `{"received":true,"note":"flaky — this one succeeded"}`); err != nil {
				log.Printf("error sending mode response: %v", err)
			}
		} else {
			http.Error(w, `{"error":"flaky — this one failed"}`, http.StatusInternalServerError)
		}
	default:
		log.Printf("[stripe] unknown mode %q, falling back to 200", mode)
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintln(w, `{"received":true}`); err != nil {
			log.Printf("error sending mode response: %v", err)
		}
	}
}

// Modes reference handler — GET /modes
func modesHandler(w http.ResponseWriter, r *http.Request) {
	modes := []map[string]string{
		{"mode": string(ModeSuccess), "description": "200 — happy path"},
		{"mode": string(Mode500), "description": "500 — destination crash; proxy should retry"},
		{"mode": string(Mode503), "description": "503 — destination unavailable; proxy should honour Retry-After"},
		{"mode": string(Mode429), "description": "429 — rate limited; proxy must back off"},
		{"mode": string(Mode400), "description": "400 — bad payload/signature; proxy should NOT retry"},
		{"mode": string(Mode401), "description": "401 — unauthorised; operator action required"},
		{"mode": string(Mode404), "description": "404 — wrong URL; retrying is pointless"},
		{"mode": string(ModeTimeout), "description": "no response — tests proxy read-deadline enforcement"},
		{"mode": string(ModeSlow), "description": "8 s delay — tests proxy timeout threshold"},
		{"mode": string(ModeDrop), "description": "TCP drop — tests proxy connection-reset handling"},
		{"mode": string(ModeFlaky), "description": "alternates 200/500 — tests retry idempotency"},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(modes); err != nil {
		log.Printf("error encoding modes: %v", err)
	}
}

func main() {
	// configure the server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/webhooks/stripe", stripeWebhookHandler)
	mux.HandleFunc("/control", controlHandler)
	mux.HandleFunc("/modes", modesHandler)
	addr := ":8081"
	addrFull := fmt.Sprintf("http://localhost%s/", addr)
	log.Printf("Webhook simulator listening on %s", addrFull)
	log.Printf("  POST /api/webhooks/stripe  — target endpoint")
	log.Printf("  GET  /control              — get current mode")
	log.Printf("  POST /control              — set mode  e.g. {\"mode\":\"503\"}")
	log.Printf("  GET  /modes                — list all modes")

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
		// Keep WriteTimeout high so ModeTimeout / ModeSlow don't get cut short server-side.
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
