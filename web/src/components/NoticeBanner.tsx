"use client";

import { useEffect, useState } from "react";

export default function NoticeBanner() {
  const [fading, setFading] = useState(false);
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    const startFade = setTimeout(() => setFading(true), 30000); // 30s
    const hide = setTimeout(() => setVisible(false), 30000 + 800); // allow fade-out
    return () => {
      clearTimeout(startFade);
      clearTimeout(hide);
    };
  }, []);

  if (!visible) return null;

  return (
    <div
      className={[
        "bg-red-950 border border-red-800 text-red-300 rounded-md p-3",
        "transition-opacity duration-700",
        fading ? "opacity-0" : "opacity-100",
      ].join(" ")}
    >
      Note: If the backend was just started, please wait ~50 seconds for it to warm up.
    </div>
  );
}


