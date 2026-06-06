"use client";

import AuthGuard from "@/components/AuthGuard";
import { ToastProvider } from "@/components/Toast";
import SupplierShell from "@/components/SupplierShell";

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard>
      <ToastProvider>
        <SupplierShell>{children}</SupplierShell>
      </ToastProvider>
    </AuthGuard>
  );
}
