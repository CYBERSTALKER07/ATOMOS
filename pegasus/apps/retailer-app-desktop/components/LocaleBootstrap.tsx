"use client";

import { useEffect } from "react";
import { bootstrapBrowserLocale } from "@pegasus/i18n/browser";

export default function LocaleBootstrap(): null {
  useEffect(() => {
    bootstrapBrowserLocale();

    const root = document.documentElement;
    root.setAttribute("data-hydrated", "");

    if ((window as unknown as Record<string, unknown>).__TAURI_INTERNALS__) {
      root.setAttribute("data-tauri", "");
    }
  }, []);

  return null;
}
