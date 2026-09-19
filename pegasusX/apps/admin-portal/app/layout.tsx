import type { Metadata } from "next";
import "./globals.css";
import { AdminSessionProvider } from "@/lib/session";

export const metadata: Metadata = {
  title: "PegasusX Admin Console",
  description: "Platform governance: tenant lifecycle, feature flags, audit",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <AdminSessionProvider>{children}</AdminSessionProvider>
      </body>
    </html>
  );
}
