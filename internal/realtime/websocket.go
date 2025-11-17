package realtime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"rtb/internal/auction"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Allow any origin for demo purposes.
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSHandler struct {
	Mgr *auction.Manager
}

type clientJoin struct {
	Type   string         `json:"type"`
	RoomID string         `json:"roomId"`
	User   auction.User   `json:"user"`
}

type clientBid struct {
	Type      string       `json:"type"`
	RoomID    string       `json:"roomId"`
	User      auction.User `json:"user"`
	AmountCts int64        `json:"amountCents"`
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Expect a join message first
	var join clientJoin
	_, data, err := conn.ReadMessage()
	if err != nil {
		log.Printf("ws read join: %v", err)
		return
	}
	if err := json.Unmarshal(data, &join); err != nil || join.Type != "join_room" || join.RoomID == "" || join.User.ID == "" {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"expected join_room"}`))
		return
	}

	room := h.Mgr.RoomFor(join.RoomID)
	if room == nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"room_not_found"}`))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subID, events, cancelSub := room.Subscribe()
	defer cancelSub()

	// Notify join
	room.Input() <- auction.Event{Type: "join_room", User: &auction.User{ID: join.User.ID, Handle: join.User.Handle}}

	// writer goroutine
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer func() {
			ticker.Stop()
			close(done)
		}()
		for {
			select {
			case out, ok := <-events:
				if !ok {
					return
				}
				bytes, _ := json.Marshal(out)
				_ = conn.WriteMessage(websocket.TextMessage, bytes)
			case <-ticker.C:
				_ = conn.WriteMessage(websocket.PingMessage, []byte("ping"))
			case <-ctx.Done():
				return
			}
		}
	}()

	// read loop
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// naive routing on "type"
		var t struct{ Type string `json:"type"` }
		if err := json.Unmarshal(msg, &t); err != nil {
			continue
		}
		switch t.Type {
		case "place_bid":
			var b clientBid
			if json.Unmarshal(msg, &b) == nil {
				room.Input() <- auction.Event{Type: "place_bid", User: &auction.User{ID: b.User.ID, Handle: b.User.Handle}, AmountCts: b.AmountCts}
			}
		case "leave_room":
			room.Input() <- auction.Event{Type: "leave_room", User: &join.User}
		}
	}

	// goodbye
	_ = subID
	room.Input() <- auction.Event{Type: "leave_room", User: &join.User}
}


