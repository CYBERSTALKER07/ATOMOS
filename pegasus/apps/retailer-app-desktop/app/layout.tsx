import type { Metadata } from "next";
import Image from "next/image";
import "./globals.css";
import LocaleBootstrap from "../components/LocaleBootstrap";
import { ThemeProvider } from "../components/ThemeProvider";

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
        className="font-sans antialiased flex h-screen min-h-screen overflow-hidden bg-background text-foreground"
      >
        <LocaleBootstrap />
        <div id="app-splash" aria-hidden="true">
          <Image src="/logo-solid-square.png" alt="" width={80} height={80} priority />
        </div>
        <ThemeProvider>{children}</ThemeProvider>
      </body>
    </html>
  );
}
