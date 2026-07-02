"use client";

import { useEffect } from "react";
import { initDesktopCache } from "@pegasusx/desktop-cache";

/** Opens SQLite on Tauri startup for offline dispatch preview cache. */
export function DesktopCacheBootstrap() {
  useEffect(() => {
    void initDesktopCache();
  }, []);

  return null;
}
