"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import {
  clearSession,
  decodeJwtPayload,
  readIsConfigured,
  readTokenFromCookie,
  resolveSupplierToken,
} from "@/lib/auth";

const PUBLIC_PATHS = new Set([
  "/",
  "/auth/register",
  "/auth/login",
  "/setup/billing",
]);

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    void (async () => {
      if (PUBLIC_PATHS.has(pathname)) {
        setReady(true);
        return;
      }

      const token = (await resolveSupplierToken()) || readTokenFromCookie();
      if (!token) {
        router.replace("/auth/login");
        return;
      }

      const claims = decodeJwtPayload(token);
      if (!claims) {
        clearSession();
        router.replace("/auth/login");
        return;
      }

      if (!readIsConfigured(token) && pathname !== "/setup/billing") {
        router.replace("/setup/billing");
        return;
      }

      setReady(true);
    })();
  }, [pathname, router]);

  if (!ready) {
    return (
      <div className="min-h-screen flex items-center justify-center" style={{ background: "var(--color-md-surface)" }}>
        <p className="md-typescale-body-large" style={{ color: "var(--color-md-outline)" }}>
          Loading…
        </p>
      </div>
    );
  }

  return <>{children}</>;
}
