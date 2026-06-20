"use client";

import type { ChangeEvent, FormEvent, ReactNode } from "react";
import { PortalField, PortalInput, PortalSelect, FormAlert } from "../portal/PortalPrimitives";
import { AUTH_COUNTRIES } from "./countries";

export type AuthLoginStep = "phone" | "otp";

export function AuthLoginCard({
  title,
  subtitle,
  step,
  countryCode,
  phoneLocal,
  otpCode,
  error,
  loading,
  registerHref,
  registerPrompt = "New account?",
  registerLabel = "Register",
  onCountryChange,
  onPhoneChange,
  onOtpChange,
  onSendOtp,
  onVerifyOtp,
  onBack,
}: {
  title: string;
  subtitle: string;
  step: AuthLoginStep;
  countryCode: string;
  phoneLocal: string;
  otpCode: string;
  error: string | null;
  loading: boolean;
  registerHref: string;
  registerPrompt?: string;
  registerLabel?: string;
  onCountryChange: (code: string) => void;
  onPhoneChange: (local: string) => void;
  onOtpChange: (code: string) => void;
  onSendOtp: (e: FormEvent) => void;
  onVerifyOtp: (e: FormEvent) => void;
  onBack: () => void;
}) {
  const dialCode = AUTH_COUNTRIES.find((c) => c.code === countryCode)?.dialCode ?? "";

  return (
    <div className="auth-card space-y-5">
      <div>
        <h1 className="md-typescale-headline-medium" style={{ margin: 0 }}>
          {title}
        </h1>
        <p className="desk-page-subtitle">{subtitle}</p>
      </div>

      {error ? <FormAlert variant="error">{error}</FormAlert> : null}

      {step === "phone" ? (
        <form onSubmit={onSendOtp} className="space-y-4">
          <div className="grid grid-cols-[120px,1fr] gap-3">
            <PortalField id="countryCode" label="Country">
              <PortalSelect
                id="countryCode"
                value={countryCode}
                onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
                  onCountryChange(e.target.value)
                }
              >
                {AUTH_COUNTRIES.map((c) => (
                  <option key={c.code} value={c.code}>
                    {c.name}
                  </option>
                ))}
              </PortalSelect>
            </PortalField>
            <PortalField
              id="phoneLocal"
              label="Phone"
              hint={`Will be sent as ${dialCode}${phoneLocal || "…"}`}
            >
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
                  onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
                    onPhoneChange(e.target.value.replace(/\D/g, ""))
                  }
                  required
                />
              </div>
            </PortalField>
          </div>
          <div id="recaptcha-container" />
          <button type="submit" className="portal-btn portal-btn--primary w-full" disabled={loading}>
            {loading ? "Sending code…" : "Continue"}
          </button>
        </form>
      ) : (
        <form onSubmit={onVerifyOtp} className="space-y-4">
          <PortalField id="otpCode" label="Verification code">
            <PortalInput
              id="otpCode"
              type="text"
              inputMode="numeric"
              className="tracking-widest text-lg font-mono text-center"
              maxLength={6}
              value={otpCode}
              onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
                onOtpChange(e.target.value.replace(/\D/g, ""))
              }
              required
            />
          </PortalField>
          <div className="flex gap-3">
            <button
              type="button"
              className="portal-btn portal-btn--ghost w-full"
              onClick={onBack}
              disabled={loading}
            >
              Back
            </button>
            <button type="submit" className="portal-btn portal-btn--primary w-full" disabled={loading}>
              {loading ? "Signing in…" : "Sign in"}
            </button>
          </div>
        </form>
      )}

      <AuthLoginRegisterLink href={registerHref} prompt={registerPrompt} label={registerLabel} />
    </div>
  );
}

function AuthLoginRegisterLink({
  href,
  prompt,
  label,
}: {
  href: string;
  prompt: string;
  label: string;
}) {
  return (
    <p className="md-typescale-body-small text-center" style={{ color: "var(--desk-text-secondary)" }}>
      {prompt}{" "}
      <a href={href} className="underline">
        {label}
      </a>
    </p>
  );
}

export function AuthLoginRegisterFooter({
  href,
  prompt,
  label,
}: {
  href: string;
  prompt: string;
  label: string;
}) {
  return (
    <p className="md-typescale-body-small text-center" style={{ color: "var(--desk-text-secondary)" }}>
      {prompt}{" "}
      <a href={href} className="underline">
        {label}
      </a>
    </p>
  );
}
