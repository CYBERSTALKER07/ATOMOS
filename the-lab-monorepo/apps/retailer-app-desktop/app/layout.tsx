import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "The Lab Retailer",
  description: "Retailer Desktop App for The Lab Industries",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="md-typescale-body-medium md-surface md-on-surface">  
        {children}
      </body>
    </html>
  );
}
