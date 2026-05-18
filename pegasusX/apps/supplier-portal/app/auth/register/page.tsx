"use client";

import { useMemo, useState } from "react";
import {
  CATEGORY_OPTIONS,
  COUNTRIES,
  INITIAL_STATE,
  STEP_LABELS,
  STEP_ORDER,
  type StepId,
  type WizardState,
  validateAccount,
  validateBusiness,
  validateCategories,
  validateLocation,
} from "./wizard-state";

// 4-step Supplier onboarding wizard.
// HARD PRODUCT INVARIANT: never collapse below 4 steps; never move
// banking / payment-gateway setup back into this form. Banking lives at
// /setup/billing post-registration.

export default function RegisterPage() {
  const [state, setState] = useState<WizardState>(INITIAL_STATE);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const stepIndex = STEP_ORDER.indexOf(state.step);
  const dialCode = useMemo(
    () => COUNTRIES.find((c) => c.code === state.account.countryCode)?.dialCode ?? "",
    [state.account.countryCode],
  );

  function validateCurrent(): Record<string, string> {
    switch (state.step) {
      case "account":    return validateAccount(state.account);
      case "location":   return validateLocation(state.location);
      case "business":   return validateBusiness(state.business);
      case "categories": return validateCategories(state.categories);
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
      const payload = {
        account:    state.account,
        location:   state.location,
        business:   state.business,
        categories: state.categories,
        phone:      dialCode + state.account.phoneLocal,
      };
      const res = await fetch("/api/auth/supplier/register", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Idempotency-Key": cryptoRandomId(),
        },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const body = await res.text();
        throw new Error(body || `register failed: ${res.status}`);
      }
      // Per onboarding gate: redirect to /setup/billing for bank + gateway.
      window.location.href = "/setup/billing";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="min-h-screen p-6 md:p-12 max-w-3xl mx-auto">
      <header className="mb-8">
        <h1 className="md-typescale-headline-large">Set up your supplier account</h1>
        <p className="md-typescale-body-medium mt-2" style={{ color: "var(--color-md-outline)" }}>
          Step {stepIndex + 1} of {STEP_ORDER.length} — {STEP_LABELS[state.step]}
        </p>
        <Stepper currentIndex={stepIndex} />
      </header>

      <section className="md-card p-6 md-shape-md">
        {state.step === "account"    && <AccountStepView state={state} setState={setState} errors={errors} dialCode={dialCode} />}
        {state.step === "location"   && <LocationStepView state={state} setState={setState} errors={errors} />}
        {state.step === "business"   && <BusinessStepView state={state} setState={setState} errors={errors} />}
        {state.step === "categories" && <CategoriesStepView state={state} setState={setState} errors={errors} />}
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
        {state.step !== "categories" ? (
          <button type="button" className="md-btn md-btn-filled" onClick={next} disabled={submitting}>
            Continue
          </button>
        ) : (
          <button type="button" className="md-btn md-btn-filled" onClick={submit} disabled={submitting}>
            {submitting ? "Submitting…" : "Create supplier"}
          </button>
        )}
      </footer>
    </main>
  );
}

