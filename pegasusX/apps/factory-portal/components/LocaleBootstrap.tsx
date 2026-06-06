"use client";

import { useEffect } from "react";
import { bootstrapBrowserLocale } from "@pegasusx/i18n/browser";

export default function LocaleBootstrap(): null {
  useEffect(() => {
    bootstrapBrowserLocale();
  }, []);

  return null;
}
