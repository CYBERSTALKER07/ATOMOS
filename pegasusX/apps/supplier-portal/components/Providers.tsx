"use client";

import type { ReactNode } from "react";
import SupplierShell from "./SupplierShell";
import { ThemeProvider } from "./ThemeProvider";
import { DesktopCacheBootstrap } from "@/lib/desktop-cache-bootstrap";
import { DesktopDeepLinkBootstrap } from "@/lib/desktop-deep-link";
import { PortalOfflineTray } from "@/lib/portal-offline-tray";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <DesktopCacheBootstrap />
      <DesktopDeepLinkBootstrap />
      <div className="app-root min-h-screen w-full">
        <SupplierShell>{children}</SupplierShell>
        <PortalOfflineTray />
      </div>
    </ThemeProvider>
  );
}
