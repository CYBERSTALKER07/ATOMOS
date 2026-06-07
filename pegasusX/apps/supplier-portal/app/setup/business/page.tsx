"use client";

import { useState } from "react";
import { supplierFetch } from "@/lib/auth";

interface BusinessState {
  taxId: string;
  registrationNumber: string;
  headquartersAddress: string;
  city: string;
  postalCode: string;
}

const INITIAL: BusinessState = {
  taxId: "",
  registrationNumber: "",
  headquartersAddress: "",
  city: "",
  postalCode: "",
};

export default function BusinessSetupPage() {
  const [state, setState] = useState<BusinessState>(INITIAL);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (state.taxId.trim().length < 5) e.taxId = "Tax ID required";
    if (state.headquartersAddress.trim().length < 5) e.headquartersAddress = "Address required";
    if (state.city.trim().length < 2) e.city = "City required";
    return e;
  }

  async function submit() {
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length > 0) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      const res = await supplierFetch("/v1/supplier/business/setup", {
        method: "POST",
        headers: {
          "Idempotency-Key": cryptoRandomId(),
        },
        body: JSON.stringify(state),
      });
      if (!res.ok) {
        const body = await res.text();
        throw new Error(body || `business setup failed: ${res.status}`);
      }
      // Usually proceeds to billing setup
      window.location.href = "/setup/billing";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  async function skip() {
    window.location.href = "/setup/billing";
  }

  return (
    <>
      <header className="setup-header">
        <div className="setup-header-icon" aria-hidden>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
          </svg>
        </div>
        <div>
          <h1 className="md-typescale-title-large" style={{ margin: 0 }}>
            Business Details
          </h1>
          <p className="desk-page-subtitle">
            Provide your tax and location information.
          </p>
        </div>
      </header>

      <section className="auth-card grid gap-4">
        <h2 className="md-typescale-title-large">Tax & Registration</h2>
        <Field id="taxId" label="Tax ID (VAT / TIN)" error={errors.taxId}>
          <input id="taxId" className="md-input-outlined" value={state.taxId}
            onChange={(e) => setState((s) => ({ ...s, taxId: e.target.value }))} />
        </Field>
        <Field id="registrationNumber" label="Company Registration Number (optional)" error={errors.registrationNumber}>
          <input id="registrationNumber" className="md-input-outlined" value={state.registrationNumber}
            onChange={(e) => setState((s) => ({ ...s, registrationNumber: e.target.value }))} />
        </Field>
        
        <h2 className="md-typescale-title-large mt-4">Location</h2>
        <Field id="headquartersAddress" label="Headquarters Address" error={errors.headquartersAddress}>
          <input id="headquartersAddress" className="md-input-outlined" value={state.headquartersAddress}
            onChange={(e) => setState((s) => ({ ...s, headquartersAddress: e.target.value }))} />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field id="city" label="City" error={errors.city}>
            <input id="city" className="md-input-outlined" value={state.city}
              onChange={(e) => setState((s) => ({ ...s, city: e.target.value }))} />
          </Field>
          <Field id="postalCode" label="Postal Code" error={errors.postalCode}>
            <input id="postalCode" className="md-input-outlined" value={state.postalCode}
              onChange={(e) => setState((s) => ({ ...s, postalCode: e.target.value }))} />
          </Field>
        </div>
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
          {submitting ? "Saving…" : "Save & Continue"}
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
