package auction

import (
	"encoding/json"
	"math/rand/v2"
	"strconv"
	"sync"
	"time"
)

// Inbound events sent from realtime layer to the room.
type Event struct {
	Type      string          `json:"type"`
	User      *User           `json:"user,omitempty"`
	AmountCts int64           `json:"amountCents,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// Outbound messages broadcast to subscribers.
type Outbound struct {
	Type    string      `json:"type"`
	RoomID  string      `json:"roomId"`
	Payload interface{} `json:"payload,omitempty"`
}

// Public snapshot of room state for UI.
type RoomState struct {
	AuctionID        string            `json:"auctionId"`
	Title            string            `json:"title"`
	CurrentPriceCts  int64             `json:"currentPriceCents"`
	LeaderUserID     string            `json:"leaderUserId,omitempty"`
	LeaderHandle     string            `json:"leaderHandle,omitempty"`
	EndsAt           time.Time         `json:"endsAt"`
	SoftCloseSeconds int64             `json:"softCloseSeconds"`
	MinIncrementCts  int64             `json:"minIncrementCents"`
	Participants     int               `json:"participants"`
	ParticipantsList []ParticipantView `json:"participantsList"`
	ReservePriceCts  int64             `json:"reservePriceCents"`
	BidHistory       []BidView         `json:"bidHistory"`
}

type BidView struct {
	UserID    string    `json:"userId"`
	Handle    string    `json:"handle"`
	AmountCts int64     `json:"amountCents"`
	Accepted  bool      `json:"accepted"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Manager holds auctions and lazily creates rooms.
type Manager struct {
	mu       sync.RWMutex
	auctions map[string]*Auction
	rooms    map[string]*Room
}

func NewManager() *Manager {
	return &Manager{
		auctions: make(map[string]*Auction),
		rooms:    make(map[string]*Room),
	}
}

func (m *Manager) List() []*Auction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Auction, 0, len(m.auctions))
	for _, a := range m.auctions {
		out = append(out, a)
	}
	return out
}

func (m *Manager) Get(id string) (*Auction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.auctions[id]
	return a, ok
}

func (m *Manager) Create(p CreateAuctionParams) *Auction {
	now := time.Now().UTC()
	id := strconv.FormatInt(now.Unix(), 10) + "-" + strconv.Itoa(rand.IntN(999999))
	a := &Auction{
		ID:                id,
		Title:             p.Title,
		StartPriceCents:   p.StartPriceCents,
		MinIncrementCents: p.MinIncrementCents,
		ReservePriceCents: p.ReservePriceCents,
		EndsAt:            now.Add(time.Duration(p.DurationSeconds) * time.Second),
		SoftCloseSeconds:  p.SoftCloseSeconds,
		CreatedAt:         now,
	}
	m.mu.Lock()
	m.auctions[a.ID] = a
	m.mu.Unlock()
	return a
}

func (m *Manager) RoomFor(id string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rooms[id]; ok {
		return r
	}
	a, ok := m.auctions[id]
	if !ok {
		return nil
	}
	r := newRoom(a)
	m.rooms[id] = r
	go r.run()
	return r
}

// Room serializes all mutations to one goroutine and fan-outs updates to subscribers.
type Room struct {
	auction *Auction

	// dynamic state
	currentPriceCts int64
	leader          *User
	participants    map[string]*User
	bidHistory      []BidView

	// wiring
	input       chan Event
	subscribers map[int]chan Outbound
	nextSubID   int
	subReq      chan subscribeRequest
	unsubReq    chan int
}

type subscribeRequest struct {
	resp chan subscribeResponse
}
type subscribeResponse struct {
	id int
	ch chan Outbound
}

func newRoom(a *Auction) *Room {
	return &Room{
		auction:         a,
		currentPriceCts: a.StartPriceCents,
		participants:    make(map[string]*User),
		input:           make(chan Event, 4096),
		subscribers:     make(map[int]chan Outbound),
		subReq:          make(chan subscribeRequest),
		unsubReq:        make(chan int),
	}
}

func (r *Room) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case ev := <-r.input:
			r.handle(ev)
		case req := <-r.subReq:
			ch := make(chan Outbound, 256)
			id := r.nextSubID
			r.nextSubID++
			r.subscribers[id] = ch
			// Send an immediate snapshot to new subscriber to avoid waiting for next tick.
			state := r.buildState()
			select {
			case ch <- Outbound{Type: "room_state", RoomID: r.auction.ID, Payload: state}:
			default:
			}
			req.resp <- subscribeResponse{id: id, ch: ch}
		case id := <-r.unsubReq:
			if ch, ok := r.subscribers[id]; ok {
				delete(r.subscribers, id)
				close(ch)
			}
		case <-ticker.C:
			// periodic state broadcast
			r.broadcastState()
			// check closure
			if time.Now().UTC().After(r.auction.EndsAt) {
				// closed - continue broadcasting state; logic to stop could be added
			}
		}
	}
}

