"use client";

import { useState } from "react";
import { factoryApiBaseUrl } from "@/lib/auth";

interface FactorySetupState {
  factoryName: string;
  facilityType: string;
  address: string;
  city: string;
  postalCode: string;
  totalCapacitySqM: string;
}

const INITIAL: FactorySetupState = {
  factoryName: "",
  facilityType: "MANUFACTURING",
  address: "",
  city: "",
  postalCode: "",
  totalCapacitySqM: "",
};

export default function FactorySetupPage() {
  const [state, setState] = useState<FactorySetupState>(INITIAL);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (state.factoryName.trim().length < 3) e.factoryName = "Name required";
    if (state.address.trim().length < 5) e.address = "Address required";
    if (state.city.trim().length < 2) e.city = "City required";
    if (!/^\d+$/.test(state.totalCapacitySqM)) e.totalCapacitySqM = "Capacity must be a number";
    return e;
  }

  async function submit() {
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length > 0) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      const res = await fetch(`${factoryApiBaseUrl}/v1/factory/setup`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": cryptoRandomId(),
          Authorization: `Bearer ${getCookie("pegasus_factory_jwt")}`,
        },
        body: JSON.stringify({
          ...state,
          totalCapacitySqM: parseInt(state.totalCapacitySqM, 10),
        }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.message || `Setup failed: ${res.status}`);
      }
      
      window.location.href = "/";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  function getCookie(name: string) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop()?.split(';').shift();
  }

  return (
    <>
      <header className="setup-header">
        <div className="setup-header-icon" aria-hidden>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
            <path d="M22 10v12H2V10l10-8 10 8zm-2 1H4v10h16v-10zm-5 1v6h-6v-6h6z" />
          </svg>
        </div>
        <div>
          <h1 className="md-typescale-title-large" style={{ margin: 0 }}>
            Factory Details
          </h1>
          <p className="desk-page-subtitle">
            Configure your factory facility type and location.
          </p>
        </div>
      </header>

      <section className="auth-card grid gap-4">
        <h2 className="md-typescale-title-large">General</h2>
        <Field id="factoryName" label="Factory Name" error={errors.factoryName}>
          <input id="factoryName" className="md-input-outlined" value={state.factoryName}
            onChange={(e) => setState((s) => ({ ...s, factoryName: e.target.value }))} />
        </Field>
        
        <Field id="facilityType" label="Facility Type" error={errors.facilityType}>
          <select id="facilityType" className="md-input-outlined" value={state.facilityType}
            onChange={(e) => setState((s) => ({ ...s, facilityType: e.target.value }))}
          >
            <option value="MANUFACTURING">Manufacturing</option>
            <option value="ASSEMBLY">Assembly</option>
            <option value="PACKAGING">Packaging</option>
            <option value="PROCESSING">Processing</option>
          </select>
        </Field>

        <Field id="totalCapacitySqM" label="Total Capacity (Square Meters)" error={errors.totalCapacitySqM}>
          <input id="totalCapacitySqM" type="number" className="md-input-outlined" value={state.totalCapacitySqM}
            onChange={(e) => setState((s) => ({ ...s, totalCapacitySqM: e.target.value }))} />
        </Field>
        
        <h2 className="md-typescale-title-large mt-4">Location</h2>
        <Field id="address" label="Street Address" error={errors.address}>
          <input id="address" className="md-input-outlined" value={state.address}
            onChange={(e) => setState((s) => ({ ...s, address: e.target.value }))} />
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
        <div /> {/* spacing */}
        <button type="button" className="md-btn md-btn-filled" onClick={submit} disabled={submitting}>
          {submitting ? "Saving…" : "Complete Setup"}
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
