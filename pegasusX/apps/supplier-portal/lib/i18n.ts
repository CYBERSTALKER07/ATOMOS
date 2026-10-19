"use client";

import { useSyncExternalStore } from "react";
import {
  createTranslator,
  LOCALE_CHANGE_EVENT,
  resolveActiveLocale,
  type Locale,
} from "@pegasusx/i18n";

function subscribe(onStoreChange: () => void) {
  if (typeof window === "undefined") {
    return () => {};
  }
  window.addEventListener(LOCALE_CHANGE_EVENT, onStoreChange);
  return () => window.removeEventListener(LOCALE_CHANGE_EVENT, onStoreChange);
}

function getSnapshot(): Locale {
  return resolveActiveLocale();
}

function getServerSnapshot(): Locale {
  return "en";
}

/** Portal chrome translator — reacts to LanguageSwitcher / LocaleBootstrap. */
export function usePortalLocale(): Locale {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

export function usePortalT() {
  const locale = usePortalLocale();
  return createTranslator(locale);
}
