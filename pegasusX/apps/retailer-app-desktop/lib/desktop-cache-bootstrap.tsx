"use client";

import { useEffect } from "react";
import { initDesktopCache } from "@pegasusx/desktop-cache";
import { initRetailerProfile } from "./retailer-profile";

/** Opens SQLite and hydrates retailer profile from cache on Tauri startup. */
export function DesktopCacheBootstrap() {
  useEffect(() => {
    void (async () => {
      await initDesktopCache();
      await initRetailerProfile();
    })();
  }, []);

  return null;
}
