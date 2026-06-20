"use client";

import AuthGuard from "@/components/AuthGuard";
import { ToastProvider } from "@/components/Toast";

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard>
      <ToastProvider>{children}</ToastProvider>
    </AuthGuard>
  );
}
