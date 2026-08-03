import type { ReactNode } from "react";

/** Placeholder path for Next `output: "export"` / Tauri packaging. Runtime navigates by real IDs. */
export function generateStaticParams() {
  return [{ id: "_" }];
}

export default function Layout({ children }: { children: ReactNode }) {
  return children;
}
