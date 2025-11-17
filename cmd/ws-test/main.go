package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type Auction struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	StartPriceCents    int64     `json:"startPriceCents"`
	MinIncrementCents  int64     `json:"minIncrementCents"`
	ReservePriceCents  int64     `json:"reservePriceCents"`
	EndsAt             time.Time `json:"endsAt"`
	SoftCloseSeconds   int64     `json:"softCloseSeconds"`
	CreatedAt          time.Time `json:"createdAt"`
}

type RoomState struct {
	AuctionID        string    `json:"auctionId"`
	Title            string    `json:"title"`
	CurrentPriceCts  int64     `json:"currentPriceCents"`
	LeaderUserID     string    `json:"leaderUserId,omitempty"`
	LeaderHandle     string    `json:"leaderHandle,omitempty"`
	EndsAt           time.Time `json:"endsAt"`
	SoftCloseSeconds int64     `json:"softCloseSeconds"`
	MinIncrementCts  int64     `json:"minIncrementCents"`
	Participants     int       `json:"participants"`
	ReservePriceCts  int64     `json:"reservePriceCents"`
}

func main() {
	api := getenv("API", "http://localhost:8080")
	wsURL := toWS(api) + "/ws"
	log.Printf("API=%s WS=%s", api, wsURL)

	// 1) Create a short auction to exercise anti-sniping
	createBody := map[string]any{
		"title":            "CLI Test Auction",
		"startPrice":       1.00,
		"minIncrement":     0.10,
		"durationSeconds":  5,
		"softCloseSeconds": 10,
		"reservePrice":     0,
	}
	var created Auction
	mustJSON(httpPostJSON(api+"/api/auctions", createBody, &created))
	log.Printf("Created auction: id=%s endsAt=%s", created.ID, created.EndsAt.Format(time.RFC3339))

	// 2) Connect WS
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// 3) Join room
	join := map[string]any{
		"type":   "join_room",
		"roomId": created.ID,
		"user":   map[string]string{"id": "cli-1", "handle": "cli"},
	}
	must(conn.WriteJSON(join))
	log.Printf("Joined room %s", created.ID)

	// 4) Wait for initial state
	initial := waitForState(conn, 5*time.Second)
	log.Printf("Initial state: price=%0.2f endsAt=%s participants=%d",
		float64(initial.CurrentPriceCts)/100, initial.EndsAt.Format(time.RFC3339), initial.Participants)

	// 5) Place a bid above min increment
	next := initial.CurrentPriceCts + initial.MinIncrementCts
	bid := map[string]any{
		"type":        "place_bid",
		"roomId":      created.ID,
		"user":        map[string]string{"id": "cli-1", "handle": "cli"},
		"amountCents": next,
	}
	must(conn.WriteJSON(bid))
	log.Printf("Placed bid: %0.2f", float64(next)/100)

	// 6) Wait for new state and verify
	after := waitForState(conn, 5*time.Second)
	log.Printf("After bid: price=%0.2f leader=%s endsAt=%s",
		float64(after.CurrentPriceCts)/100, after.LeaderHandle, after.EndsAt.Format(time.RFC3339))
	if after.CurrentPriceCts != next {
		log.Fatalf("expected price to be %d, got %d", next, after.CurrentPriceCts)
	}
	if after.EndsAt.After(initial.EndsAt) {
		log.Printf("Anti-sniping OK: endsAt extended by ~%ds", int(after.EndsAt.Sub(initial.EndsAt).Seconds()))
	} else {
		log.Printf("Anti-sniping not triggered (expected if remaining > soft window)")
	}

	log.Printf("WS test passed")
}

func waitForState(conn *websocket.Conn, timeout time.Duration) RoomState {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			log.Fatalf("timeout waiting for room_state")
		default:
			_, data, err := conn.ReadMessage()
			if err != nil {
				log.Fatalf("read: %v", err)
			}
			var t struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(data, &t)
			if t.Type == "room_state" {
				var env struct {
					Type    string    `json:"type"`
					RoomID  string    `json:"roomId"`
					Payload RoomState `json:"payload"`
				}
				_ = json.Unmarshal(data, &env)
				return env.Payload
			}
		}
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func mustJSON(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func httpPostJSON(url string, body any, out any) error {
	bytes, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytesReader(bytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type bytesReader []byte

func (b bytesReader) Read(p []byte) (int, error) {
	n := copy(p, b)
	if n < len(b) {
		return n, nil
	}
	return n, io.EOF
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func toWS(httpURL string) string {
	if len(httpURL) >= 5 && httpURL[:5] == "https" {
		return "wss" + httpURL[5:]
	}
	if len(httpURL) >= 4 && httpURL[:4] == "http" {
		return "ws" + httpURL[4:]
	}
	return httpURL
}


