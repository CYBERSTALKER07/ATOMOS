"use client";

import { useEffect } from "react";
import type { Route } from "next";
import { useRouter } from "next/navigation";
import { subscribeDesktopDeepLinks } from "@pegasusx/desktop-bridge";

/** Routes `pegasusx-supplier://…` handoff links into the Next.js app. */
export function DesktopDeepLinkBootstrap() {
  const router = useRouter();

  useEffect(() => {
    void subscribeDesktopDeepLinks((path) => {
      // typedRoutes: deep-link paths arrive as runtime strings; Route cast is
      // the narrowest honest assertion for dynamic navigation targets.
      router.push(path as Route);
    });
  }, [router]);

  return null;
}
