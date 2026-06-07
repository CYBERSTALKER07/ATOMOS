"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, useMemo } from "react";
import { createSupplierApi } from "@/lib/api";
import { persistSession } from "@/lib/auth";
import { COUNTRIES } from "../register/wizard-state";

type LoginStep = "phone" | "otp";

export default function SupplierLoginPage() {
  const router = useRouter();
  const [step, setStep] = useState<LoginStep>("phone");
  const [countryCode, setCountryCode] = useState("UZ");
  const [phoneLocal, setPhoneLocal] = useState("");
  const [otpCode, setOtpCode] = useState("");
  
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const dialCode = useMemo(
    () => COUNTRIES.find((c) => c.code === countryCode)?.dialCode ?? "",
    [countryCode]
  );

  async function handleSendOtp(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!/^\d{6,14}$/.test(phoneLocal)) {
      setError("Enter a valid phone number (6-14 digits)");
      return;
    }
    // TODO: In a real implementation, trigger Firebase Recaptcha and send OTP here
    setStep("otp");
  }

  async function handleVerifyOtp(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!/^\d{6}$/.test(otpCode)) {
      setError("Enter the 6-digit code");
      return;
    }
    
    setLoading(true);
    try {
      const phone = `${dialCode}${phoneLocal}`;
      // Here otpCode is mimicking the firebase ID token for scaffold purposes
      const api = createSupplierApi();
      const resp = await api.loginSupplier({ phone, id_token: otpCode });
      if (resp.token) {
        persistSession(resp.token, resp.refresh_token);
      }
      router.replace(resp.is_configured ? "/dashboard" : "/setup/billing");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-card space-y-5">
      <div>
        <h1 className="md-typescale-headline-medium" style={{ margin: 0 }}>
          Supplier sign in
        </h1>
        <p className="desk-page-subtitle">
          {step === "phone" ? "Enter your registered phone number." : `Enter the 6-digit code sent to ${dialCode}${phoneLocal}`}
        </p>
      </div>

      {error && (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-danger)" }}>
          {error}
        </p>
      )}

      {step === "phone" ? (
        <form onSubmit={handleSendOtp} className="space-y-4">
          <div className="grid grid-cols-[120px,1fr] gap-3">
            <label className="block space-y-1">
              <span className="md-typescale-label-medium">Country</span>
              <select
                className="md-input-outlined"
                value={countryCode}
                onChange={(e) => setCountryCode(e.target.value)}
              >
                {COUNTRIES.map((c) => (
                  <option key={c.code} value={c.code}>{c.name}</option>
                ))}
              </select>
            </label>
            <label className="block space-y-1">
              <span className="md-typescale-label-medium">Phone</span>
              <div className="flex">
                <span
                  className="inline-flex items-center px-3 border border-r-0 rounded-l text-sm"
                  style={{
                    borderColor: "var(--color-md-outline)",
                    background: "var(--color-md-surface-container-high)",
                  }}
                >
                  {dialCode}
                </span>
                <input
                  type="tel"
                  inputMode="numeric"
                  className="md-input-outlined"
                  style={{ borderTopLeftRadius: 0, borderBottomLeftRadius: 0 }}
                  value={phoneLocal}
                  onChange={(e) => setPhoneLocal(e.target.value.replace(/\D/g, ""))}
                  required
                />
              </div>
            </label>
          </div>
          <div id="recaptcha-container"></div>
          <button type="submit" className="md-btn md-btn-filled w-full" disabled={loading}>
            Continue
          </button>
        </form>
      ) : (
        <form onSubmit={handleVerifyOtp} className="space-y-4">
          <label className="block space-y-1">
            <span className="md-typescale-label-medium">Verification Code</span>
            <input
              type="text"
              inputMode="numeric"
              className="md-input-outlined tracking-widest text-lg font-mono text-center"
              maxLength={6}
              value={otpCode}
              onChange={(e) => setOtpCode(e.target.value.replace(/\D/g, ""))}
              required
            />
          </label>
          <div className="flex gap-3">
            <button type="button" className="md-btn md-btn-text w-full" onClick={() => setStep("phone")} disabled={loading}>
              Back
            </button>
            <button type="submit" className="md-btn md-btn-filled w-full" disabled={loading}>
              {loading ? "Signing in…" : "Sign in"}
            </button>
          </div>
        </form>
      )}

      <p className="md-typescale-body-small text-center" style={{ color: "var(--desk-text-secondary)" }}>
        New supplier?{" "}
        <Link href="/auth/register" className="underline">
          Register
        </Link>
      </p>
    </div>
  );
}
