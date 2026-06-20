import type { Metadata } from "next";
import { Inter, EB_Garamond } from "next/font/google";
import Providers from "@/components/Providers";
import "./globals.css";

const fontInter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const fontGaramond = EB_Garamond({
  variable: "--font-garamond",
  subsets: ["latin"],
  weight: ["400", "500"],
});

export const metadata: Metadata = {
  title: "Supplier Portal - pegasusX",
  description: "Single-tenant logistics control plane.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning className={`${fontInter.variable} ${fontGaramond.variable}`}>
      <head>
        <script dangerouslySetInnerHTML={{ __html: `
          (function(){try{var m=localStorage.getItem('pegasus-theme-mode');
          var d=m==='dark'||(m!=='light'&&matchMedia('(prefers-color-scheme:dark)').matches);
          var r=document.documentElement;r.classList.toggle('dark',d);r.style.colorScheme=d?'dark':'light';}catch(e){}})();
        `}} />
      </head>
      <body
        className={`${fontInter.variable} ${fontGaramond.variable} font-sans flex h-screen overflow-hidden bg-background text-foreground`}
      >
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
