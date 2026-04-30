import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Suspense } from "react";
import "./globals.css";
import FactoryShell from "../components/FactoryShell";
import AuthGuard from "../components/AuthGuard";
import { PageSkeleton } from "../components/Skeleton";
import { ToastProvider } from "../components/Toast";
import { ThemeProvider } from "../components/ThemeProvider";
import PageTransition from "../components/PageTransition";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Factory Portal — Pegasus",
  description: "Factory loading bay, transfer management, and dispatch operations.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: `
          (function(){try{var m=localStorage.getItem('pegasus-factory-theme-mode')||localStorage.getItem('lab-factory-theme-mode');
          if(m==='dark'||(m!=='light'&&matchMedia('(prefers-color-scheme:dark)').matches))
          document.documentElement.classList.add('dark')}catch(e){}})();
        `}} />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} font-sans flex h-screen overflow-hidden bg-background text-foreground`}
      >
        <div id="app-splash" aria-hidden="true">
          <img src="/logo-solid-square.png" alt="" width="80" height="80" />
        </div>

        <ThemeProvider>
          <AuthGuard>
            <FactoryShell>
              <ToastProvider>
                <Suspense fallback={<PageSkeleton />}>
                  <PageTransition>
                    {children}
                  </PageTransition>
                </Suspense>
              </ToastProvider>
            </FactoryShell>
          </AuthGuard>
        </ThemeProvider>
      </body>
    </html>
  );
}
