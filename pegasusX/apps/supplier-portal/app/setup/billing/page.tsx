"use client";

import { useState } from "react";
import { supplierFetch } from "@/lib/auth";

// /setup/billing — bank + payment-gateway configuration.
// Decoupled from the 4-step registration wizard to reduce friction.
// Hard product invariant: this page must remain a post-registration step
// that suppliers reach AFTER /auth/register. Do not merge into the wizard.

const GATEWAYS = [
  { id: "GLOBAL_PAY", label: "Global Pay" },
  { id: "ADYEN",      label: "Adyen" },
  { id: "AIRWALLEX",  label: "Airwallex" },
  { id: "CASH",       label: "Cash on delivery only" },
] as const;

type GatewayId = typeof GATEWAYS[number]["id"];

interface BillingState {
  bankName: string;
  accountHolder: string;
  accountNumber: string;
  swiftBic: string;
  iban: string;
  selectedGateways: GatewayId[];
}

const INITIAL: BillingState = {
  bankName: "",
  accountHolder: "",
  accountNumber: "",
  swiftBic: "",
  iban: "",
  selectedGateways: [],
};

export default function BillingSetupPage() {
  const [state, setState] = useState<BillingState>(INITIAL);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (state.bankName.trim().length < 2) e.bankName = "Bank name required";
    if (state.accountHolder.trim().length < 2) e.accountHolder = "Account holder required";
    if (state.accountNumber.trim().length < 4) e.accountNumber = "Account number required";
    if (state.swiftBic.trim().length < 4) e.swiftBic = "SWIFT / BIC required";
    if (state.selectedGateways.length === 0) e.selectedGateways = "Choose at least one gateway";
    return e;
  }

  function toggleGateway(id: GatewayId) {
    setState((s) => {
      const set = new Set(s.selectedGateways);
      if (set.has(id)) set.delete(id); else set.add(id);
      return { ...s, selectedGateways: Array.from(set) as GatewayId[] };
    });
  }

  async function submit() {
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length > 0) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      const res = await supplierFetch("/v1/supplier/billing/setup", {
        method: "POST",
        headers: {
          "Idempotency-Key": cryptoRandomId(),
        },
        body: JSON.stringify(state),
      });
      if (!res.ok) {
        const body = await res.text();
        throw new Error(body || `billing setup failed: ${res.status}`);
      }
      window.location.href = "/";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  async function skip() {
    // Per onboarding gate spec, skip is allowed but flags is_configured=false
    // so the gate keeps redirecting until the supplier completes setup.
    window.location.href = "/?billing=skipped";
  }

  return (
    <>
      <header className="setup-header">
        <div className="setup-header-icon" aria-hidden>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 4H4v2h16V4zm1 10v-2l-1-5H4l-1 5v2h1v6h10v-6h4v6h2v-6h1zm-9 4H6v-4h6v4z" />
          </svg>
        </div>
        <div>
          <h1 className="md-typescale-title-large" style={{ margin: 0 }}>
            Billing &amp; payment gateways
          </h1>
          <p className="desk-page-subtitle">
            Configure payouts and the gateways retailers use at checkout.
          </p>
        </div>
      </header>

      <section className="auth-card grid gap-4">
        <h2 className="md-typescale-title-large">Bank account</h2>
        <Field id="bankName" label="Bank name" error={errors.bankName}>
          <input id="bankName" className="md-input-outlined" value={state.bankName}
            onChange={(e) => setState((s) => ({ ...s, bankName: e.target.value }))} />
        </Field>
        <Field id="accountHolder" label="Account holder name" error={errors.accountHolder}>
          <input id="accountHolder" className="md-input-outlined" value={state.accountHolder}
            onChange={(e) => setState((s) => ({ ...s, accountHolder: e.target.value }))} />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field id="accountNumber" label="Account number" error={errors.accountNumber}>
            <input id="accountNumber" className="md-input-outlined" value={state.accountNumber}
              onChange={(e) => setState((s) => ({ ...s, accountNumber: e.target.value }))} />
          </Field>
          <Field id="swiftBic" label="SWIFT / BIC" error={errors.swiftBic}>
            <input id="swiftBic" className="md-input-outlined" value={state.swiftBic}
              onChange={(e) => setState((s) => ({ ...s, swiftBic: e.target.value }))} />
          </Field>
        </div>
        <Field id="iban" label="IBAN" error={errors.iban} hint="Optional for non-IBAN regions.">
          <input id="iban" className="md-input-outlined" value={state.iban}
            onChange={(e) => setState((s) => ({ ...s, iban: e.target.value }))} />
        </Field>

        <h2 className="md-typescale-title-large mt-4">Payment gateways</h2>
        <p className="md-typescale-body-medium" style={{ color: "var(--color-md-outline)" }}>
          Pick every gateway you want retailers to use at checkout. You can change this later.
        </p>
        <div className="flex flex-wrap gap-2" role="group" aria-label="Payment gateways">
          {GATEWAYS.map((g) => {
            const pressed = state.selectedGateways.includes(g.id);
            return (
              <button key={g.id} type="button" className="md-chip" aria-pressed={pressed} onClick={() => toggleGateway(g.id)}>
                {g.label}
              </button>
            );
          })}
        </div>
        {errors.selectedGateways && <p className="md-helper" data-error="true">{errors.selectedGateways}</p>}
      </section>

      {submitError && (
        <p role="alert" className="md-typescale-body-medium mt-4" style={{ color: "var(--color-md-error)" }}>
          {submitError}
        </p>
      )}

      <footer className="mt-6 flex items-center justify-between gap-4">
        <button type="button" className="md-btn md-btn-text" onClick={skip} disabled={submitting}>
          Skip for now
        </button>
        <button type="button" className="md-btn md-btn-filled" onClick={submit} disabled={submitting}>
          {submitting ? "Saving…" : "Save billing"}
        </button>
      </footer>
    </>
  );
}

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

function cryptoRandomId() {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
