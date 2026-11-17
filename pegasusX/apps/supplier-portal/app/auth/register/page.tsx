"use client";

import { usePortalT } from "@/lib/i18n";
import { useMemo, useState } from "react";
import Link from "next/link";
import type { INNLookupResponse, JoinRequestConflictPayload } from "@pegasusx/types";
import {
  AuthRegisterShell,
  AuthRegisterIdentityStep,
  AuthRegisterVerificationStep,
  AuthRegisterProfileStep,
  dialCodeForCountry,
  FormAlert,
} from "@/components/ui";
import { createSupplierApi } from "@/lib/api";
import { persistSession, supplierFetch } from "@/lib/auth";
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

export default function RegisterPage() {
  const t = usePortalT();
  const api = useMemo(() => createSupplierApi(), []);
  const [state, setState] = useState<WizardState>(INITIAL_STATE);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [verifyingInn, setVerifyingInn] = useState(false);
  const [innLookup, setInnLookup] = useState<INNLookupResponse | null>(null);
  const [innError, setInnError] = useState<string | null>(null);
  const [joinPending, setJoinPending] = useState<JoinRequestConflictPayload | null>(null);
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
        setSubmitError(err instanceof Error ? err.message : t("supplier_portal.residual.text.failed_to_send_verification_code"));
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
        setSubmitError(err instanceof Error ? err.message : t("supplier_portal.residual.text.invalid_verification_code"));
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

  async function verifyInn(): Promise<INNLookupResponse | null> {
    const cleaned = state.profile.taxId.replace(/\D/g, "");
    if (cleaned.length !== 9) {
      setErrors((prev) => ({
        ...prev,
        taxId: t("supplier_portal.onboarding_inn.invalid_inn"),
      }));
      return null;
    }
    setVerifyingInn(true);
    setInnError(null);
    setJoinPending(null);
    setErrors((prev) => ({ ...prev, taxId: "" }));
    try {
      const res = await api.lookupPlatformINN({
        inn: cleaned,
        tenant_type: "SUPPLIER",
      });
      setInnLookup(res);
      if (res.status === "UNAVAILABLE") {
        setInnError(t("supplier_portal.onboarding_inn.registry_unavailable"));
        return null;
      }
      if (res.status !== "FOUND") {
        setInnError(t("supplier_portal.onboarding_inn.invalid_inn"));
        return null;
      }
      if (res.legal_name && !state.profile.legalName.trim()) {
        setState((s) => ({
          ...s,
          profile: { ...s.profile, legalName: res.legal_name ?? s.profile.legalName },
        }));
      }
      return res;
    } catch (err) {
      setInnError(
        err instanceof Error
          ? err.message
          : t("supplier_portal.onboarding_inn.registry_unavailable"),
      );
      return null;
    } finally {
      setVerifyingInn(false);
    }
  }

  async function submit() {
    const e = validateCurrent();
    setErrors(e);
    if (Object.keys(e).length > 0) return;

    const cleanedInn = state.profile.taxId.replace(/\D/g, "");
    if (
      !innLookup ||
      innLookup.status !== "FOUND" ||
      innLookup.normalized_inn !== cleanedInn
    ) {
      const verified = await verifyInn();
      if (!verified) return;
    }

    setSubmitting(true);
    setSubmitError(null);
    setJoinPending(null);
    try {
      const phone = `${dialCode}${state.identity.phoneLocal}`;
      const payload = {
        legal_name: state.profile.legalName.trim(),
        contact_name: state.profile.contactName.trim(),
        email: state.profile.email.trim(),
        phone,
        password: state.profile.password,
        market_code: state.identity.countryCode,
        tax_id: cleanedInn,
      };
      const res = await supplierFetch("/v1/platform/tenants/register", {
        method: "POST",
        headers: {
          "Idempotency-Key": cryptoRandomId(),
        },
        body: JSON.stringify(payload),
      });
      if (res.status === 409) {
        const conflict = (await res.json().catch(() => null)) as JoinRequestConflictPayload | null;
        if (conflict?.request_id) {
          setJoinPending(conflict);
          return;
        }
        throw new Error((conflict as { error?: string } | null)?.error || "Registration conflict");
      }
      if (!res.ok) {
        const body = await res.text();
        throw new Error(body || `register failed: ${res.status}`);
      }
      const data = (await res.json()) as { token?: string; next_step?: string };
      if (data.token) {
        persistSession(data.token);
      }
      window.location.href = data.next_step || "/setup/business";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  const alertBanner = joinPending ? (
    <FormAlert variant="info">
      <span className="flex flex-col gap-1">
        <strong>{t("supplier_portal.onboarding_inn.existing_company_title")}</strong>
        <span>
          {t("supplier_portal.onboarding_inn.existing_company_message", {
            legal_name: joinPending.legal_name || state.profile.legalName || "Registered Supplier",
            inn: joinPending.normalized_inn || state.profile.taxId,
          })}
        </span>
        <span className="text-xs font-mono">
          {t("supplier_portal.onboarding_inn.join_request_sent")} (#{joinPending.request_id})
        </span>
        <Link href="/auth/login" className="underline text-xs mt-1">
          {t("supplier_portal.join_requests.status_pending")}
        </Link>
      </span>
    </FormAlert>
  ) : submitError ? (
    <FormAlert variant="error">{submitError}</FormAlert>
  ) : null;

  return (
    <AuthRegisterShell
      title={t("supplier_portal.auth.register.text.set_up_your_supplier_account")}
      subtitle={`Step ${stepIndex + 1} of ${STEP_ORDER.length} — ${STEP_LABELS[state.step]}`}
      stepOrder={STEP_ORDER}
      stepLabels={STEP_LABELS}
      currentIndex={stepIndex}
      error={alertBanner}
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
              {submitting ? "Creating…" : "Create supplier"}
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
          setProfile={(value) => {
            const nextProfile = typeof value === "function" ? value(state.profile) : value;
            if (nextProfile.taxId !== state.profile.taxId) {
              setInnLookup(null);
              setInnError(null);
              setJoinPending(null);
            }
            setState((s) => ({
              ...s,
              profile: typeof value === "function" ? value(s.profile) : value,
            }));
          }}
          errors={errors}
          onVerifyInn={() => void verifyInn()}
          verifyingInn={verifyingInn}
          innLookup={innLookup}
          innError={innError}
          taxIdLabel={t("supplier_portal.onboarding_inn.tax_id_label")}
          taxIdPlaceholder={t("supplier_portal.onboarding_inn.tax_id_placeholder")}
          verifyButtonLabel={t("supplier_portal.onboarding_inn.verify_button")}
          verifyingLabel={t("supplier_portal.onboarding_inn.verifying")}
          verifiedBadgeLabel={t("supplier_portal.onboarding_inn.verified_badge")}
          existingCompanyHint={
            innLookup?.existing_join_eligible
              ? t("supplier_portal.onboarding_inn.existing_company_message", {
                  legal_name: innLookup.existing_legal_name || innLookup.legal_name || "",
                  inn: innLookup.normalized_inn,
                })
              : null
          }
        />
      )}
    </AuthRegisterShell>
  );
}

function cryptoRandomId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2) + Date.now().toString(36);
}
