"use client";

import type { ReactNode } from "react";
import WarehouseShell from "./WarehouseShell";
import { ThemeProvider } from "./ThemeProvider";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <WarehouseShell>{children}</WarehouseShell>
    </ThemeProvider>
  );
}
