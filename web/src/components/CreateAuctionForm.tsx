"use client";

import { useMemo, useState } from "react";
import { createAuction } from "../lib/api";

export default function CreateAuctionForm() {
  const [form, setForm] = useState({
    title: "",
    startPrice: 1,
    minIncrement: 1,
    durationSeconds: 60,
    softCloseSeconds: 10,
    reservePrice: 0,
  });
  const canCreate = useMemo(() => form.title.trim().length >= 2, [form]);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!canCreate) return;
    const a = await createAuction(form);
    setForm({ ...form, title: "" });
    window.location.href = `/auction/${a.id}`;
  }

  return (
    <section className="bg-neutral-900 rounded-lg p-4">
      <h2 className="text-lg font-semibold mb-3">Create Auction</h2>
      <form onSubmit={onCreate} className="grid md:grid-cols-3 gap-6">
        <div className="space-y-1 md:col-span-3">
          <label className="text-sm text-neutral-300">Title</label>
          <input
            placeholder="e.g., Vintage Camera, MacBook Pro, Rare NFT"
            value={form.title}
            onChange={(e) => setForm({ ...form, title: e.target.value })}
            className="px-3 py-2 rounded border border-neutral-800 w-full"
          />
          <p className="text-xs text-neutral-500">A short descriptive name that will appear in the auction list.</p>
        </div>
        <div className="space-y-1">
          <label className="text-sm text-neutral-300">Start price (USD)</label>
          <div className="flex">
            <span className="inline-flex items-center px-3 border border-neutral-800 rounded-l bg-neutral-900">$</span>
            <input
              type="number"
              min={0}
              step="0.01"
              placeholder="1.00"
              value={form.startPrice}
              onChange={(e) => setForm({ ...form, startPrice: Number(e.target.value) })}
              className="px-3 py-2 rounded-r border border-l-0 border-neutral-800 w-full"
            />
          </div>
          <p className="text-xs text-neutral-500">Opening price shown when the auction starts.</p>
        </div>
        <div className="space-y-1">
          <label className="text-sm text-neutral-300">Minimum increment (USD)</label>
          <div className="flex">
            <span className="inline-flex items-center px-3 border border-neutral-800 rounded-l bg-neutral-900">$</span>
            <input
              type="number"
              min={0.01}
              step="0.01"
              placeholder="0.25"
              value={form.minIncrement}
              onChange={(e) => setForm({ ...form, minIncrement: Number(e.target.value) })}
              className="px-3 py-2 rounded-r border border-l-0 border-neutral-800 w-full"
            />
          </div>
          <p className="text-xs text-neutral-500">The smallest amount each new bid must increase by.</p>
        </div>
        <div className="space-y-1">
          <label className="text-sm text-neutral-300">Duration (seconds)</label>
          <input
            type="number"
            min={10}
            step={1}
            placeholder="60"
            value={form.durationSeconds}
            onChange={(e) => setForm({ ...form, durationSeconds: Number(e.target.value) })}
            className="px-3 py-2 rounded border border-neutral-800 w-full"
          />
          <p className="text-xs text-neutral-500">How long the auction runs.</p>
        </div>
        <div className="space-y-1">
          <label className="text-sm text-neutral-300">Soft close window (seconds)</label>
          <input
            type="number"
            min={0}
            step={1}
            placeholder="10"
            value={form.softCloseSeconds}
            onChange={(e) => setForm({ ...form, softCloseSeconds: Number(e.target.value) })}
            className="px-3 py-2 rounded border border-neutral-800 w-full"
          />
          <p className="text-xs text-neutral-500">
            If a bid arrives within this window, the timer extends by the same amount (anti‑sniping).
          </p>
        </div>
        <div className="space-y-1">
          <label className="text-sm text-neutral-300">Reserve price (optional, USD)</label>
          <div className="flex">
            <span className="inline-flex items-center px-3 border border-neutral-800 rounded-l bg-neutral-900">$</span>
            <input
              type="number"
              min={0}
              step="0.01"
              placeholder="0.00"
              value={form.reservePrice}
              onChange={(e) => setForm({ ...form, reservePrice: Number(e.target.value) })}
              className="px-3 py-2 rounded-r border border-l-0 border-neutral-800 w-full"
            />
          </div>
          <p className="text-xs text-neutral-500">Minimum price required to sell. Leave at 0 for no reserve.</p>
        </div>
        <div className="md:col-span-3">
          <button
            disabled={!canCreate}
            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 rounded"
          >
            Create auction
          </button>
          {!canCreate && <span className="ml-3 text-xs text-red-400">Enter a title to enable create.</span>}
        </div>
      </form>
    </section>
  );
}

