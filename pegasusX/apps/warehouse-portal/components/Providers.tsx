"use client";

import type { ReactNode } from "react";
import { Suspense } from "react";
import WarehouseShell from "./WarehouseShell";
import { ThemeProvider } from "./ThemeProvider";
import { ToastProvider } from "./Toast";
import { PageSkeleton } from "./Skeleton";
import { DesktopCacheBootstrap } from "@/lib/desktop-cache-bootstrap";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <ToastProvider>
        <DesktopCacheBootstrap />
        <div className="app-root min-h-screen w-full">
          <WarehouseShell>
            <Suspense fallback={<PageSkeleton />}>
              {children}
            </Suspense>
          </WarehouseShell>
        </div>
      </ToastProvider>
    </ThemeProvider>
  );
}
