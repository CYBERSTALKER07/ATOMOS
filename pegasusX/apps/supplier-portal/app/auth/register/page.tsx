"use client";

import { useMemo, useState } from "react";
import { persistSession, supplierFetch } from "@/lib/auth";
import {
  COUNTRIES,
  INITIAL_STATE,
  STEP_LABELS,
  STEP_ORDER,
  type StepId,
  type WizardState,
  validateIdentity,
  validateVerification,
  validateProfile,
} from "./wizard-state";

function composeAddress(parts: Array<string>): string {
  return parts.map((part) => part.trim()).filter(Boolean).join(", ");
}

// 3-step Supplier onboarding wizard.
// HARD PRODUCT INVARIANT: never move business/location/payment setup back into this form. 
// That complex setup lives at /setup/billing post-registration.

export default function RegisterPage() {
  const [state, setState] = useState<WizardState>(INITIAL_STATE);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const stepIndex = STEP_ORDER.indexOf(state.step);
  const dialCode = useMemo(
    () => COUNTRIES.find((c) => c.code === state.identity.countryCode)?.dialCode ?? "",
    [state.identity.countryCode],
  );

  function validateCurrent(): Record<string, string> {
    switch (state.step) {
      case "identity":    return validateIdentity(state.identity);
      case "verification":return validateVerification(state.verification);
      case "profile":     return validateProfile(state.profile);
    }
  }

  function next() {
    const e = validateCurrent();
    setErrors(e);
    if (Object.keys(e).length > 0) return;
    const ni = Math.min(stepIndex + 1, STEP_ORDER.length - 1);
    setState((s) => ({ ...s, step: STEP_ORDER[ni] }));
  }

  function back() {
    const pi = Math.max(stepIndex - 1, 0);
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
        id_token: state.verification.otpCode, // Usually we pass the firebase token here. Passing OTP as fallback for scaffold.
      };
      const res = await supplierFetch("/v1/auth/supplier/register", {
        method: "POST",
        headers: {
          "Idempotency-Key": cryptoRandomId(),
        },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const body = await res.text();
        throw new Error(body || `register failed: ${res.status}`);
      }
      const data = (await res.json()) as { token?: string };
      if (data.token) {
        persistSession(data.token);
      }
      window.location.href = "/setup/business";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="auth-card">
      <header className="mb-6">
        <h1 className="md-typescale-headline-large" style={{ margin: 0 }}>
          Set up your supplier account
        </h1>
        <p className="desk-page-subtitle">
          Step {stepIndex + 1} of {STEP_ORDER.length} — {STEP_LABELS[state.step]}
        </p>
        <Stepper currentIndex={stepIndex} />
      </header>

      <section className="md-card p-6">
        {state.step === "identity"     && <IdentityStepView state={state} setState={setState} errors={errors} dialCode={dialCode} />}
        {state.step === "verification" && <VerificationStepView state={state} setState={setState} errors={errors} />}
        {state.step === "profile"      && <ProfileStepView state={state} setState={setState} errors={errors} />}
      </section>

      {submitError && (
        <p role="alert" className="md-typescale-body-medium mt-4" style={{ color: "var(--color-md-error)" }}>
          {submitError}
        </p>
      )}

      <footer className="mt-6 flex items-center justify-between gap-4">
        <button type="button" className="md-btn md-btn-text" onClick={back} disabled={stepIndex === 0 || submitting}>
          Back
        </button>
        {state.step !== "profile" ? (
          <button type="button" className="md-btn md-btn-filled" onClick={next} disabled={submitting}>
            Continue
          </button>
        ) : (
          <button type="button" className="md-btn md-btn-filled" onClick={submit} disabled={submitting}>
            {submitting ? "Creating..." : "Create supplier"}
          </button>
        )}
      </footer>
    </div>
  );
}

