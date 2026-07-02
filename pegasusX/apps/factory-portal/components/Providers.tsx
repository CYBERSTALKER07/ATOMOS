"use client";

import type { ReactNode } from "react";
import { Suspense } from "react";
import AuthGuard from "./AuthGuard";
import FactoryShell from "./FactoryShell";
import { PageSkeleton } from "./Skeleton";
import { ToastProvider } from "./Toast";
import { ThemeProvider } from "./ThemeProvider";
import { PortalOfflineTray } from "@/lib/portal-offline-tray";
import { DesktopDeepLinkBootstrap } from "@/lib/desktop-deep-link";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <DesktopDeepLinkBootstrap />
      <div className="app-root min-h-screen w-full">
        <AuthGuard>
          <FactoryShell>
            <ToastProvider>
              <Suspense fallback={<PageSkeleton />}>
                {children}
              </Suspense>
            </ToastProvider>
          </FactoryShell>
        </AuthGuard>
        <PortalOfflineTray />
      </div>
    </ThemeProvider>
  );
}
