"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getStoredToken, isTauri, storeToken } from "../lib/bridge";
import { readToken } from "../lib/auth";
import { motion, AnimatePresence } from "framer-motion";
import { Button } from "@heroui/react";
import {
  ShieldCheck,
  Phone,
  Lock,
  Loader2,
  ChevronRight,
  AlertTriangle,
  WifiOff,
} from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function Home() {
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [restoringSession, setRestoringSession] = useState(true);
  const [isOnline, setIsOnline] = useState(() =>
    typeof navigator === "undefined" ? true : navigator.onLine,
  );
  const router = useRouter();

  useEffect(() => {
    let active = true;
    const restoreSession = async () => {
      try {
        const cookieToken = readToken();
        if (cookieToken) {
          router.replace("/dashboard");
          return;
        }
        if (!isTauri()) return;
        const storedToken = await getStoredToken();
        if (!storedToken) return;
        document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(storedToken)}; path=/; max-age=86400; SameSite=Lax`;
        router.replace("/dashboard");
      } finally {
        if (active) {
          setRestoringSession(false);
        }
      }
    };
    void restoreSession();
    return () => {
      active = false;
    };
  }, [router]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const onOnline = () => setIsOnline(true);
    const onOffline = () => setIsOnline(false);
    window.addEventListener("online", onOnline);
    window.addEventListener("offline", onOffline);
    return () => {
      window.removeEventListener("online", onOnline);
      window.removeEventListener("offline", onOffline);
    };
  }, []);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!isOnline) {
      setError("Network offline. Reconnect and retry authentication.");
      return;
    }

    const normalizedPhone = phone.trim().replace(/\s+/g, "");
    if (!/^\+?\d{9,15}$/.test(normalizedPhone)) {
      setError("Enter a valid phone number in international format.");
      return;
    }

    if (password.trim().length < 6) {
      setError("Password must be at least 6 characters.");
      return;
    }

    setLoading(true);

    try {
      const res = await fetch(`${API}/v1/auth/retailer/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ phone_number: normalizedPhone, password }),
      });

      if (!res.ok) {
        if (res.status === 401 || res.status === 403) {
          throw new Error("Authentication failed. Check credentials.");
        }
        if (res.status >= 500) {
          throw new Error("Authentication service unavailable. Retry shortly.");
        }
        throw new Error(`Authentication failed (${res.status}).`);
      }
      const data = await res.json();
      if (!data?.token) throw new Error("Login response is missing token");

      document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      await storeToken(data.token, data.refresh_token || "");
      if (data.user)
        localStorage.setItem("retailer_profile", JSON.stringify(data.user));

      router.replace("/dashboard");
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "An error occurred";
      if (/Failed to fetch|NetworkError|Load failed/i.test(message)) {
        setError("Network request failed. Verify connectivity and retry.");
      } else {
        setError(message);
      }
    } finally {
      setLoading(false);
    }
  };

  const canSubmit =
    !loading &&
    !restoringSession &&
    isOnline &&
    phone.trim().length > 0 &&
    password.length > 0;

  return (
    <div className="min-h-dvh w-full flex items-center justify-center p-6 bg-[var(--desk-canvas)]">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="w-full max-w-md bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-[32px] shadow-2xl overflow-hidden"
      >
        <div className="p-10">
          <div className="flex flex-col items-center text-center mb-10">
            <div className="w-16 h-16 rounded-2xl bg-[var(--desk-accent)] flex items-center justify-center text-white shadow-xl mb-6 rotate-3 hover:rotate-0 transition-transform cursor-default">
              <ShieldCheck size={32} />
            </div>
            <h1 className="md-typescale-display-small font-bold text-[var(--desk-text-primary)] tracking-tight">
              Retailer Hub
            </h1>
            <p className="mt-2 md-typescale-body-large text-[var(--desk-text-secondary)]">
              Secure network portal entry
            </p>
          </div>

          <form onSubmit={handleLogin} className="space-y-6">
            <AnimatePresence>
              {restoringSession && (
                <motion.div
                  initial={{ opacity: 0, y: -6 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -6 }}
                  className="p-3 rounded-xl bg-[var(--desk-info)]/10 border border-[var(--desk-info)]/30 text-[var(--desk-info)] text-xs font-bold text-center"
                >
                  Restoring secure session...
                </motion.div>
              )}
            </AnimatePresence>

            <AnimatePresence>
              {!isOnline && (
                <motion.div
                  initial={{ opacity: 0, y: -6 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -6 }}
                  className="p-3 rounded-xl bg-[var(--desk-warning)]/10 border border-[var(--desk-warning)]/30 text-[var(--desk-warning)] text-xs font-bold text-center flex items-center justify-center gap-2"
                >
                  <WifiOff size={14} />
                  Network offline. Login is temporarily unavailable.
                </motion.div>
              )}
            </AnimatePresence>

            <div className="space-y-1.5">
              <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                Protocol Handle
              </label>
              <div className="relative group">
                <Phone
                  className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors"
                  size={18}
                />
                <input
                  type="tel"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  placeholder="+998"
                  className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                  required
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                Access Cipher
              </label>
              <div className="relative group">
                <Lock
                  className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors"
                  size={18}
                />
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                  required
                />
              </div>
            </div>

            <AnimatePresence>
              {error && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  className="p-3 rounded-xl bg-red-50 border border-red-100 text-red-600 text-xs font-bold text-center flex items-center justify-center gap-2"
                >
                  <AlertTriangle size={14} />
                  {error}
                </motion.div>
              )}
            </AnimatePresence>

            <Button
              type="submit"
              isDisabled={!canSubmit}
              className="w-full h-14 bg-[var(--desk-text-primary)] text-white font-bold rounded-2xl shadow-xl flex items-center justify-center gap-3 transition-all hover:scale-[1.02] active:scale-95 disabled:opacity-30"
            >
              {loading || restoringSession ? (
                <Loader2 size={20} className="animate-spin" />
              ) : (
                <>
                  Initialize Connection <ChevronRight size={20} />
                </>
              )}
            </Button>
          </form>
        </div>

        <div className="px-10 py-6 bg-[var(--desk-surface-subtle)] border-t border-[var(--desk-border)] text-center">
          <p className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
            Pegasus Logistics Core v2.0.0
          </p>
        </div>
      </motion.div>
    </div>
  );
}