function Stepper({ currentIndex }: { currentIndex: number }) {
  return (
    <ol className="auth-step-indicator" aria-label="Onboarding progress">
      {STEP_ORDER.map((id, index) => {
        const done = index < currentIndex;
        const active = index === currentIndex;
        return (
          <li key={id} className="flex items-center gap-3">
            {index > 0 ? <span className="auth-step-connector" aria-hidden /> : null}
            <span
              className={`auth-step-dot ${active ? "auth-step-dot--active" : ""} ${done ? "auth-step-dot--done" : ""}`}
              aria-current={active ? "step" : undefined}
              title={STEP_LABELS[id]}
            >
              {index + 1}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

// ── Step views ───────────────────────────────────────────────────────────

type ViewProps = {
  state: WizardState;
  setState: React.Dispatch<React.SetStateAction<WizardState>>;
  errors: Record<string, string>;
};

function IdentityStepView({ state, setState, errors, dialCode }: ViewProps & { dialCode: string }) {
  return (
    <div className="grid gap-4">
      <div className="grid grid-cols-[160px,1fr] gap-3">
        <Field id="countryCode" label="Country" error={errors.countryCode}>
          <select
            id="countryCode"
            className="md-input-outlined"
            value={state.identity.countryCode}
            onChange={(e) => setState((s) => ({ ...s, identity: { ...s.identity, countryCode: e.target.value } }))}
          >
            {COUNTRIES.map((c) => (
              <option key={c.code} value={c.code}>{c.name}</option>
            ))}
          </select>
        </Field>
        <Field id="phoneLocal" label="Phone" error={errors.phoneLocal} hint={`Will be sent as ${dialCode}${state.identity.phoneLocal || "…"}`}>
          <div className="flex">
            <span
              className="inline-flex items-center px-3 border border-r-0 rounded-l text-sm"
              style={{
                borderColor: "var(--color-md-outline)",
                background: "var(--color-md-surface-container-high)",
              }}
            >
              {dialCode}
            </span>
            <input
              id="phoneLocal"
              inputMode="numeric"
              className="md-input-outlined"
              style={{ borderTopLeftRadius: 0, borderBottomLeftRadius: 0 }}
              value={state.identity.phoneLocal}
              aria-invalid={!!errors.phoneLocal}
              onChange={(e) => setState((s) => ({ ...s, identity: { ...s.identity, phoneLocal: e.target.value.replace(/\D/g, "") } }))}
            />
          </div>
        </Field>
      </div>
      <div id="recaptcha-container"></div>
    </div>
  );
}

function VerificationStepView({ state, setState, errors }: ViewProps) {
  return (
    <div className="grid gap-4">
      <Field id="otpCode" label="Verification Code" error={errors.otpCode} hint="Enter the 6-digit code sent via SMS.">
        <input
          id="otpCode"
          inputMode="numeric"
          className="md-input-outlined tracking-widest text-lg font-mono text-center"
          value={state.verification.otpCode}
          maxLength={6}
          aria-invalid={!!errors.otpCode}
          onChange={(e) => setState((s) => ({ ...s, verification: { ...s.verification, otpCode: e.target.value.replace(/\D/g, "") } }))}
        />
      </Field>
    </div>
  );
}

function ProfileStepView({ state, setState, errors }: ViewProps) {
  return (
    <div className="grid gap-4">
      <Field id="legalName" label="Legal company name" error={errors.legalName}>
        <input
          id="legalName"
          className="md-input-outlined"
          value={state.profile.legalName}
          aria-invalid={!!errors.legalName}
          onChange={(e) => setState((s) => ({ ...s, profile: { ...s.profile, legalName: e.target.value } }))}
        />
      </Field>
      <Field id="contactName" label="Primary contact name" error={errors.contactName}>
        <input
          id="contactName"
          className="md-input-outlined"
          value={state.profile.contactName}
          aria-invalid={!!errors.contactName}
          onChange={(e) => setState((s) => ({ ...s, profile: { ...s.profile, contactName: e.target.value } }))}
        />
      </Field>
      <Field id="email" label="Work email" error={errors.email}>
        <input
          id="email"
          type="email"
          autoComplete="email"
          className="md-input-outlined"
          value={state.profile.email}
          aria-invalid={!!errors.email}
          onChange={(e) => setState((s) => ({ ...s, profile: { ...s.profile, email: e.target.value } }))}
        />
      </Field>
    </div>
  );
}

// ── Helpers ──────────────────────────────────────────────────────────────

function Field({ id, label, error, hint, children }: { id: string; label: string; error?: string; hint?: string; children: React.ReactNode }) {
  return (
    <div>
      <label htmlFor={id} className="md-label">{label}</label>
      {children}
      {error
        ? <p className="md-helper" data-error="true">{error}</p>
        : hint ? <p className="md-helper">{hint}</p> : null}
    </div>
  );
}

function parseIntOrZero(v: string): number {
  const n = parseInt(v, 10);
  return Number.isFinite(n) && n >= 0 ? n : 0;
}

function cryptoRandomId(): string {
  // Browser-safe random key for the Idempotency-Key header.
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2) + Date.now().toString(36);
}
