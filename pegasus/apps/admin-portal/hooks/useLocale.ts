"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  createTranslator,
  defaultLocale,
  detectBrowserLocale,
  readStoredLocale,
  syncDocumentLocale,
  type Locale,
} from "@pegasus/i18n";

export function useLocale() {
  const [locale, setLocaleState] = useState<Locale>(defaultLocale);

  useEffect(() => {
    const resolvedLocale = readStoredLocale() ?? detectBrowserLocale();
    syncDocumentLocale(resolvedLocale);
    setLocaleState(resolvedLocale);
  }, []);

  const setLocale = useCallback((nextLocale: Locale) => {
    syncDocumentLocale(nextLocale);
    setLocaleState(nextLocale);
  }, []);

  const t = useMemo(() => createTranslator(locale), [locale]);

  return { locale, setLocale, t };
}
