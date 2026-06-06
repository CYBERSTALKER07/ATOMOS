import type { Metadata } from "next";
import { Inter, EB_Garamond } from "next/font/google";
import Image from "next/image";
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
          <Image src="/logo-solid-square.png" alt="" width={80} height={80} priority />
        </div>

        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
