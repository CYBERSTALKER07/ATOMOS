"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";

interface SessionContextType {
  token: string;
  setToken: (t: string) => void;
  isLoading: boolean;
  logout: () => Promise<void>;
  login: (subject?: string, password?: string, directToken?: string) => Promise<{ ok: boolean; error?: string }>;
}

const TokenCtx = createContext<SessionContextType>({
  token: "",
  setToken: () => {},
  isLoading: true,
  logout: async () => {},
  login: async () => ({ ok: false }),
});

export function AdminSessionProvider({ children }: { children: ReactNode }) {
  const [token, setTok] = useState("");
  const [isLoading, setIsLoading] = useState(true);

  // Hydrate session from HTTP-only cookie endpoint on mount
  useEffect(() => {
    let active = true;
    async function checkSession() {
      try {
        const res = await fetch("/api/auth/session", { cache: "no-store" });
        if (res.ok) {
          const data = await res.json();
          if (active && data.authenticated && data.token) {
            setTok(data.token);
          }
        }
      } catch {
        // Fallback to empty session
      } finally {
        if (active) setIsLoading(false);
      }
    }
    void checkSession();
    return () => {
      active = false;
    };
  }, []);

  const setToken = useCallback((t: string) => {
    const cleaned = t.trim();
    setTok(cleaned);
    if (cleaned) {
      // Sync to cookie
      void fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ directToken: cleaned }),
      });
    } else {
      void fetch("/api/auth/logout", { method: "POST" });
    }
  }, []);

  const login = useCallback(
    async (subject?: string, password?: string, directToken?: string) => {
      try {
        const res = await fetch("/api/auth/login", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ subject, password, directToken }),
        });
        const data = await res.json();
        if (res.ok && data.ok && data.token) {
          setTok(data.token);
          return { ok: true };
        }
        return { ok: false, error: data.error || "Login failed" };
      } catch (err) {
        return {
          ok: false,
          error: err instanceof Error ? err.message : "Network error during authentication",
        };
      }
    },
    [],
  );

  const logout = useCallback(async () => {
    setTok("");
    try {
      await fetch("/api/auth/logout", { method: "POST" });
    } catch {
      // Ignore
    }
  }, []);

  return (
    <TokenCtx.Provider value={{ token, setToken, isLoading, login, logout }}>
      {children}
    </TokenCtx.Provider>
  );
}

export function useAdminToken() {
  return useContext(TokenCtx);
}
