"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard } from "@pegasusx/ui-kit/auth";
import { dialCodeForCountry } from "@pegasusx/ui-kit/auth";
import { storeToken } from "@/lib/bridge";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

type LoginStep = "phone" | "otp";

export default function RetailerLoginPage() {
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
      const res = await fetch(`${API}/v1/auth/retailer/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ phone_number: phone, id_token: idToken }),
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
      if (!data?.token) throw new Error("Login response is missing token");

      document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      if (data.refresh_token) {
        document.cookie = `pegasus_retailer_refresh=${encodeURIComponent(data.refresh_token)}; path=/; max-age=604800; SameSite=Lax`;
      }
      await storeToken(data.token, data.refresh_token || "");
      if (data.user) localStorage.setItem("retailer_profile", JSON.stringify(data.user));

      router.replace(data.is_configured ? "/dashboard" : "/setup/tax");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLoginCard
      title="Retailer sign in"
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
      registerPrompt="New retailer?"
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
