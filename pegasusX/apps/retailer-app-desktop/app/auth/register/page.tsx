"use client";

import { usePortalT } from "@/lib/i18n";
import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  AuthRegisterShell,
  AuthRegisterIdentityStep,
  AuthRegisterVerificationStep,
  AuthRegisterProfileStep,
} from "@pegasusx/ui-kit/auth";
import { dialCodeForCountry } from "@pegasusx/ui-kit/auth";
import { FormAlert } from "@pegasusx/ui-kit/portal";
import { storeToken } from "@/lib/bridge";
import { sessionMapCenter } from "@pegasusx/api-client";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";
import {
  INITIAL_STATE,
  STEP_LABELS,
  STEP_ORDER,
  type WizardState,
  validateIdentity,
  validateVerification,
  validateProfile,
} from "./wizard-state";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

// 3-step Retailer onboarding wizard.
// Store location and tax setup live at /setup/* post-registration.

export default function RetailerRegisterPage() {
  const t = usePortalT();
  const router = useRouter();
  const [state, setState] = useState<WizardState>(INITIAL_STATE);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [stepBusy, setStepBusy] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const stepIndex = STEP_ORDER.indexOf(state.step);
  const dialCode = useMemo(
    () => dialCodeForCountry(state.identity.countryCode),
    [state.identity.countryCode],
  );

  function validateCurrent(): Record<string, string> {
    switch (state.step) {
      case "identity":
        return validateIdentity(state.identity);
      case "verification":
        return validateVerification(state.verification);
      case "profile":
        return validateProfile(state.profile);
    }
  }

  async function next() {
    const e = validateCurrent();
    setErrors(e);
    if (Object.keys(e).length > 0) return;

    if (state.step === "identity") {
      setStepBusy(true);
      setSubmitError(null);
      try {
        const phone = `${dialCode}${state.identity.phoneLocal}`;
        await sendPhoneOtp(phone);
        setState((s) => ({
          ...s,
          step: "verification",
          verification: { otpCode: "", idToken: "" },
        }));
      } catch (err) {
        setSubmitError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.failed_to_send_verification_code"));
      } finally {
        setStepBusy(false);
      }
      return;
    }

    if (state.step === "verification") {
      setStepBusy(true);
      setSubmitError(null);
      try {
        const idToken = await verifyPhoneOtp(state.verification.otpCode);
        setState((s) => ({
          ...s,
          step: "profile",
          verification: { ...s.verification, idToken },
        }));
      } catch (err) {
        setSubmitError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.invalid_verification_code"));
      } finally {
        setStepBusy(false);
      }
      return;
    }

    const ni = Math.min(stepIndex + 1, STEP_ORDER.length - 1);
    setState((s) => ({ ...s, step: STEP_ORDER[ni] }));
  }

  function back() {
    const pi = Math.max(stepIndex - 1, 0);
    if (state.step === "verification") {
      resetPhoneOtpFlow();
    }
    setState((s) => ({ ...s, step: STEP_ORDER[pi] }));
  }

  async function submit() {
    const e = validateCurrent();
    setErrors(e);
    if (Object.keys(e).length > 0) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      const phone = `${dialCode}${state.identity.phoneLocal}`;
      const registerRes = await fetch(`${API}/v1/auth/retailer/register`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": crypto.randomUUID(),
        },
        body: JSON.stringify({
          phone,
          name: state.profile.legalName.trim(),
          lat: sessionMapCenter()?.lat ?? 0,
          lng: sessionMapCenter()?.lng ?? 0,
          delivery_address: "Pending setup",
        }),
      });

      if (!registerRes.ok) {
        const errorData = await registerRes.json().catch(() => null);
        throw new Error(errorData?.error || "Registration failed");
      }

      const idToken = state.verification.idToken || (await verifyPhoneOtp(state.verification.otpCode));
      const loginRes = await fetch(`${API}/v1/auth/retailer/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ phone_number: phone, id_token: idToken }),
      });

      if (!loginRes.ok) {
        const errorData = await loginRes.json().catch(() => null);
        throw new Error(errorData?.error || "Login after registration failed");
      }

      const data = await loginRes.json();
      if (!data?.token) throw new Error("Login response is missing token");

      document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      if (data.refresh_token) {
        document.cookie = `pegasus_retailer_refresh=${encodeURIComponent(data.refresh_token)}; path=/; max-age=604800; SameSite=Lax`;
      }
      await storeToken(data.token, data.refresh_token || "");

      router.replace(data.is_configured ? "/dashboard" : "/setup/tax");
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : t("supplier_portal.auth.register.error.registration_failed"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthRegisterShell
      title={t("retailer_desktop.auth.register.text.set_up_your_retailer_account")}
      subtitle={`Step ${stepIndex + 1} of ${STEP_ORDER.length} — ${STEP_LABELS[state.step]}`}
      stepOrder={STEP_ORDER}
      stepLabels={STEP_LABELS}
      currentIndex={stepIndex}
      error={submitError ? <FormAlert variant="error">{submitError}</FormAlert> : null}
      footer={
        <>
          <button
            type="button"
            className="portal-btn portal-btn--ghost"
            onClick={back}
            disabled={stepIndex === 0 || submitting || stepBusy}
          >
            Back
          </button>
          {state.step !== "profile" ? (
            <button
              type="button"
              className="portal-btn portal-btn--primary"
              onClick={next}
              disabled={submitting || stepBusy}
            >
              {stepBusy ? "Please wait…" : "Continue"}
            </button>
          ) : (
            <button
              type="button"
              className="portal-btn portal-btn--primary"
              onClick={submit}
              disabled={submitting}
            >
              {submitting ? "Creating…" : "Create retailer"}
            </button>
          )}
        </>
      }
    >
      {state.step === "identity" && (
        <AuthRegisterIdentityStep
          identity={state.identity}
          setIdentity={(value) =>
            setState((s) => ({
              ...s,
              identity: typeof value === "function" ? value(s.identity) : value,
            }))
          }
          errors={errors}
          dialCode={dialCode}
        />
      )}
      {state.step === "verification" && (
        <AuthRegisterVerificationStep
          verification={state.verification}
          setVerification={(value) =>
            setState((s) => ({
              ...s,
              verification: typeof value === "function" ? value(s.verification) : value,
            }))
          }
          errors={errors}
        />
      )}
      {state.step === "profile" && (
        <AuthRegisterProfileStep
          profile={state.profile}
          setProfile={(value) =>
            setState((s) => ({
              ...s,
              profile: typeof value === "function" ? value(s.profile) : value,
            }))
          }
          errors={errors}
        />
      )}
    </AuthRegisterShell>
  );
}
