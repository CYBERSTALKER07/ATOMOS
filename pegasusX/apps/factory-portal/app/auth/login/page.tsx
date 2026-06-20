"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard } from "@pegasusx/ui-kit/auth";
import { dialCodeForCountry } from "@pegasusx/ui-kit/auth";
import { persistSession, factoryApiBaseUrl } from "@/lib/auth";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";

type LoginStep = "phone" | "otp";

export default function FactoryLoginPage() {
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
    setLoading(true);
    try {
      await sendPhoneOtp(`${dialCode}${phoneLocal}`);
      setStep("otp");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to send verification code");
    } finally {
      setLoading(false);
    }
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
      const idToken = await verifyPhoneOtp(otpCode);
      const res = await fetch(`${factoryApiBaseUrl}/v1/auth/factory/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id_token: idToken }),
      });

      if (!res.ok) {
        if (res.status === 404) {
          router.push(`/auth/register?phone=${encodeURIComponent(phone)}`);
          return;
        }
        const errorData = await res.json().catch(() => null);
        throw new Error(errorData?.message || errorData?.error || "Login failed");
      }

      const data = await res.json();
      persistSession(data.token, data.refresh_token);
      router.replace(data.is_configured ? "/" : "/setup/factory");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLoginCard
      title="Factory sign in"
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
      registerPrompt="New factory?"
      registerLabel="Register"
      onCountryChange={setCountryCode}
      onPhoneChange={setPhoneLocal}
      onOtpChange={setOtpCode}
      onSendOtp={handleSendOtp}
      onVerifyOtp={handleVerifyOtp}
      onBack={() => {
        resetPhoneOtpFlow();
        setStep("phone");
      }}
    />
  );
}
