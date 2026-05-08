import type { Metadata } from "next";
import Image from "next/image";
import "./globals.css";
import LocaleBootstrap from "../components/LocaleBootstrap";
import { ThemeProvider } from "../components/ThemeProvider";
import PageTransition from "../components/PageTransition";

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
    <html lang="en" suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `
          (function(){try{var m=localStorage.getItem('pegasus-retailer-theme-mode');
          var d=m==='dark'||(m!=='light'&&matchMedia('(prefers-color-scheme: dark)').matches);
          var r=document.documentElement;r.classList.toggle('dark',d);r.style.colorScheme=d?'dark':'light';}catch(e){}})();
        `,
          }}
        />
      </head>
      <body
        className="font-sans antialiased flex h-screen min-h-screen overflow-hidden text-[var(--desk-text-primary)]"
        style={{ background: "var(--desk-canvas)" }}
      >
        <LocaleBootstrap />
        <div id="app-splash" aria-hidden="true">
          <Image
            src="/logo-solid-square.png"
            alt=""
            width={80}
            height={80}
            priority
          />
        </div>
        <ThemeProvider>
          <PageTransition>{children}</PageTransition>
        </ThemeProvider>
      </body>
    </html>
  );
}
