## What this app does (and technologies used)
- Real-time bidding (RTB) demo: create auctions, join as a participant, place bids, see instant updates (price, leader, timer, bid history, presence).
- Technologies:
  - Backend: Go 1.22, Pion WebRTC (DataChannels) with secure WebSocket fallback, Gorilla Mux/WebSocket.
  - Frontend: Next.js 14 (App Router), React 18, Tailwind CSS.
  - Design: in-memory state for demo; one-goroutine-per-auction room engine; anti-sniping; backpressure-aware fan-out.

## How to run (backend and frontend)
- Backend (Go):
  ```bash
  cd /Users/Rijey/Github/Real-Time-Bidding
  go mod tidy
  go run ./cmd/rtb-server
  ```
  - Serves API and realtime at http://localhost:8080

- Frontend (Next.js):
  ```bash
  cd /Users/Rijey/Github/Real-Time-Bidding/web
  npm install
  # API URL is read from web/.env (NEXT_PUBLIC_API_URL); defaults to http://localhost:8080
  npm run dev
  ```
  - Open http://localhost:3000

Notes:
- WebRTC DataChannel label: `rtb-v1` (signaling over `/signal`). If WebRTC isn’t available, the UI auto-falls back to `/ws`.

## Features
- Live auctions
  - Create auctions with: title, start price, minimum increment, duration, soft-close (anti-sniping), optional reserve.
  - Join an auction, place bids, see updates instantly (leader, price, participants, bid history, timer).
- Anti-sniping (soft close)
  - If a bid arrives within N seconds of the end, the end time extends by N seconds.
- Concurrency and performance
  - One goroutine per auction (single-writer state), buffered input queue, slow-subscriber eviction for critical events.
- Resilient realtime
  - WebRTC for low latency; automatic WebSocket fallback for restrictive networks.
- Polished UI
  - Clear forms and helper text, participants list, next valid bid guidance, reserve status.