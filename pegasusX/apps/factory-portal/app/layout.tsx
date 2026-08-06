"use client";

import { usePortalT } from "@/lib/i18n";
import type { Metadata } from "next";
import { Plus_Jakarta_Sans, EB_Garamond } from "next/font/google";
import LocaleBootstrap from "../components/LocaleBootstrap";
import Providers from "../components/Providers";
import "./globals.css";

const fontJakarta = Plus_Jakarta_Sans({
  variable: "--font-sans",
  subsets: ["latin"],
  weight: ["300", "400", "500", "600", "700"],
});

const fontGaramond = EB_Garamond({
  variable: "--font-garamond",
  subsets: ["latin"],
  weight: ["400", "500"],
});

export const metadata: Metadata = {
  title: t("factory_portal.residual.text.factory_portal_pegasus"),
  description: t("factory_portal.residual.text.factory_loading_bay_transfer_management_and_dispatch_operations"),
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const t = usePortalT();
  return (
    <html lang="en" suppressHydrationWarning className={`${fontJakarta.variable} ${fontGaramond.variable}`}>
      <head>
        <script dangerouslySetInnerHTML={{ __html: `
          (function(){try{var m=localStorage.getItem('pegasus-factory-theme-mode');
          var d=m==='dark'||(m!=='light'&&matchMedia('(prefers-color-scheme:dark)').matches);
          var r=document.documentElement;r.classList.toggle('dark',d);r.style.colorScheme=d?'dark':'light';
          if(window.__TAURI_INTERNALS__)r.setAttribute('data-tauri','');}catch(e){}})();
        `}} />
      </head>
      <body
        className={`${fontJakarta.variable} ${fontGaramond.variable} font-sans min-h-screen bg-background text-foreground`}
      >
        <LocaleBootstrap />
        <div id="app-splash" aria-hidden="true">
          <div
            className="flex h-20 w-20 items-center justify-center rounded-2xl"
            style={{ background: "var(--desk-accent)", color: "var(--desk-accent-on)" }}
          >
            <svg width="40" height="40" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M20 4H4v2h16V4zm1 10v-2l-1-5H4l-1 5v2h1v6h10v-6h4v6h2v-6h1zm-9 4H6v-4h6v4z" />
            </svg>
          </div>
        </div>

        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
