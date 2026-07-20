"use client";

import type { ReactNode } from "react";
import PageTransition from "./PageTransition";
import { ThemeProvider } from "./ThemeProvider";
import { EnterpriseDesktopUpdateBootstrap } from "../lib/desktop-updater";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <EnterpriseDesktopUpdateBootstrap />
      <div className="app-root min-h-screen w-full">
        <PageTransition>{children}</PageTransition>
      </div>
    </ThemeProvider>
  );
}
