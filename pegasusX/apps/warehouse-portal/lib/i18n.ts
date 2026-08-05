"use client";

import { useMemo } from "react";
import { createTranslator, detectBrowserLocale } from "@pegasusx/i18n";

/** Portal chrome translator — locale from cookie/browser via LocaleBootstrap. */
export function usePortalT() {
  return useMemo(() => createTranslator(detectBrowserLocale()), []);
}
