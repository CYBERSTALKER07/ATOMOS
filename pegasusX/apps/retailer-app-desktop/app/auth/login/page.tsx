"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getStoredToken, isTauri, storeToken } from "../../../lib/bridge";
import { readToken } from "../../../lib/auth";
import { motion, AnimatePresence } from "framer-motion";
import {
  ShieldCheck,
  Phone,
  KeyRound,
  Loader2,
  ChevronRight,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import Link from "next/link";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

type LoginStep = "phone" | "otp";

export default function RetailerLoginPage() {
  const [step, setStep] = useState<LoginStep>("phone");
  const [phone, setPhone] = useState("");
  const [otpCode, setOtpCode] = useState("");
  
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

  const handleSendOtp = async (e: React.FormEvent) => {
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

    // TODO: Trigger Firebase Recaptcha and send OTP here
    setStep("otp");
  };

  const handleVerifyOtp = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!/^\d{6}$/.test(otpCode)) {
      setError("Enter the 6-digit code.");
      return;
    }

    setLoading(true);

    try {
      const normalizedPhone = phone.trim().replace(/\s+/g, "");
      
      const res = await fetch(`${API}/v1/auth/retailer/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ phone_number: normalizedPhone, id_token: otpCode }), // using otpCode as fallback id_token scaffold
      });

      if (!res.ok) {
        if (res.status === 404) {
          router.push(`/auth/register?phone=${encodeURIComponent(normalizedPhone)}`);
          return;
        }
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
      if (data.refresh_token) {
        document.cookie = `pegasus_retailer_refresh=${encodeURIComponent(data.refresh_token)}; path=/; max-age=604800; SameSite=Lax`;
      }
      await storeToken(data.token, data.refresh_token || "");
      if (data.user)
        localStorage.setItem("retailer_profile", JSON.stringify(data.user));

      router.replace(data.is_configured ? "/dashboard" : "/setup/tax");
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

  const canSubmitPhone =
    !loading &&
    !restoringSession &&
    isOnline &&
    phone.trim().length > 0;

  const canSubmitOtp =
    !loading &&
    isOnline &&
    otpCode.trim().length === 6;

  return (
    <>
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

      <AnimatePresence mode="wait">
            {step === "phone" ? (
              <motion.form 
                key="phone-form"
                onSubmit={handleSendOtp} 
                className="space-y-6"
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
              >
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

                <div id="recaptcha-container"></div>

                <button
                  type="submit"
                  disabled={!canSubmitPhone}
                  className="portal-btn portal-btn--primary w-full h-14 font-bold rounded-2xl shadow-xl flex items-center justify-center gap-3 transition-all hover:scale-[1.02] active:scale-95 disabled:opacity-30"
                >
                  {loading || restoringSession ? (
                    <Loader2 size={20} className="animate-spin" />
                  ) : (
                    <>
                      Initialize Connection <ChevronRight size={20} />
                    </>
                  )}
                </button>

                <p className="md-typescale-body-small text-center" style={{ color: "var(--desk-text-secondary)" }}>
                  New retailer?{" "}
                  <Link href="/auth/register" className="font-bold underline hover:text-[var(--desk-text-primary)]">
                    Register
                  </Link>
                </p>
              </motion.form>
            ) : (
              <motion.form 
                key="otp-form"
                onSubmit={handleVerifyOtp} 
                className="space-y-6"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: 20 }}
              >
                <div className="space-y-1.5">
                  <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                    Verification Code
                  </label>
                  <div className="relative group">
                    <KeyRound
                      className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors"
                      size={18}
                    />
                    <input
                      type="text"
                      inputMode="numeric"
                      value={otpCode}
                      onChange={(e) => setOtpCode(e.target.value.replace(/\D/g, ""))}
                      maxLength={6}
                      placeholder="••••••"
                      className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold tracking-[0.5em] text-center"
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

                <div className="flex gap-3">
                  <button
                    type="button"
                    onClick={() => setStep("phone")}
                    disabled={loading}
                    className="portal-btn portal-btn--ghost flex-1 h-14 font-bold rounded-2xl transition-all hover:scale-[1.02] active:scale-95 disabled:opacity-30"
                  >
                    Back
                  </button>
                  <button
                    type="submit"
                    disabled={!canSubmitOtp}
                    className="portal-btn portal-btn--primary flex-1 h-14 font-bold rounded-2xl shadow-xl transition-all hover:scale-[1.02] active:scale-95 disabled:opacity-30"
                  >
                    {loading ? <Loader2 size={20} className="animate-spin" /> : "Verify & Connect"}
                  </button>
                </div>
              </motion.form>
            )}
          </AnimatePresence>
    </>
  );
}
