"use client";

import { useState } from "react";
import { factoryApiBaseUrl, persistSession } from "@/lib/auth";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";

interface FactorySetupState {
  factoryName: string;
  facilityType: string;
  totalCapacitySqM: string;
}

const INITIAL: FactorySetupState = {
  factoryName: "",
  facilityType: "MANUFACTURING",
  totalCapacitySqM: "",
};

const DEFAULT_LOCATION: LocationValue = { address: "", lat: "0", lng: "0" };

export default function FactorySetupPage() {
  const [state, setState] = useState<FactorySetupState>(INITIAL);
  const [location, setLocation] = useState<LocationValue>(DEFAULT_LOCATION);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (state.factoryName.trim().length < 3) e.factoryName = "Name required";
    if (location.address.trim().length < 5) e.address = "Address required";
    const lat = Number(location.lat);
    const lng = Number(location.lng);
    if (!Number.isFinite(lat) || !Number.isFinite(lng) || (lat === 0 && lng === 0)) {
      e.address = "Select a valid address from suggestions";
    }
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
          factoryName: state.factoryName.trim(),
          facilityType: state.facilityType,
          totalCapacitySqM: parseInt(state.totalCapacitySqM, 10),
          address: location.address.trim(),
          place_id: location.place_id,
          lat: Number(location.lat),
          lng: Number(location.lng),
        }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.message || body?.error || `Setup failed: ${res.status}`);
      }

      const data = await res.json();
      if (data.token) {
        persistSession(data.token, data.refresh_token);
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
    if (parts.length === 2) return parts.pop()?.split(";").shift();
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
          <input
            id="factoryName"
            className="md-input-outlined"
            value={state.factoryName}
            onChange={(e) => setState((s) => ({ ...s, factoryName: e.target.value }))}
          />
        </Field>

        <Field id="facilityType" label="Facility Type" error={errors.facilityType}>
          <select
            id="facilityType"
            className="md-input-outlined"
            value={state.facilityType}
            onChange={(e) => setState((s) => ({ ...s, facilityType: e.target.value }))}
          >
            <option value="MANUFACTURING">Manufacturing</option>
            <option value="ASSEMBLY">Assembly</option>
            <option value="PACKAGING">Packaging</option>
            <option value="PROCESSING">Processing</option>
          </select>
        </Field>

        <Field id="totalCapacitySqM" label="Total Capacity (Square Meters)" error={errors.totalCapacitySqM}>
          <input
            id="totalCapacitySqM"
            type="number"
            className="md-input-outlined"
            value={state.totalCapacitySqM}
            onChange={(e) => setState((s) => ({ ...s, totalCapacitySqM: e.target.value }))}
          />
        </Field>

        <h2 className="md-typescale-title-large mt-4">Location</h2>
        <Field id="address" label="Factory address" error={errors.address}>
          <LocationPicker value={location} onChange={setLocation} label="Street address" />
        </Field>
      </section>

      {submitError && (
        <p role="alert" className="md-typescale-body-medium mt-4" style={{ color: "var(--color-md-error)" }}>
          {submitError}
        </p>
      )}

      <footer className="mt-6 flex items-center justify-between gap-4">
        <div />
        <button type="button" className="md-btn md-btn-filled" onClick={submit} disabled={submitting}>
          {submitting ? "Saving…" : "Complete Setup"}
        </button>
      </footer>
    </>
  );
}

function Field({
  id,
  label,
  error,
  hint,
  children,
}: {
  id: string;
  label: string;
  error?: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <label htmlFor={id} className="md-label">
        {label}
      </label>
      {children}
      {error ? (
        <p className="md-helper" data-error="true">
          {error}
        </p>
      ) : hint ? (
        <p className="md-helper">{hint}</p>
      ) : null}
    </div>
  );
}

function cryptoRandomId() {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
