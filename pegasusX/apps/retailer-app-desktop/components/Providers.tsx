"use client";

import type { ReactNode } from "react";
import PageTransition from "./PageTransition";
import { ThemeProvider } from "./ThemeProvider";

export default function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <div className="app-root min-h-screen w-full">
        <PageTransition>{children}</PageTransition>
      </div>
    </ThemeProvider>
  );
}
