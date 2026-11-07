import type { Metadata } from "next";
import "./globals.css";
import { AdminSessionProvider } from "@/lib/session";
import { AdminWebSocketProvider } from "@/lib/ws-provider";
import { AdminAppShell } from "@/components/layout/AdminAppShell";

export const metadata: Metadata = {
  title: "PegasusX Admin Control Tower",
  description: "Platform Owner Governance: Outbox, Lot Recall, Scenarios, Cashier Vault, Fleet Safety",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="antialiased bg-gray-50 text-gray-900">
        <AdminSessionProvider>
          <AdminWebSocketProvider>
            <AdminAppShell>{children}</AdminAppShell>
          </AdminWebSocketProvider>
        </AdminSessionProvider>
      </body>
    </html>
  );
}