function Stepper({ currentIndex }: { currentIndex: number }) {
  return (
    <ol className="mt-4 flex gap-2" aria-label="Onboarding progress">
      {STEP_ORDER.map((id, i) => {
        const done = i < currentIndex;
        const active = i === currentIndex;
        return (
          <li key={id} className="flex-1">
            <div
              className="h-1 rounded-full"
              style={{
                background:
                  done || active
                    ? "var(--color-md-primary)"
                    : "var(--color-md-outline-variant)",
              }}
              aria-current={active ? "step" : undefined}
            />
            <span className="md-typescale-label-small mt-1 block" style={{ color: "var(--color-md-outline)" }}>
              {STEP_LABELS[id]}
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

function AccountStepView({ state, setState, errors, dialCode }: ViewProps & { dialCode: string }) {
  return (
    <div className="grid gap-4">
      <Field id="legalName" label="Legal company name" error={errors.legalName}>
        <input
          id="legalName"
          className="md-input-outlined"
          value={state.account.legalName}
          aria-invalid={!!errors.legalName}
          onChange={(e) => setState((s) => ({ ...s, account: { ...s.account, legalName: e.target.value } }))}
        />
      </Field>
      <Field id="contactName" label="Primary contact name" error={errors.contactName}>
        <input
          id="contactName"
          className="md-input-outlined"
          value={state.account.contactName}
          aria-invalid={!!errors.contactName}
          onChange={(e) => setState((s) => ({ ...s, account: { ...s.account, contactName: e.target.value } }))}
        />
      </Field>
      <Field id="email" label="Work email" error={errors.email}>
        <input
          id="email"
          type="email"
          autoComplete="email"
          className="md-input-outlined"
          value={state.account.email}
          aria-invalid={!!errors.email}
          onChange={(e) => setState((s) => ({ ...s, account: { ...s.account, email: e.target.value } }))}
        />
      </Field>
      <div className="grid grid-cols-[160px,1fr] gap-3">
        <Field id="countryCode" label="Country" error={errors.countryCode}>
          <select
            id="countryCode"
            className="md-input-outlined"
            value={state.account.countryCode}
            onChange={(e) => setState((s) => ({ ...s, account: { ...s.account, countryCode: e.target.value } }))}
          >
            {COUNTRIES.map((c) => (
              <option key={c.code} value={c.code}>{c.name}</option>
            ))}
          </select>
        </Field>
        <Field id="phoneLocal" label="Phone" error={errors.phoneLocal} hint={`Will be sent as ${dialCode}${state.account.phoneLocal || "…"}`}>
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
              value={state.account.phoneLocal}
              aria-invalid={!!errors.phoneLocal}
              onChange={(e) => setState((s) => ({ ...s, account: { ...s.account, phoneLocal: e.target.value.replace(/\D/g, "") } }))}
            />
          </div>
        </Field>
      </div>
      <Field id="password" label="Password" error={errors.password} hint="At least 10 characters.">
        <input
          id="password"
          type="password"
          autoComplete="new-password"
          className="md-input-outlined"
          value={state.account.password}
          aria-invalid={!!errors.password}
          onChange={(e) => setState((s) => ({ ...s, account: { ...s.account, password: e.target.value } }))}
        />
      </Field>
    </div>
  );
}

function LocationStepView({ state, setState, errors }: ViewProps) {
  return (
    <div className="grid gap-4">
      <h2 className="md-typescale-title-large">Primary warehouse</h2>
      <Field id="warehouseName" label="Warehouse name" error={errors.warehouseName}>
        <input id="warehouseName" className="md-input-outlined" value={state.location.warehouseName}
          onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, warehouseName: e.target.value } }))} />
      </Field>
      <Field id="warehouseLine1" label="Street address" error={errors.warehouseLine1}>
        <input id="warehouseLine1" className="md-input-outlined" value={state.location.warehouseLine1}
          onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, warehouseLine1: e.target.value } }))} />
      </Field>
      <div className="grid grid-cols-3 gap-3">
        <Field id="warehouseCity" label="City" error={errors.warehouseCity}>
          <input id="warehouseCity" className="md-input-outlined" value={state.location.warehouseCity}
            onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, warehouseCity: e.target.value } }))} />
        </Field>
        <Field id="warehouseRegion" label="Region" error={errors.warehouseRegion}>
          <input id="warehouseRegion" className="md-input-outlined" value={state.location.warehouseRegion}
            onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, warehouseRegion: e.target.value } }))} />
        </Field>
        <Field id="warehousePostalCode" label="Postal code" error={errors.warehousePostalCode}>
          <input id="warehousePostalCode" className="md-input-outlined" value={state.location.warehousePostalCode}
            onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, warehousePostalCode: e.target.value } }))} />
        </Field>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <Field id="warehouseLat" label="Latitude" error={errors.warehouseLat}>
          <input id="warehouseLat" inputMode="decimal" className="md-input-outlined" value={state.location.warehouseLat}
            onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, warehouseLat: e.target.value } }))} />
        </Field>
        <Field id="warehouseLng" label="Longitude" error={errors.warehouseLng}>
          <input id="warehouseLng" inputMode="decimal" className="md-input-outlined" value={state.location.warehouseLng}
            onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, warehouseLng: e.target.value } }))} />
        </Field>
      </div>

      <h2 className="md-typescale-title-large mt-4">Billing address</h2>
      <label className="inline-flex items-center gap-2">
        <input
          type="checkbox"
          checked={state.location.billingSameAsWarehouse}
          onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, billingSameAsWarehouse: e.target.checked } }))}
        />
        <span className="md-typescale-body-medium">Billing address is the same as the warehouse</span>
      </label>
      {!state.location.billingSameAsWarehouse && (
        <>
          <Field id="billingLine1" label="Billing street address" error={errors.billingLine1}>
            <input id="billingLine1" className="md-input-outlined" value={state.location.billingLine1}
              onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, billingLine1: e.target.value } }))} />
          </Field>
          <div className="grid grid-cols-3 gap-3">
            <Field id="billingCity" label="City" error={errors.billingCity}>
              <input id="billingCity" className="md-input-outlined" value={state.location.billingCity}
                onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, billingCity: e.target.value } }))} />
            </Field>
            <Field id="billingRegion" label="Region" error={errors.billingRegion}>
              <input id="billingRegion" className="md-input-outlined" value={state.location.billingRegion}
                onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, billingRegion: e.target.value } }))} />
            </Field>
            <Field id="billingPostalCode" label="Postal code" error={errors.billingPostalCode}>
              <input id="billingPostalCode" className="md-input-outlined" value={state.location.billingPostalCode}
                onChange={(e) => setState((s) => ({ ...s, location: { ...s.location, billingPostalCode: e.target.value } }))} />
            </Field>
          </div>
        </>
      )}
    </div>
  );
}