func (r *Room) handle(ev Event) {
	switch ev.Type {
	case "join_room":
		if ev.User != nil {
			r.participants[ev.User.ID] = ev.User
		}
		// Notify presence and state immediately.
		r.broadcast(Outbound{Type: "presence", RoomID: r.auction.ID, Payload: map[string]int{"participants": len(r.participants)}})
		r.broadcastState()
	case "leave_room":
		if ev.User != nil {
			delete(r.participants, ev.User.ID)
		}
		r.broadcast(Outbound{Type: "presence", RoomID: r.auction.ID, Payload: map[string]int{"participants": len(r.participants)}})
	case "place_bid":
		r.processBid(ev)
	}
}

func (r *Room) processBid(ev Event) {
	now := time.Now().UTC()
	user := ev.User
	amount := ev.AmountCts
	reason := ""
	accepted := false

	if user == nil {
		reason = "unauthorized"
	} else if now.After(r.auction.EndsAt) {
		reason = "auction_closed"
	} else if amount < r.currentPriceCts+r.auction.MinIncrementCents {
		reason = "below_min_increment"
	} else {
		// accept
		accepted = true
		r.currentPriceCts = amount
		r.leader = user
		// anti-sniping
		if r.auction.SoftCloseSeconds > 0 {
			remaining := time.Until(r.auction.EndsAt)
			if remaining <= time.Duration(r.auction.SoftCloseSeconds)*time.Second {
				r.auction.EndsAt = r.auction.EndsAt.Add(time.Duration(r.auction.SoftCloseSeconds) * time.Second)
			}
		}
	}

	entry := BidView{
		UserID:    userID(user),
		Handle:    userHandle(user),
		AmountCts: amount,
		Accepted:  accepted,
		Reason:    reason,
		CreatedAt: now,
	}
	r.bidHistory = append(r.bidHistory, entry)

	if accepted {
		r.broadcastCritical(Outbound{
			Type:   "bid_accepted",
			RoomID: r.auction.ID,
			Payload: map[string]any{
				"amountCents":  amount,
				"leaderUserId": user.ID,
				"leaderHandle": user.Handle,
				"endsAt":       r.auction.EndsAt,
			},
		})
		r.broadcastState()
	} else {
		r.broadcast(Outbound{
			Type:   "bid_rejected",
			RoomID: r.auction.ID,
			Payload: map[string]any{
				"reason": reason,
			},
		})
	}
}

func (r *Room) broadcastState() {
	state := r.buildState()
	r.broadcast(Outbound{
		Type:    "room_state",
		RoomID:  r.auction.ID,
		Payload: state,
	})
}

func (r *Room) buildState() RoomState {
	plist := make([]ParticipantView, 0, len(r.participants))
	for _, u := range r.participants {
		if u == nil {
			continue
		}
		plist = append(plist, ParticipantView{UserID: u.ID, Handle: u.Handle})
	}
	state := RoomState{
		AuctionID:        r.auction.ID,
		Title:            r.auction.Title,
		CurrentPriceCts:  r.currentPriceCts,
		EndsAt:           r.auction.EndsAt,
		SoftCloseSeconds: r.auction.SoftCloseSeconds,
		MinIncrementCts:  r.auction.MinIncrementCents,
		Participants:     len(r.participants),
		ParticipantsList: plist,
		ReservePriceCts:  r.auction.ReservePriceCents,
		BidHistory:       r.bidHistory,
	}
	if r.leader != nil {
		state.LeaderUserID = r.leader.ID
		state.LeaderHandle = r.leader.Handle
	}
	return state
}

func (r *Room) broadcast(msg Outbound) {
	for _, ch := range r.subscribers {
		select {
		case ch <- msg:
		default:
			// drop if subscriber is slow; critical events should be retried by client via next state tick
		}
	}
}

// broadcastCritical never silently drops; slow subscribers are evicted.
func (r *Room) broadcastCritical(msg Outbound) {
	for id, ch := range r.subscribers {
		select {
		case ch <- msg:
		default:
			// evict slow subscriber to keep the room loop healthy
			close(ch)
			delete(r.subscribers, id)
		}
	}
}

// Subscribe returns a channel for outbound messages.
func (r *Room) Subscribe() (int, <-chan Outbound, func()) {
	req := subscribeRequest{resp: make(chan subscribeResponse)}
	r.subReq <- req
	resp := <-req.resp
	cancel := func() {
		r.unsubReq <- resp.id
	}
	return resp.id, resp.ch, cancel
}

func (r *Room) Input() chan<- Event {
	return r.input
}

func userID(u *User) string {
	if u == nil {
		return ""
	}
	return u.ID
}
func userHandle(u *User) string {
	if u == nil {
		return ""
	}
	return u.Handle
}
