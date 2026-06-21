"use client";

import type { ReactNode } from "react";
import WarehouseShell from "./WarehouseShell";
import { ThemeProvider } from "./ThemeProvider";
import { ToastProvider } from "./Toast";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <ToastProvider>
        <div className="app-root min-h-screen w-full">
          <WarehouseShell>{children}</WarehouseShell>
        </div>
      </ToastProvider>
    </ThemeProvider>
  );
}
