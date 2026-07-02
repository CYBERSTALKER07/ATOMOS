"use client";

import type { ReactNode } from "react";
import SupplierShell from "./SupplierShell";
import { ThemeProvider } from "./ThemeProvider";
import { DesktopCacheBootstrap } from "@/lib/desktop-cache-bootstrap";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <DesktopCacheBootstrap />
      <div className="app-root min-h-screen w-full">
        <SupplierShell>{children}</SupplierShell>
      </div>
    </ThemeProvider>
  );
}
