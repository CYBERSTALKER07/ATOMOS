"use client";

import { usePathname, useRouter } from "next/navigation";
import { WebSocketProvider, useWebSocket } from "../../lib/ws";
import { clearStoredToken } from "../../lib/bridge";
import { CartProvider } from "../../lib/cart";
import RetailerShell from "../../components/RetailerShell";
import PaymentModal from "../../components/PaymentModal";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <WebSocketProvider>
      <CartProvider>
        <RetailerShell>{children}</RetailerShell>
        <PaymentModal />
      </CartProvider>
    </WebSocketProvider>
  );
}