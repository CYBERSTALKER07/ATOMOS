"use client";

import { usePathname, useRouter } from "next/navigation";
import { WebSocketProvider } from "../../lib/ws";
import { SessionReconcileListener } from "../../lib/session-reconcile-listener";
import { PendingCheckoutFlusher } from "../../lib/pending-checkout-flusher";
import { DesktopCacheBootstrap } from "../../lib/desktop-cache-bootstrap";
import { DesktopDeepLinkBootstrap } from "../../lib/desktop-deep-link";
import { NotificationsProvider } from "../../lib/notifications";
import { clearStoredToken } from "../../lib/bridge";
import { CartProvider } from "../../lib/cart";
import RetailerShell from "../../components/RetailerShell";
import ClientPolicyBanner from "../../components/ClientPolicyBanner";
import PaymentModal from "../../components/PaymentModal";
import ShopClosedModal from "../../components/ShopClosedModal";
import { RetailerOfflineTray } from "../../lib/retailer-offline-tray";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <WebSocketProvider>
      <SessionReconcileListener />
      <DesktopCacheBootstrap />
      <DesktopDeepLinkBootstrap />
      <PendingCheckoutFlusher />
      <NotificationsProvider>
        <CartProvider>
          <RetailerShell>
            <ClientPolicyBanner />
            {children}
          </RetailerShell>
          <PaymentModal />
          <ShopClosedModal />
          <RetailerOfflineTray />
        </CartProvider>
      </NotificationsProvider>
    </WebSocketProvider>
  );
}