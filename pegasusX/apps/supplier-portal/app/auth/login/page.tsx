"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard } from "@pegasusx/ui-kit/auth";
import { dialCodeForCountry } from "@pegasusx/ui-kit/auth";
import { createSupplierApi } from "@/lib/api";
import { persistSession } from "@/lib/auth";

type LoginStep = "phone" | "otp";

export default function SupplierLoginPage() {
  const router = useRouter();
  const [step, setStep] = useState<LoginStep>("phone");
  const [countryCode, setCountryCode] = useState("UZ");
  const [phoneLocal, setPhoneLocal] = useState("");
  const [otpCode, setOtpCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const dialCode = useMemo(() => dialCodeForCountry(countryCode), [countryCode]);

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
    <AuthLoginCard
      title="Supplier sign in"
      subtitle={
        step === "phone"
          ? "Enter your registered phone number."
          : `Enter the 6-digit code sent to ${dialCode}${phoneLocal}`
      }
      step={step}
      countryCode={countryCode}
      phoneLocal={phoneLocal}
      otpCode={otpCode}
      error={error}
      loading={loading}
      registerHref="/auth/register"
      registerPrompt="New supplier?"
      registerLabel="Register"
      onCountryChange={setCountryCode}
      onPhoneChange={setPhoneLocal}
      onOtpChange={setOtpCode}
      onSendOtp={handleSendOtp}
      onVerifyOtp={handleVerifyOtp}
      onBack={() => setStep("phone")}
    />
  );
}
