"use client";

import { usePathname, useRouter } from "next/navigation";
import { WebSocketProvider, useWebSocket } from "../../lib/ws";
import { NotificationsProvider } from "../../lib/notifications";
import { clearStoredToken } from "../../lib/bridge";
import { CartProvider } from "../../lib/cart";
import RetailerShell from "../../components/RetailerShell";
import ClientPolicyBanner from "../../components/ClientPolicyBanner";
import PaymentModal from "../../components/PaymentModal";
import ShopClosedModal from "../../components/ShopClosedModal";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <WebSocketProvider>
      <NotificationsProvider>
        <CartProvider>
          <RetailerShell>
            <ClientPolicyBanner />
            {children}
          </RetailerShell>
          <PaymentModal />
          <ShopClosedModal />
        </CartProvider>
      </NotificationsProvider>
    </WebSocketProvider>
  );
}