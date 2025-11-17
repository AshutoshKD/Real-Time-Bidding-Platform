export type Auction = {
  id: string;
  title: string;
  startPriceCents: number;
  minIncrementCents: number;
  endsAt: string;
  softCloseSeconds: number;
  reservePriceCents: number;
  createdAt: string;
};

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function listAuctions(): Promise<Auction[]> {
  const res = await fetch(`${API_URL}/api/auctions`, { cache: "no-store" });
  if (!res.ok) throw new Error("failed to list auctions");
  return res.json();
}

export async function createAuction(input: {
  title: string;
  startPrice: number;
  minIncrement: number;
  durationSeconds: number;
  softCloseSeconds: number;
  reservePrice: number;
}): Promise<Auction> {
  const res = await fetch(`${API_URL}/api/auctions`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!res.ok) throw new Error("failed to create auction");
  return res.json();
}

export function centsToDisplay(cents: number): string {
  return (cents / 100).toFixed(2);
}


