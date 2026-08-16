import { useEffect, useState } from "react";
import {
  cacheAuthSession,
  fetchAuthSession,
  readCachedAuthSession,
  type AuthSession,
  type MarketPack,
} from "./market-pack";

/** GS-R: load session pack after login. Empty token keeps cache (or null). */
export function useMarketPack(opts: { baseUrl: string; token: string }): {
  session: AuthSession | null;
  pack: MarketPack | null;
} {
  const [session, setSession] = useState<AuthSession | null>(() => readCachedAuthSession());

  useEffect(() => {
    const token = String(opts.token || "").trim();
    if (!token) {
      cacheAuthSession(null);
      setSession(null);
      return;
    }
    let cancelled = false;
    void fetchAuthSession(opts.baseUrl, token)
      .then((next) => {
        if (!cancelled) setSession(next);
      })
      .catch(() => {
        if (!cancelled) setSession(readCachedAuthSession());
      });
    return () => {
      cancelled = true;
    };
  }, [opts.baseUrl, opts.token]);

  return { session, pack: session?.pack ?? null };
}
