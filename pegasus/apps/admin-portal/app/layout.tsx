import type { Metadata } from "next";
import { Inter, EB_Garamond } from "next/font/google";
import Image from "next/image";
import { Suspense } from "react";
import "./globals.css";
import AdminShell from "../components/AdminShell";
import AuthGuard from "../components/AuthGuard";
import LocaleBootstrap from "../components/LocaleBootstrap";
import { PageSkeleton } from "../components/Skeleton";
import { ToastProvider } from "../components/Toast";
import { ThemeProvider } from "../components/ThemeProvider";
import { NetworkStatusBanner } from "../components/NetworkStatusBanner";
import PageTransition from "../components/PageTransition";

const fontInter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const fontGaramond = EB_Garamond({
  variable: "--font-garamond",
  subsets: ["latin"],
  weight: ["400", "500"], // 300 is ideal per spec, usually 400 is lightest in some subsets, but Garamond supports 400+
});

export const metadata: Metadata = {
  title: "Admin Portal - Pegasus",
  description: "Live operational ledger and dispatch command center.",
  icons: {
    icon: [
      { url: "/favicon-32x32.png", sizes: "32x32", type: "image/png" },
      { url: "/favicon-16x16.png", sizes: "16x16", type: "image/png" },
    ],
    apple: "/apple-touch-icon.png",
  },
  manifest: "/manifest.json",
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
          if(m==='dark'||(m!=='light'&&matchMedia('(prefers-color-scheme:dark)').matches))
          document.documentElement.classList.add('dark')}catch(e){}})();
        `}} />
      </head>
      <body
        className={`${fontInter.variable} ${fontGaramond.variable} font-sans flex h-screen overflow-hidden bg-background text-foreground`}
      >
        <LocaleBootstrap />
        {/* Pre-hydration splash — rendered by the browser before React mounts */}
        <div id="app-splash" aria-hidden="true">
          <Image src="/logo-solid-square.png" alt="" width={80} height={80} priority />
        </div>

        <NetworkStatusBanner />
        <ThemeProvider>
          <AuthGuard>
            <AdminShell>
              <ToastProvider>
                <Suspense fallback={<PageSkeleton />}>
                  <PageTransition>
                    {children}
                  </PageTransition>
                </Suspense>
              </ToastProvider>
            </AdminShell>
          </AuthGuard>
        </ThemeProvider>
      </body>
    </html>
  );
}
