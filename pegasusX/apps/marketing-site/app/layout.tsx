import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Footer } from "@/components/layout/Footer";
import { ScrollProgress } from "@/components/layout/ScrollProgress";
import { LenisProvider } from "@/components/motion/LenisProvider";
import { ReducedMotionProvider } from "@/components/motion/ReducedMotionProvider";
import { GsapRouteCleanup } from "@/components/motion/GsapRouteCleanup";
import { GlitchLoader } from "@/components/void/GlitchLoader";
import Providers from "@/components/Providers";
import "./globals.css";

const fontGeist = Geist({
  variable: "--font-geist",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800", "900"],
});

const fontGeistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Pegasus — Logistics operating system",
  description:
    "Run supplier-led logistics from one platform — dispatch, fleet tracking, payments, and coordination across six roles.",
  openGraph: {
    title: "Pegasus — Logistics operating system",
    description:
      "Connect suppliers, warehouses, factories, drivers, and retailers with live dispatch, tracking, and payments.",
    type: "website",
    siteName: "Pegasus",
  },
  twitter: {
    card: "summary_large_image",
    title: "Pegasus — Logistics operating system",
    description:
      "Run supplier-led logistics from one platform.",
  },
  robots: { index: true, follow: true },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`dark ${fontGeist.variable} ${fontGeistMono.variable}`}
      suppressHydrationWarning
    >
      <head>
        <link rel="icon" href="/icon.svg" type="image/svg+xml" />
      </head>
      <body className="min-h-dvh bg-black font-sans text-white antialiased">
        <Providers>
          <ReducedMotionProvider>
            <LenisProvider>
              <GsapRouteCleanup />
              <GlitchLoader />
              <ScrollProgress />
              <div className="flex min-h-dvh flex-col pt-[var(--void-nav-offset)]">
                <div id="main-content" tabIndex={-1} className="flex-1 outline-none">
                  {children}
                </div>
                <Footer />
              </div>
            </LenisProvider>
          </ReducedMotionProvider>
        </Providers>
      </body>
    </html>
  );
}
