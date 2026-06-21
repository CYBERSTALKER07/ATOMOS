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
  title: "Pegasus Retailer",
  description: "Retailer Desktop App for Pegasus",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning className={`${fontJakarta.variable} ${fontGaramond.variable}`}>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `
          (function(){try{var m=localStorage.getItem('pegasus-retailer-theme-mode');
          var d=m==='dark'||(m!=='light'&&matchMedia('(prefers-color-scheme: dark)').matches);
          var r=document.documentElement;r.classList.toggle('dark',d);r.style.colorScheme=d?'dark':'light';
          if(window.__TAURI_INTERNALS__)r.setAttribute('data-tauri','');}catch(e){}})();
        `,
          }}
        />
      </head>
      <body
        className={`${fontJakarta.variable} ${fontGaramond.variable} font-sans min-h-screen antialiased text-[var(--desk-text-primary)]`}
        style={{ background: "var(--desk-canvas)" }}
      >
        <LocaleBootstrap />
        <div id="app-splash" aria-hidden="true">
          <div
            className="flex h-20 w-20 items-center justify-center rounded-2xl"
            style={{ background: "var(--desk-accent)", color: "var(--desk-accent-on)" }}
          >
            <svg width="40" height="40" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M20 4H4v2h16V4zm-2 4H6v2h12V8zm-5 4H6v2h7v-2zm9 4v2H4v-2h16zm-2-4H6v-2h12v2z" />
            </svg>
          </div>
        </div>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
