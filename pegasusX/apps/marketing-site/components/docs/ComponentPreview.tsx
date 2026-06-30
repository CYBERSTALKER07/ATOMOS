"use client";

import { useState, type ReactNode } from "react";

type ComponentPreviewProps = {
  children: ReactNode;
};

export function ComponentPreview({ children }: ComponentPreviewProps) {
  const [theme, setTheme] = useState<"dark" | "light">("dark");

  return (
    <div className="space-y-3">
      <div className="flex gap-2">
        <button
          type="button"
          className={`rounded-full px-3 py-1 text-xs font-medium ${theme === "dark" ? "bg-white text-black" : "border border-[var(--mkt-border)]"}`}
          onClick={() => setTheme("dark")}
        >
          Dark
        </button>
        <button
          type="button"
          className={`rounded-full px-3 py-1 text-xs font-medium ${theme === "light" ? "bg-white text-black" : "border border-[var(--mkt-border)]"}`}
          onClick={() => setTheme("light")}
        >
          Light
        </button>
      </div>
      <div
        className={`rounded-xl border border-[var(--mkt-border)] p-6 ${theme === "light" ? "doc-preview-light" : ""}`}
      >
        {children}
      </div>
    </div>
  );
}
