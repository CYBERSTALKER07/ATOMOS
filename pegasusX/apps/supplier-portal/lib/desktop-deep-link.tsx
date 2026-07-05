"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { subscribeDesktopDeepLinks } from "@pegasusx/desktop-bridge";

/** Routes `pegasusx-supplier://…` handoff links into the Next.js app. */
export function DesktopDeepLinkBootstrap() {
  const router = useRouter();

  useEffect(() => {
    void subscribeDesktopDeepLinks((path) => {
      router.push(path as any);
    });
  }, [router]);

  return null;
}
