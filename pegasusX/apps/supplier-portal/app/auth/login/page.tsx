"use client";

import { usePortalT } from "@/lib/i18n";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard } from "@pegasusx/ui-kit/auth";
import { dialCodeForCountry } from "@pegasusx/ui-kit/auth";
import { createSupplierApi } from "@/lib/api";
import { persistSession } from "@/lib/auth";
import { discoverOIDC } from "@/lib/oidc";

type LoginStep = "phone" | "otp";

export default function SupplierLoginPage() {
  const t = usePortalT();
  const router = useRouter();
  const [step, setStep] = useState<LoginStep>("phone");
  const [countryCode, setCountryCode] = useState("UZ");
  const [phoneLocal, setPhoneLocal] = useState("");
  const [otpCode, setOtpCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [companyId, setCompanyId] = useState("");
  const [idpLoading, setIdpLoading] = useState(false);

  const dialCode = useMemo(() => dialCodeForCountry(countryCode), [countryCode]);

  async function handleSendOtp(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!/^\d{6,14}$/.test(phoneLocal)) {
      setError(t("supplier_portal.residual.text.enter_a_valid_phone_number_6_14_digits"));
      return;
    }
    setStep("otp");
  }

  async function handleVerifyOtp(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!/^\d{6}$/.test(otpCode)) {
      setError(t("supplier_portal.residual.text.enter_the_6_digit_code"));
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
      setError(err instanceof Error ? err.message : t("auth.error.login_failed"));
    } finally {
      setLoading(false);
    }
  }

  async function handleIdP(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const sid = companyId.trim();
    if (!sid) {
      setError("Enter the company id to use SSO");
      return;
    }
    setIdpLoading(true);
    try {
      const nonce = crypto.randomUUID();
      sessionStorage.setItem("oidc_supplier_id", sid);
      sessionStorage.setItem("oidc_nonce", nonce);
      const disc = await discoverOIDC(sid, nonce);
      window.location.href = disc.authorization_url;
    } catch (err) {
      setError(err instanceof Error ? err.message : "IdP is not attached");
    } finally {
      setIdpLoading(false);
    }
  }

  return (
    <>
    <AuthLoginCard
      title={t("supplier_portal.auth.login.text.supplier_sign_in")}
      subtitle={
        step === "phone"
          ? t("supplier_portal.auth.login.text.enter_registered_phone")
          : t("supplier_portal.auth.login.text.enter_otp_sent_to", { phone: `${dialCode}${phoneLocal}` })
      }
      step={step}
      countryCode={countryCode}
      phoneLocal={phoneLocal}
      otpCode={otpCode}
      error={error}
      loading={loading}
      registerHref="/auth/register"
      registerPrompt={t("supplier_portal.residual.text.new_supplier")}
      registerLabel={t("supplier_portal.residual.text.register")}
      onCountryChange={setCountryCode}
      onPhoneChange={setPhoneLocal}
      onOtpChange={setOtpCode}
      onSendOtp={handleSendOtp}
      onVerifyOtp={handleVerifyOtp}
      onBack={() => setStep("phone")}
    />
    <form className="mx-auto mt-6 max-w-md space-y-2 px-4" onSubmit={handleIdP}>
      <p className="text-center text-sm text-[var(--desk-text-secondary)]">Or sign in with your company IdP</p>
      <input
        className="w-full rounded border px-3 py-2 text-sm"
        placeholder="Company id"
        value={companyId}
        onChange={(e) => setCompanyId(e.target.value)}
      />
      <button type="submit" disabled={idpLoading} className="w-full rounded border px-3 py-2 text-sm">
        {idpLoading ? "Opening IdP…" : "Login with IdP"}
      </button>
    </form>
    </>
  );
}
