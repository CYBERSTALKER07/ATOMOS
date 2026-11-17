"use client";

import {
  supportedLocales,
  syncDocumentLocale,
  type Locale,
} from "@pegasusx/i18n";
import { usePortalLocale, usePortalT } from "@/lib/i18n";

/** Compact en/ru/uz switcher for portal chrome (sidebar footer). */
export default function LanguageSwitcher() {
  const t = usePortalT();
  const locale = usePortalLocale();

  return (
    <label className="desk-sidebar-item" style={{ gap: 8, cursor: "pointer" }}>
      <span className="sr-only">{t("portal.chrome.language")}</span>
      <select
        value={locale}
        aria-label={t("portal.chrome.language")}
        onChange={(event) => {
          syncDocumentLocale(event.target.value as Locale);
        }}
        style={{
          flex: 1,
          minWidth: 0,
          border: "1px solid var(--desk-border)",
          borderRadius: "var(--radius-sm, 6px)",
          background: "var(--desk-surface)",
          color: "var(--desk-text-primary)",
          font: "var(--type-caption-sm, 12px/16px var(--desktop-font-sans, system-ui))",
          padding: "6px 8px",
        }}
      >
        {supportedLocales.map((code) => (
          <option key={code} value={code}>
            {t(`app.language.${code}`)}
          </option>
        ))}
      </select>
    </label>
  );
}
