"use client";

import type { ReactNode } from "react";
import { Suspense } from "react";
import AuthGuard from "./AuthGuard";
import FactoryShell from "./FactoryShell";
import { PageSkeleton } from "./Skeleton";
import { ToastProvider } from "./Toast";
import { ThemeProvider } from "./ThemeProvider";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <AuthGuard>
        <FactoryShell>
          <ToastProvider>
            <Suspense fallback={<PageSkeleton />}>
              {children}
            </Suspense>
          </ToastProvider>
        </FactoryShell>
      </AuthGuard>
    </ThemeProvider>
  );
}
