"use client";

import { useEffect, useState } from "react";
import { listAuctions, centsToDisplay, type Auction } from "../lib/api";

export default function AuctionList() {
  const [auctions, setAuctions] = useState<Auction[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    (async () => {
      setLoading(true);
      try {
        const res = await listAuctions();
        if (mounted) setAuctions(res);
      } finally {
        if (mounted) setLoading(false);
      }
    })();
    return () => {
      mounted = false;
    };
  }, []);

  return (
    <section>
      <h2 className="text-lg font-semibold mb-3">Live Auctions</h2>
      {loading ? (
        <div className="text-neutral-400">Loading…</div>
      ) : auctions.length === 0 ? (
        <div className="text-neutral-400">No auctions yet. Create one above.</div>
      ) : (
        <div className="grid md:grid-cols-2 gap-4">
          {auctions.map((a) => (
            <a
              key={a.id}
              href={`/auction/${a.id}`}
              className="block bg-neutral-900 rounded-lg p-4 hover:ring-1 hover:ring-neutral-700"
            >
              <div className="font-medium">{a.title}</div>
              <div className="text-sm text-neutral-400">
                Starts at ${centsToDisplay(a.startPriceCents)} • Min +${centsToDisplay(a.minIncrementCents)}
              </div>
              <div className="text-sm text-neutral-400">
                Ends at {new Date(a.endsAt).toLocaleTimeString()}
              </div>
            </a>
          ))}
        </div>
      )}
    </section>
  );
}


