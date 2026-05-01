"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { readTokenFromCookie, decodeJwtPayload } from "@/lib/auth";

function isTokenExpired(token: string): boolean {
  const payload = decodeJwtPayload(token);
  if (!payload || typeof payload.exp !== "number") return true;
  return payload.exp * 1000 < Date.now();
}

const AUTH_ROUTES = ["/auth/login"];

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    const token = readTokenFromCookie();
    const hasValidToken = !!token && !isTokenExpired(token);
    const isAuthRoute = AUTH_ROUTES.some(
      (r) => pathname === r || pathname.startsWith("/auth/")
    );

    if (isAuthRoute && hasValidToken) {
      router.replace("/");
      return;
    }

    if (!isAuthRoute && !hasValidToken) {
      if (token && isTokenExpired(token)) {
        document.cookie = "pegasus_warehouse_jwt=; Max-Age=0; path=/";
      }
      router.replace("/auth/login");
      return;
    }

    setChecked(true);
  }, [pathname, router]);

  if (!checked) {
    return (
      <div className="flex h-screen w-screen items-center justify-center bg-background">
        <div className="md-skeleton" style={{ width: 48, height: 48, borderRadius: 12 }} />
      </div>
    );
  }

  return <>{children}</>;
}
