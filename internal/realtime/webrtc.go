package realtime

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"rtb/internal/auction"
)

type SignalWS struct {
	Mgr *auction.Manager
}

type offerMsg struct {
	Type   string `json:"type"`
	SDP    string `json:"sdp"`
}

type answerMsg struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}

func (s *SignalWS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Read offer
	_, data, err := conn.ReadMessage()
	if err != nil {
		return
	}
	var offer offerMsg
	if err := json.Unmarshal(data, &offer); err != nil || offer.Type != "offer" || offer.SDP == "" {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"expected offer"}`))
		return
	}

	api := webrtc.NewAPI()
	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		log.Printf("pc create: %v", err)
		return
	}
	defer pc.Close()

	// DataChannel handling
	var room *auction.Room
	var user *auction.User
	var cancelSub func()

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if dc.Label() != "rtb-v1" {
			return
		}
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			var envelope struct {
				Type      string         `json:"type"`
				RoomID    string         `json:"roomId"`
				User      auction.User   `json:"user"`
				AmountCts int64          `json:"amountCents"`
			}
			if err := json.Unmarshal(msg.Data, &envelope); err != nil {
				return
			}
			switch envelope.Type {
			case "join_room":
				user = &auction.User{ID: envelope.User.ID, Handle: envelope.User.Handle}
				room = s.Mgr.RoomFor(envelope.RoomID)
				if room == nil {
					_ = dc.SendText(`{"type":"error","message":"room_not_found"}`)
					return
				}
				_, events, cancel := room.Subscribe()
				cancelSub = cancel
				room.Input() <- auction.Event{Type: "join_room", User: user}
				// writer for outbound
				go func() {
					for out := range events {
						bytes, _ := json.Marshal(out)
						_ = dc.SendText(string(bytes))
					}
				}()
			case "place_bid":
				if room != nil && user != nil {
					room.Input() <- auction.Event{Type: "place_bid", User: user, AmountCts: envelope.AmountCts}
				}
			case "leave_room":
				if room != nil && user != nil {
					room.Input() <- auction.Event{Type: "leave_room", User: user}
				}
			}
		})
		dc.OnClose(func() {
			if room != nil && user != nil {
				room.Input() <- auction.Event{Type: "leave_room", User: user}
			}
			if cancelSub != nil {
				cancelSub()
			}
		})
	})

	// Set remote offer
	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offer.SDP,
	}); err != nil {
		log.Printf("set remote: %v", err)
		return
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Printf("create answer: %v", err)
		return
	}
	gather := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		log.Printf("set local: %v", err)
		return
	}
	<-gather
	local := pc.LocalDescription()

	resp := answerMsg{Type: "answer", SDP: local.SDP}
	bytes, _ := json.Marshal(resp)
	_ = conn.WriteMessage(websocket.TextMessage, bytes)

	// Keep the websocket around a bit to ensure client receives the answer.
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, _ = conn.ReadMessage()
}


