"use client";

import { createContext, useContext, useState, useCallback, type ReactNode } from "react";

// Minimal bearer-token session for the PLATFORM_ADMIN console. The token is
// minted out-of-band (break-glass governance identity) and held in memory only.
const TokenCtx = createContext<{ token: string; setToken: (t: string) => void }>({
  token: "",
  setToken: () => {},
});

export function AdminSessionProvider({ children }: { children: ReactNode }) {
  const [token, setTok] = useState("");
  const setToken = useCallback((t: string) => setTok(t.trim()), []);
  return <TokenCtx.Provider value={{ token, setToken }}>{children}</TokenCtx.Provider>;
}

export function useAdminToken() {
  return useContext(TokenCtx);
}