function BusinessStepView({ state, setState, errors }: ViewProps) {
  return (
    <div className="grid gap-4">
      <Field id="taxId" label="Tax ID" error={errors.taxId}>
        <input id="taxId" className="md-input-outlined" value={state.business.taxId}
          onChange={(e) => setState((s) => ({ ...s, business: { ...s.business, taxId: e.target.value } }))} />
      </Field>
      <Field id="companyRegNumber" label="Company registration number" error={errors.companyRegNumber}>
        <input id="companyRegNumber" className="md-input-outlined" value={state.business.companyRegNumber}
          onChange={(e) => setState((s) => ({ ...s, business: { ...s.business, companyRegNumber: e.target.value } }))} />
      </Field>
      <div className="grid grid-cols-3 gap-3">
        <Field id="fleetVehicleCount" label="Fleet vehicles" error={errors.fleetVehicleCount}>
          <input id="fleetVehicleCount" type="number" min={0} className="md-input-outlined" value={state.business.fleetVehicleCount}
            onChange={(e) => setState((s) => ({ ...s, business: { ...s.business, fleetVehicleCount: parseIntOrZero(e.target.value) } }))} />
        </Field>
        <Field id="fleetMaxVU" label="Total fleet VU" error={errors.fleetMaxVU} hint="Total Volumetric Units across the fleet.">
          <input id="fleetMaxVU" type="number" min={0} className="md-input-outlined" value={state.business.fleetMaxVU}
            onChange={(e) => setState((s) => ({ ...s, business: { ...s.business, fleetMaxVU: parseIntOrZero(e.target.value) } }))} />
        </Field>
        <Field id="factoryCount" label="Factories" error={errors.factoryCount}>
          <input id="factoryCount" type="number" min={0} className="md-input-outlined" value={state.business.factoryCount}
            onChange={(e) => setState((s) => ({ ...s, business: { ...s.business, factoryCount: parseIntOrZero(e.target.value) } }))} />
        </Field>
      </div>
    </div>
  );
}

function CategoriesStepView({ state, setState, errors }: ViewProps) {
  const selected = new Set(state.categories.selectedCategoryIds);
  function toggle(id: string) {
    setState((s) => {
      const set = new Set(s.categories.selectedCategoryIds);
      if (set.has(id)) set.delete(id); else set.add(id);
      return { ...s, categories: { selectedCategoryIds: Array.from(set) } };
    });
  }
  return (
    <div className="grid gap-3">
      <p className="md-typescale-body-medium" style={{ color: "var(--color-md-outline)" }}>
        Choose every category you serve. You can adjust this later from settings.
      </p>
      <div className="flex flex-wrap gap-2" role="group" aria-label="Categories">
        {CATEGORY_OPTIONS.map((c) => {
          const pressed = selected.has(c.id);
          return (
            <button
              key={c.id}
              type="button"
              className="md-chip"
              aria-pressed={pressed}
              onClick={() => toggle(c.id)}
            >
              {c.label}
            </button>
          );
        })}
      </div>
      {errors.selectedCategoryIds && (
        <p className="md-helper" data-error="true">{errors.selectedCategoryIds}</p>
      )}
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
  // Browser-safe random key for the X-Idempotency-Key header.
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2) + Date.now().toString(36);
}
