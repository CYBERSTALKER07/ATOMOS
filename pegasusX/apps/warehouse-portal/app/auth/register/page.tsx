"use client";

import { useMemo, useState, type ChangeEvent } from "react";
import { useRouter } from "next/navigation";
import { warehouseApiBaseUrl, persistSession } from "@/lib/auth";
import {
  COUNTRIES,
  INITIAL_STATE,
  STEP_LABELS,
  STEP_ORDER,
  type WizardState,
  validateIdentity,
  validateVerification,
  validateProfile,
} from "./wizard-state";
import { PortalField, PortalInput, PortalSelect, FormAlert } from "@/components/portal";



// 3-step Warehouse onboarding wizard.
// HARD PRODUCT INVARIANT: never move business/location/payment setup back into this form. 
// That complex setup lives at /setup/location post-registration.

export default function RegisterPage() {
  const [state, setState] = useState<WizardState>(INITIAL_STATE);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const router = useRouter();

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
        id_token: state.verification.otpCode,
      };
      
      const res = await fetch(`${warehouseApiBaseUrl}/v1/auth/warehouse/register`, {
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
      router.replace(data.is_configured ? "/" : "/setup/location");
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="auth-card">
      <header className="mb-6">
        <h1 className="md-typescale-headline-medium" style={{ margin: 0 }}>
          Warehouse Registration
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

      {submitError ? <FormAlert variant="error">{submitError}</FormAlert> : null}

      <footer className="mt-6 flex items-center justify-between gap-4">
        <button type="button" className="portal-btn portal-btn--ghost" onClick={back} disabled={stepIndex === 0 || submitting}>
          Back
        </button>
        {state.step !== "profile" ? (
          <button type="button" className="portal-btn portal-btn--primary" onClick={next} disabled={submitting}>
            Continue
          </button>
        ) : (
          <button type="button" className="portal-btn portal-btn--primary" onClick={submit} disabled={submitting}>
            {submitting ? "Creating…" : "Create warehouse"}
          </button>
        )}
      </footer>
    </div>
  );
}

function Stepper({ currentIndex }: { currentIndex: number }) {
  return (
    <ol className="setup-step-list mt-4" aria-label="Onboarding progress">
      {STEP_ORDER.map((id, index) => {
        const done = index < currentIndex;
        const active = index === currentIndex;
        const stateClass = done ? "setup-step-item--done" : active ? "setup-step-item--active" : "";
        return (
          <li key={id} className={`setup-step-item ${stateClass}`} aria-current={active ? "step" : undefined}>
            <span className="setup-step-badge" aria-hidden>
              {done ? (
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                  <path d="M5 12l5 5L20 7" />
                </svg>
              ) : (
                index + 1
              )}
            </span>
            <div className="setup-step-copy">
              <span className="setup-step-label">{STEP_LABELS[id]}</span>
            </div>
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
        <PortalField id="countryCode" label="Country" error={errors.countryCode}>
          <PortalSelect
            id="countryCode"
            value={state.identity.countryCode}
            onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, identity: { ...s.identity, countryCode: e.target.value } }))}
            error={errors.countryCode}
          >
            {COUNTRIES.map((c) => (
              <option key={c.code} value={c.code}>{c.name}</option>
            ))}
          </PortalSelect>
        </PortalField>
        <PortalField id="phoneLocal" label="Phone" error={errors.phoneLocal} hint={`Will be sent as ${dialCode}${state.identity.phoneLocal || "…"}`}>
          <div className="flex">
            <span className="inline-flex items-center px-3 border border-r-0 rounded-l text-sm portal-input" style={{ width: "auto", minWidth: 56, borderTopRightRadius: 0, borderBottomRightRadius: 0 }}>
              {dialCode}
            </span>
            <PortalInput
              id="phoneLocal"
              inputMode="numeric"
              className="rounded-l-none"
              style={{ borderTopLeftRadius: 0, borderBottomLeftRadius: 0 }}
              value={state.identity.phoneLocal}
              onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, identity: { ...s.identity, phoneLocal: e.target.value.replace(/\D/g, "") } }))}
              error={errors.phoneLocal}
            />
          </div>
        </PortalField>
      </div>
      <div id="recaptcha-container" />
    </div>
  );
}

function VerificationStepView({ state, setState, errors }: ViewProps) {
  return (
    <PortalField id="otpCode" label="Verification code" error={errors.otpCode} hint="Enter the 6-digit code sent via SMS.">
      <PortalInput
        id="otpCode"
        inputMode="numeric"
        className="tracking-widest text-lg font-mono text-center"
        value={state.verification.otpCode}
        maxLength={6}
        onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, verification: { ...s.verification, otpCode: e.target.value.replace(/\D/g, "") } }))}
        error={errors.otpCode}
      />
    </PortalField>
  );
}

function ProfileStepView({ state, setState, errors }: ViewProps) {
  return (
    <div className="grid gap-4">
      <PortalField id="legalName" label="Legal company name" error={errors.legalName}>
        <PortalInput id="legalName" value={state.profile.legalName} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, profile: { ...s.profile, legalName: e.target.value } }))} error={errors.legalName} />
      </PortalField>
      <PortalField id="contactName" label="Primary contact name" error={errors.contactName}>
        <PortalInput id="contactName" value={state.profile.contactName} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, profile: { ...s.profile, contactName: e.target.value } }))} error={errors.contactName} />
      </PortalField>
      <PortalField id="email" label="Work email" error={errors.email}>
        <PortalInput id="email" type="email" autoComplete="email" value={state.profile.email} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, profile: { ...s.profile, email: e.target.value } }))} error={errors.email} />
      </PortalField>
    </div>
  );
}
