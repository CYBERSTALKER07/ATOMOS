"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, useMemo } from "react";
import { createSupplierApi } from "@/lib/api";
import { persistSession } from "@/lib/auth";
import { PortalField, PortalInput, PortalSelect, FormAlert } from "@/components/portal";
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
    [countryCode],
  );

  async function handleSendOtp(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!/^\d{6,14}$/.test(phoneLocal)) {
      setError("Enter a valid phone number (6-14 digits)");
      return;
    }
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
      const api = createSupplierApi();
      const resp = await api.loginSupplier({ phone, password: otpCode });
      if (resp.token) {
        persistSession(resp.token, resp.refresh_token);
      }
      router.replace(resp.is_configured ? "/dashboard" : "/setup/billing");
    } catch (err: unknown) {
      const apiErr = err as { status?: number };
      if (apiErr?.status === 404) {
        router.push(`/auth/register?phone=${encodeURIComponent(dialCode + phoneLocal)}`);
        return;
      }
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

      {error ? <FormAlert variant="error">{error}</FormAlert> : null}

      {step === "phone" ? (
        <form onSubmit={handleSendOtp} className="space-y-4">
          <div className="grid grid-cols-[120px,1fr] gap-3">
            <PortalField id="countryCode" label="Country">
              <PortalSelect
                id="countryCode"
                value={countryCode}
                onChange={(e) => setCountryCode(e.target.value)}
              >
                {COUNTRIES.map((c) => (
                  <option key={c.code} value={c.code}>
                    {c.name}
                  </option>
                ))}
              </PortalSelect>
            </PortalField>
            <PortalField id="phoneLocal" label="Phone" hint={`Will be sent as ${dialCode}${phoneLocal || "…"}`}>
              <div className="flex">
                <span
                  className="inline-flex items-center px-3 border border-r-0 rounded-l text-sm portal-input"
                  style={{
                    width: "auto",
                    minHeight: 44,
                    borderTopRightRadius: 0,
                    borderBottomRightRadius: 0,
                    background: "var(--desk-surface-subtle)",
                  }}
                >
                  {dialCode}
                </span>
                <PortalInput
                  id="phoneLocal"
                  type="tel"
                  inputMode="numeric"
                  style={{ borderTopLeftRadius: 0, borderBottomLeftRadius: 0 }}
                  value={phoneLocal}
                  onChange={(e) => setPhoneLocal(e.target.value.replace(/\D/g, ""))}
                  required
                />
              </div>
            </PortalField>
          </div>
          <div id="recaptcha-container" />
          <button type="submit" className="portal-btn portal-btn--primary w-full" disabled={loading}>
            Continue
          </button>
        </form>
      ) : (
        <form onSubmit={handleVerifyOtp} className="space-y-4">
          <PortalField id="otpCode" label="Verification code">
            <PortalInput
              id="otpCode"
              type="text"
              inputMode="numeric"
              className="tracking-widest text-lg font-mono text-center"
              maxLength={6}
              value={otpCode}
              onChange={(e) => setOtpCode(e.target.value.replace(/\D/g, ""))}
              required
            />
          </PortalField>
          <div className="flex gap-3">
            <button type="button" className="portal-btn portal-btn--ghost w-full" onClick={() => setStep("phone")} disabled={loading}>
              Back
            </button>
            <button type="submit" className="portal-btn portal-btn--primary w-full" disabled={loading}>
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
