"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { readTokenFromCookie } from "@/lib/auth";
import { clearStoredToken, getStoredToken, isTauri } from "@/lib/bridge";

/**
 * Decode JWT payload without verification (matches middleware.ts logic).
 */
function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = atob(parts[1].replace(/-/g, "+").replace(/_/g, "/"));
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

function isTokenExpired(token: string): boolean {
  const payload = decodeJwtPayload(token);
  if (!payload || typeof payload.exp !== "number") return true;
  return payload.exp * 1000 < Date.now();
}

const AUTH_ROUTES = ["/auth/login", "/auth/register", "/login", "/signup"];

/**
 * Client-side auth guard. Required for Tauri SSG export where Edge middleware
 * doesn't run. Also provides belt-and-suspenders protection on web.
 *
 * - Unauthenticated users on protected routes → redirect to /auth/login
 * - Authenticated users on auth routes → redirect to /
 */
export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    async function checkAuth() {
      let token = readTokenFromCookie();

      // Desktop: try OS keychain if cookie is empty
      if (!token && isTauri()) {
        const storedToken = await getStoredToken();
        if (storedToken && !isTokenExpired(storedToken)) {
          // Write to cookie so the rest of the app (apiFetch, etc.) picks it up
          document.cookie = `pegasus_admin_jwt=${encodeURIComponent(storedToken)}; path=/; max-age=86400; SameSite=Lax`;
          token = storedToken;
        } else if (storedToken) {
          clearStoredToken().catch(() => {});
        }
      }

      const hasValidToken = !!token && !isTokenExpired(token);
      const isAuthRoute = AUTH_ROUTES.some(
        (r) => pathname === r || pathname.startsWith("/auth/")
      );

      if (isAuthRoute && hasValidToken) {
        // Already authenticated — go to dashboard
        router.replace("/");
        return;
      }

      if (!isAuthRoute && !hasValidToken) {
        // Not authenticated — clear stale cookies and redirect
        if (token && isTokenExpired(token)) {
          document.cookie = "pegasus_admin_jwt=; Max-Age=0; path=/";
          document.cookie = "pegasus_supplier_jwt=; Max-Age=0; path=/";
          if (isTauri()) {
            clearStoredToken().catch(() => {});
          }
        }
        router.replace("/auth/login");
        return;
      }

      setChecked(true);
    }

    checkAuth();
  }, [pathname, router]);

  // Don't render until auth check completes (prevents flash of wrong content)
  if (!checked) return null;

  return <>{children}</>;
}
