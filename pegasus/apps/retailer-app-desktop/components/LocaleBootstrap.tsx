"use client";

import { useEffect } from "react";
import { bootstrapBrowserLocale } from "@pegasus/i18n/browser";

const THEME_STORAGE_KEY = "pegasus-retailer-theme-mode";

type ThemeMode = "system" | "light" | "dark";

function readThemeMode(): ThemeMode {
  if (typeof window === "undefined") return "system";
  const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "system";
}

function resolveTheme(mode: ThemeMode): "light" | "dark" {
  if (mode === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }
  return mode;
}

function applyTheme(mode: ThemeMode): void {
  const root = document.documentElement;
  const resolved = resolveTheme(mode);
  root.classList.toggle("dark", resolved === "dark");
  root.style.colorScheme = resolved;
}

export default function LocaleBootstrap(): null {
  useEffect(() => {
    bootstrapBrowserLocale();

    const root = document.documentElement;
    root.setAttribute("data-hydrated", "");

    if ((window as unknown as Record<string, unknown>).__TAURI_INTERNALS__) {
      root.setAttribute("data-tauri", "");
    }

    applyTheme(readThemeMode());

    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleThemeChange = () => {
      if (readThemeMode() === "system") {
        applyTheme("system");
      }
    };

    mediaQuery.addEventListener("change", handleThemeChange);
    return () => mediaQuery.removeEventListener("change", handleThemeChange);
  }, []);

  return null;
}
