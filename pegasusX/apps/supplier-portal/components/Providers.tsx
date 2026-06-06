"use client";

import type { ReactNode } from "react";
import SupplierShell from "./SupplierShell";
import { ThemeProvider } from "./ThemeProvider";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <SupplierShell>{children}</SupplierShell>
    </ThemeProvider>
  );
}
