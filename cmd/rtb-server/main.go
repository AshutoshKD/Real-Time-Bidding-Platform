package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"rtb/internal/auction"
	"rtb/internal/realtime"
)

// Basic types for the HTTP API (auctions CRUD) kept in this file for simplicity of scaffold.
type CreateAuctionRequest struct {
	Title            string  `json:"title"`
	StartPrice       float64 `json:"startPrice"`
	MinIncrement     float64 `json:"minIncrement"`
	DurationSeconds  int64   `json:"durationSeconds"`
	SoftCloseSeconds int64   `json:"softCloseSeconds"`
	ReservePrice     float64 `json:"reservePrice"`
}

func toCents(v float64) int64 {
	return int64(v*100 + 0.5)
}

func fromCents(v int64) float64 {
	return float64(v) / 100
}

func main() {
	addr := getEnv("RTB_HTTP_ADDR", ":8080")
	mgr := auction.NewManager()

	r := mux.NewRouter()
	r.Use(simpleCORS)

	// Health
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods(http.MethodGet, http.MethodOptions)

	// Auctions API
	r.HandleFunc("/api/auctions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, mgr.List())
			return
		case http.MethodPost:
			var req CreateAuctionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid json")
				return
			}
			if req.Title == "" {
				writeErr(w, http.StatusBadRequest, "title required")
				return
			}
			if req.DurationSeconds <= 0 {
				req.DurationSeconds = 60
			}
			if req.MinIncrement <= 0 {
				req.MinIncrement = 1
			}
			a := mgr.Create(auction.CreateAuctionParams{
				Title:             req.Title,
				StartPriceCents:   toCents(req.StartPrice),
				MinIncrementCents: toCents(req.MinIncrement),
				DurationSeconds:   req.DurationSeconds,
				SoftCloseSeconds:  req.SoftCloseSeconds,
				ReservePriceCents: toCents(req.ReservePrice),
			})
			writeJSON(w, http.StatusCreated, a)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}).Methods(http.MethodGet, http.MethodPost, http.MethodOptions)

	r.HandleFunc("/api/auctions/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		a, ok := mgr.Get(id)
		if !ok {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, a)
	}).Methods(http.MethodGet, http.MethodOptions)

	// Realtime WebSocket
	r.Handle("/ws", &realtime.WSHandler{Mgr: mgr})
	// WebRTC signaling over WebSocket
	r.Handle("/signal", &realtime.SignalWS{Mgr: mgr})

	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("rtb-server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// simpleCORS is a minimal CORS middleware for local dev and demo.
func simpleCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}


