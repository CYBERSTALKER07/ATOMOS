"use client";

import { usePortalT } from "@/lib/i18n";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard } from "@pegasusx/ui-kit/auth";
import { dialCodeForCountry } from "@pegasusx/ui-kit/auth";
import { persistSession, warehouseApiBaseUrl } from "@/lib/auth";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";
import { COUNTRIES } from "../register/wizard-state";

type LoginStep = "phone" | "otp";

export default function WarehouseLoginPage() {
  const t = usePortalT();
  const router = useRouter();
  const [step, setStep] = useState<LoginStep>("phone");
  const [countryCode, setCountryCode] = useState("UZ");
  const [phoneLocal, setPhoneLocal] = useState("");
  const [otpCode, setOtpCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const dialCode = useMemo(() => dialCodeForCountry(countryCode) || COUNTRIES.find((c) => c.code === countryCode)?.dialCode || "", [countryCode]);

  async function handleSendOtp(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!/^\d{6,14}$/.test(phoneLocal)) {
      setError(t("warehouse_portal.residual.text.enter_a_valid_phone_number_6_14_digits"));
      return;
    }
    setLoading(true);
    try {
      await sendPhoneOtp(`${dialCode}${phoneLocal}`);
      setStep("otp");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("warehouse_portal.residual.text.failed_to_send_verification_code"));
    } finally {
      setLoading(false);
    }
  }

  async function handleVerifyOtp(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!/^\d{6}$/.test(otpCode)) {
      setError(t("warehouse_portal.residual.text.enter_the_6_digit_code"));
      return;
    }

    setLoading(true);
    try {
      const phone = `${dialCode}${phoneLocal}`;
      const idToken = await verifyPhoneOtp(otpCode);
      const res = await fetch(`${warehouseApiBaseUrl}/v1/auth/warehouse/login`, {
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
      router.replace(data.is_configured ? "/" : "/setup/location");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.error.login_failed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLoginCard
      title={t("warehouse_portal.auth.login.text.warehouse_sign_in")}
      subtitle={
        step === "phone"
          ? t("warehouse_portal.auth.login.text.enter_registered_phone")
          : t("warehouse_portal.auth.login.text.enter_otp_sent_to", { phone: `${dialCode}${phoneLocal}` })
      }
      step={step}
      countryCode={countryCode}
      phoneLocal={phoneLocal}
      otpCode={otpCode}
      error={error}
      loading={loading}
      registerHref="/auth/register"
      registerPrompt={t("warehouse_portal.residual.text.new_warehouse")}
      registerLabel={t("warehouse_portal.residual.text.register")}
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
