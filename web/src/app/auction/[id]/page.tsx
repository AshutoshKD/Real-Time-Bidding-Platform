"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { centsToDisplay } from "../../../lib/api";
import { connectRealtime, type RTBMessage, type RealtimeConn } from "../../../lib/realtime";

type RoomState = {
  auctionId: string;
  title: string;
  currentPriceCents: number;
  leaderUserId?: string;
  leaderHandle?: string;
  endsAt: string;
  softCloseSeconds: number;
  minIncrementCents: number;
  participants: number;
  reservePriceCents: number;
  bidHistory: Array<{
    userId: string;
    handle: string;
    amountCents: number;
    accepted: boolean;
    reason?: string;
    createdAt: string;
  }>;
  participantsList: Array<{ userId: string; handle: string }>;
};

export default function AuctionPage({ params }: { params: { id: string } }) {
  const roomId = params.id;
  const [state, setState] = useState<RoomState | null>(null);
  const [msRemaining, setMsRemaining] = useState(0);
  const [handle, setHandle] = useState<string>("");
  const [bidDelta, setBidDelta] = useState<number>(0);
  const connRef = useRef<RealtimeConn | null>(null);
  const [transport, setTransport] = useState<"webrtc" | "ws" | "">("");
  const [userId, setUserId] = useState<string>("");

  useEffect(() => {
    const saved = localStorage.getItem("rtb_handle") || "";
    setHandle(saved);
  }, []);

  useEffect(() => {
    // Safely access localStorage only on client
    let id = "";
    try {
      id = localStorage.getItem("rtb_user_id") || "";
      if (!id) {
        id = crypto.randomUUID();
        localStorage.setItem("rtb_user_id", id);
      }
    } catch {}
    setUserId(id);
  }, []);

  const user = useMemo(() => {
    const id = userId;
    const h = handle || (id ? `user-${id.slice(0, 4)}` : "");
    return { id, handle: h };
  }, [handle, userId]);

  useEffect(() => {
    if (!user.handle || !user.id) return;
    let closed = false;
    connectRealtime(roomId, user, onMessage)
      .then((c) => {
        if (closed) {
          c.close();
          return;
        }
        connRef.current = c;
        setTransport(c.transport);
      })
      .catch(() => {});
    return () => {
      closed = true;
      connRef.current?.close();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user.handle, user.id, roomId]);

  function onMessage(m: RTBMessage) {
    if (m.type === "room_state") {
      const rs = m.payload as RoomState;
      const fixed: RoomState = {
        ...rs,
        bidHistory: (rs as any).bidHistory ?? [],
        participantsList: (rs as any).participantsList ?? [],
      };
      setState(fixed);
    }
    if (m.type === "bid_accepted") {
      const p = (m as any).payload as {
        amountCents: number;
        leaderUserId: string;
        leaderHandle: string;
        endsAt: string | Date;
      };
      setState((prev) => {
        if (!prev) return prev;
        const endsAtISO = typeof p.endsAt === "string" ? p.endsAt : new Date(p.endsAt).toISOString();
        const newHistory = [
          ...((prev as any).bidHistory ?? []),
          {
            userId: p.leaderUserId,
            handle: p.leaderHandle,
            amountCents: p.amountCents,
            accepted: true,
            createdAt: new Date().toISOString(),
          },
        ];
        return {
          ...prev,
          currentPriceCents: p.amountCents,
          leaderUserId: p.leaderUserId,
          leaderHandle: p.leaderHandle,
          endsAt: endsAtISO,
          bidHistory: newHistory,
        };
      });
    }
    if (m.type === "presence") {
      const payload = (m as any).payload as { participants: number };
      if (payload && typeof payload.participants === "number") {
        setState((prev) => (prev ? { ...prev, participants: payload.participants } : prev));
      }
    }
  }

  useEffect(() => {
    let id: any;
    function tick() {
      if (!state) return;
      const t = new Date(state.endsAt).getTime() - Date.now();
      setMsRemaining(Math.max(0, t));
    }
    id = setInterval(tick, 200);
    return () => clearInterval(id);
  }, [state]);

  function placeBid() {
    if (!state || !connRef.current) return;
    const next = state.currentPriceCents + (bidDelta > 0 ? bidDelta * 100 : state.minIncrementCents);
    connRef.current.send({
      type: "place_bid",
      roomId,
      user,
      amountCents: next,
    });
  }

  function saveHandle(h: string) {
    setHandle(h);
    localStorage.setItem("rtb_handle", h);
  }

  return (
    <div className="space-y-6">
      <div className="bg-neutral-900 rounded-lg p-4">
        <div className="font-medium mb-1">How it works</div>
        <ul className="text-sm text-neutral-300 list-disc pl-5 space-y-1">
          <li>Enter your handle, then place bids.</li>
          <li>The next valid bid must be at least the current price + minimum increment.</li>
          <li>If a bid arrives near the end, the timer extends (anti-sniping).</li>
        </ul>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <div className="text-sm text-neutral-400">Auction</div>
          <div className="text-2xl font-semibold">{state?.title || "…"}</div>
        </div>
        <div className="text-sm text-neutral-400">Transport: <span className="text-neutral-200">{transport || "…"}</span></div>
      </div>

      <div className="grid lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-4">
          <div className="bg-neutral-900 rounded-lg p-4">
            <div className="text-neutral-400 text-sm">Current price</div>
            <div className="text-4xl font-bold">${centsToDisplay(state?.currentPriceCents || 0)}</div>
            <div className="text-sm text-neutral-400">
              Leader: {state?.leaderHandle ? <span className="text-neutral-200">{state.leaderHandle}</span> : "—"}
            </div>
            <div className="text-sm text-neutral-400">
              Ends in: <span className={msRemaining < 10_000 ? "text-amber-400" : ""}>{(msRemaining/1000).toFixed(1)}s</span>
            </div>
            <div className="text-sm text-neutral-400">
              Min increment: <span className="text-neutral-200">${centsToDisplay(state?.minIncrementCents || 0)}</span>
            </div>
            <div className="text-sm text-neutral-400">
              Next valid bid: <span className="text-neutral-200">
                ${centsToDisplay((state?.currentPriceCents || 0) + (state?.minIncrementCents || 0))}
              </span>
            </div>
            {state && state.reservePriceCents > 0 && state.currentPriceCents < state.reservePriceCents && (
              <div className="text-xs text-amber-400 mt-1">Reserve not met</div>
            )}
          </div>

          <div className="bg-neutral-900 rounded-lg p-4 space-y-3">
            <div className="text-lg font-medium">Place bid</div>
            <div className="grid sm:grid-cols-3 gap-3">
              <input
                placeholder="Your handle"
                value={handle}
                onChange={(e) => saveHandle(e.target.value)}
                className="px-3 py-2 rounded border border-neutral-800"
              />
              <input
                type="number"
                min={0}
                step="0.01"
                placeholder={`Custom increment (min +${centsToDisplay(state?.minIncrementCents || 0)})`}
                value={bidDelta}
                onChange={(e) => setBidDelta(Number(e.target.value))}
                className="px-3 py-2 rounded border border-neutral-800"
              />
              <button
                onClick={placeBid}
                disabled={!user.handle}
                className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 rounded disabled:opacity-50"
              >
                Bid +${bidDelta > 0 ? bidDelta.toFixed(2) : centsToDisplay(state?.minIncrementCents || 0)}
              </button>
            </div>
            {!user.handle && <div className="text-xs text-red-400">Enter your handle to enable bidding.</div>}
          </div>
        </div>

        <div className="space-y-4">
          <div className="bg-neutral-900 rounded-lg p-4">
            <div className="font-medium mb-2">Participants ({state?.participants ?? 0})</div>
            <div className="space-y-1 max-h-[200px] overflow-auto pr-1">
              {state?.participantsList?.length ? (
                state.participantsList.map((p) => (
                  <div key={p.userId} className="text-sm text-neutral-300">
                    {p.handle || p.userId.slice(0, 6)}
                  </div>
                ))
              ) : (
                <div className="text-neutral-500 text-sm">No one here yet.</div>
              )}
            </div>
          </div>
          <div className="bg-neutral-900 rounded-lg p-4">
            <div className="font-medium mb-2">Bid history</div>
            <div className="space-y-2 max-h-[380px] overflow-auto pr-1">
              {state?.bidHistory?.slice().reverse().map((b, i) => (
                <div key={i} className="flex items-center justify-between text-sm">
                  <div className={b.accepted ? "text-neutral-200" : "text-neutral-500"}>
                    <span className="text-neutral-400">{new Date(b.createdAt).toLocaleTimeString()}</span>{" "}
                    <span className="font-mono">${centsToDisplay(b.amountCents)}</span>{" "}
                    by <span className="font-medium">{b.handle}</span>
                  </div>
                  <div className={`text-xs ${b.accepted ? "text-emerald-400" : "text-red-400"}`}>
                    {b.accepted ? "accepted" : b.reason}
                  </div>
                </div>
              ))}
              {!state?.bidHistory?.length && <div className="text-neutral-500 text-sm">No bids yet.</div>}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}


