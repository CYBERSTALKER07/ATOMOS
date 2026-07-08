"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { subscribeDesktopDeepLinks } from "@pegasusx/desktop-bridge";

/** Routes `pegasusx-retailer://…` handoff links into the Next.js app. */
export function DesktopDeepLinkBootstrap() {
  const router = useRouter();

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    subscribeDesktopDeepLinks((path) => {
      router.push(path);
    }).then((unsub: () => void) => {
      cleanup = unsub;
    });
    return () => {
      if (cleanup) cleanup();
    };
  }, [router]);

  return null;
}
