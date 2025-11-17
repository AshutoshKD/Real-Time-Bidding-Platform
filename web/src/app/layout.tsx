import "./../styles/globals.css";
import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "RTB Demo",
  description: "Real-time bidding demo with Go + WebRTC",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <div className="min-h-screen">
          <header className="border-b border-neutral-800 sticky top-0 bg-neutral-950/70 backdrop-blur">
            <div className="max-w-6xl mx-auto px-4 py-3 flex items-center justify-between">
              <h1 className="text-lg font-semibold">
                <Link href="/" className="hover:underline">
                  RTB Demo
                </Link>
              </h1>
              <div className="text-xs text-neutral-400">
                Go (Pion WebRTC) + Next.js
              </div>
            </div>
          </header>
          <main className="max-w-6xl mx-auto px-4 py-6">{children}</main>
        </div>
      </body>
    </html>
  );
}


