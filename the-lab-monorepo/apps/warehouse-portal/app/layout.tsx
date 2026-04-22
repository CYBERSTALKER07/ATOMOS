import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Suspense } from "react";
import "./globals.css";
import WarehouseShell from "../components/WarehouseShell";
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
  title: "Warehouse Portal — The Lab Industries",
  description: "Warehouse supply requests, demand forecasting, and dispatch operations.",
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
          (function(){try{var m=localStorage.getItem('lab-warehouse-theme-mode');
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
            <WarehouseShell>
              <ToastProvider>
                <Suspense fallback={<PageSkeleton />}>
                  <PageTransition>
                    {children}
                  </PageTransition>
                </Suspense>
              </ToastProvider>
            </WarehouseShell>
          </AuthGuard>
        </ThemeProvider>
      </body>
    </html>
  );
}
