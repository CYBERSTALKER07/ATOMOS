"use client";

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
import { persistSession, factoryApiBaseUrl } from "@/lib/auth";
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

export default function FactoryRegisterPage() {
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
        await sendPhoneOtp(`${dialCode}${state.identity.phoneLocal}`);
        setState((s) => ({
          ...s,
          step: "verification",
          verification: { otpCode: "", idToken: "" },
        }));
      } catch (err) {
        setSubmitError(err instanceof Error ? err.message : "Failed to send verification code");
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
        setSubmitError(err instanceof Error ? err.message : "Invalid verification code");
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
      const payload = {
        account: {
          legalName: state.profile.legalName,
          contactName: state.profile.contactName,
          email: state.profile.email,
          country: state.identity.countryCode,
          phone,
        },
        id_token: state.verification.idToken,
      };

      const res = await fetch(`${factoryApiBaseUrl}/v1/auth/factory/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const errorData = await res.json().catch(() => null);
        throw new Error(errorData?.message || "Registration failed");
      }

      const data = await res.json();
      persistSession(data.token, data.refresh_token);
      router.replace(data.is_configured ? "/" : "/setup/factory");
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthRegisterShell
      title="Set up your factory account"
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
              {submitting ? "Creating…" : "Create factory"}
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
