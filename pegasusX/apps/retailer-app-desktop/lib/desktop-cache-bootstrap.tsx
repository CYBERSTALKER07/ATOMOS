"use client";

import { useEffect } from "react";
import { initDesktopCache } from "@pegasusx/desktop-cache";

/** Opens SQLite and runs migrations on Tauri startup. */
export function DesktopCacheBootstrap() {
  useEffect(() => {
    void initDesktopCache();
  }, []);

  return null;
}
