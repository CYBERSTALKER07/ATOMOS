import type { Metadata } from "next";
import Image from "next/image";
import "./globals.css";
import LocaleBootstrap from "../components/LocaleBootstrap";

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
        <script dangerouslySetInnerHTML={{ __html: `
          (function(){try{var m=localStorage.getItem('pegasus-retailer-theme-mode');
          if(m==='dark'||(m!=='light'&&matchMedia('(prefers-color-scheme: dark)').matches))
          document.documentElement.classList.add('dark')}catch(e){}})();
        `}} />
      </head>
      <body
        className="font-sans antialiased min-h-screen"
        style={{ background: "var(--background)", color: "var(--foreground)" }}
      >
        <LocaleBootstrap />
        <div id="app-splash" aria-hidden="true">
          <Image src="/logo-solid-square.png" alt="" width={80} height={80} priority />
        </div>
        {children}
      </body>
    </html>
  );
}
