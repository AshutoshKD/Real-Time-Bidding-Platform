package auction

import "time"

type Auction struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	StartPriceCents  int64     `json:"startPriceCents"`
	MinIncrementCents int64    `json:"minIncrementCents"`
	ReservePriceCents int64    `json:"reservePriceCents"`
	EndsAt           time.Time `json:"endsAt"`
	SoftCloseSeconds int64     `json:"softCloseSeconds"`
	CreatedAt        time.Time `json:"createdAt"`
}

type User struct {
	ID     string `json:"id"`
	Handle string `json:"handle"`
}

type ParticipantView struct {
	UserID string `json:"userId"`
	Handle string `json:"handle"`
}

type CreateAuctionParams struct {
	Title            string
	StartPriceCents  int64
	MinIncrementCents int64
	DurationSeconds  int64
	SoftCloseSeconds int64
	ReservePriceCents int64
}


