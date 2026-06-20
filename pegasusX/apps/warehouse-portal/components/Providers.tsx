"use client";

import type { ReactNode } from "react";
import WarehouseShell from "./WarehouseShell";
import { ThemeProvider } from "./ThemeProvider";
import { ToastProvider } from "./Toast";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <ToastProvider>
        <WarehouseShell>{children}</WarehouseShell>
      </ToastProvider>
    </ThemeProvider>
  );
}
