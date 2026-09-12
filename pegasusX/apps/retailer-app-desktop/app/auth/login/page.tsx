"use client";

import { usePortalT } from "@/lib/i18n";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard } from "@pegasusx/ui-kit/auth";
import { dialCodeForCountry } from "@pegasusx/ui-kit/auth";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

type LoginStep = "phone" | "otp";

export default function RetailerLoginPage() {
  const t = usePortalT();
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
      setError(t("retailer_desktop.residual.text.enter_a_valid_phone_number_6_14_digits"));
      return;
    }
    setLoading(true);
    try {
      await sendPhoneOtp(`${dialCode}${phoneLocal}`);
      setStep("otp");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.failed_to_send_verification_code"));
    } finally {
      setLoading(false);
    }
  }

  async function handleVerifyOtp(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!/^\d{6}$/.test(otpCode)) {
      setError(t("retailer_desktop.residual.text.enter_the_6_digit_code"));
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

      // C1.3 multi-org: intermediate token → org picker (flag on server only).
      const { isPendingOrgSelectResponse, persistPendingOrgToken, stashPendingMemberships, applyFullAuthResponse } =
        await import("@/lib/multi-org-auth");
      if (isPendingOrgSelectResponse(data)) {
        persistPendingOrgToken(data.token, data.expires_in_sec ?? 420);
        if (Array.isArray(data.memberships)) {
          stashPendingMemberships(data.memberships);
        }
        router.replace("/auth/select-org");
        return;
      }

      await applyFullAuthResponse(data, { clearScoped: false });
      router.replace(data.is_configured ? "/dashboard" : "/setup/tax");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.error.login_failed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLoginCard
      title={t("retailer_desktop.auth.login.text.retailer_sign_in")}
      subtitle={
        step === "phone"
          ? t("retailer_desktop.auth.login.text.enter_registered_phone")
          : t("retailer_desktop.auth.login.text.enter_otp_sent_to", { phone: `${dialCode}${phoneLocal}` })
      }
      step={step}
      countryCode={countryCode}
      phoneLocal={phoneLocal}
      otpCode={otpCode}
      error={error}
      loading={loading}
      registerHref="/auth/register"
      registerPrompt={t("retailer_desktop.residual.text.new_retailer")}
      registerLabel={t("retailer_desktop.residual.text.register")}
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
