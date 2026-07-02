"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { subscribeDesktopDeepLinks } from "@pegasusx/desktop-bridge";

/** Routes `pegasusx-supplier://…` handoff links into the Next.js app. */
export function DesktopDeepLinkBootstrap() {
  const router = useRouter();

  useEffect(() => {
    return subscribeDesktopDeepLinks((path) => {
      router.push(path);
    });
  }, [router]);

  return null;
}
