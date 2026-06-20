"use client";

import { useState } from "react";
import { supplierFetch } from "@/lib/auth";
import {
  SetupCallout,
  SetupField,
  SetupFooter,
  SetupInput,
  SetupPageHeader,
  SetupSection,
} from "@/components/setup/SetupPrimitives";

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
    if (state.taxId.trim().length < 5) e.taxId = "Enter a valid tax ID (VAT, TIN, or equivalent)";
    if (state.headquartersAddress.trim().length < 5) e.headquartersAddress = "Street address is required";
    if (state.city.trim().length < 2) e.city = "City is required";
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
      window.location.href = "/setup/billing";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  function skip() {
    window.location.href = "/setup/billing";
  }

  return (
    <>
      <SetupPageHeader
        icon="supplier"
        title="Business details"
        subtitle="Legal and location information used for invoicing, compliance, and delivery routing."
      />

      <SetupCallout>
        This information appears on retailer invoices and tax documents. You can update it later in Profile settings.
      </SetupCallout>

      <SetupSection
        icon="kyc"
        title="Tax & registration"
        description="Required for compliance and payout verification."
      >
        <SetupField id="taxId" label="Tax ID (VAT / TIN)" error={errors.taxId} hint="Minimum 5 characters.">
          <SetupInput
            id="taxId"
            error={errors.taxId}
            value={state.taxId}
            autoComplete="off"
            placeholder="e.g. GB123456789"
            onChange={(e) => setState((s) => ({ ...s, taxId: e.target.value }))}
          />
        </SetupField>
        <SetupField id="registrationNumber" label="Company registration number" optional error={errors.registrationNumber}>
          <SetupInput
            id="registrationNumber"
            value={state.registrationNumber}
            autoComplete="off"
            placeholder="Companies House or local registry ID"
            onChange={(e) => setState((s) => ({ ...s, registrationNumber: e.target.value }))}
          />
        </SetupField>
      </SetupSection>

      <SetupSection
        icon="pin"
        title="Headquarters location"
        description="Your registered business address — not a warehouse dispatch point."
      >
        <SetupField id="headquartersAddress" label="Street address" error={errors.headquartersAddress}>
          <SetupInput
            id="headquartersAddress"
            error={errors.headquartersAddress}
            value={state.headquartersAddress}
            autoComplete="street-address"
            placeholder="Building, street, suite"
            onChange={(e) => setState((s) => ({ ...s, headquartersAddress: e.target.value }))}
          />
        </SetupField>
        <div className="setup-grid-2">
          <SetupField id="city" label="City" error={errors.city}>
            <SetupInput
              id="city"
              error={errors.city}
              value={state.city}
              autoComplete="address-level2"
              onChange={(e) => setState((s) => ({ ...s, city: e.target.value }))}
            />
          </SetupField>
          <SetupField id="postalCode" label="Postal code" error={errors.postalCode}>
            <SetupInput
              id="postalCode"
              value={state.postalCode}
              autoComplete="postal-code"
              onChange={(e) => setState((s) => ({ ...s, postalCode: e.target.value }))}
            />
          </SetupField>
        </div>
      </SetupSection>

      {submitError ? <SetupCallout variant="error">{submitError}</SetupCallout> : null}

      <SetupFooter
        skip={{ label: "Skip for now", onClick: skip, disabled: submitting }}
        primary={{ label: "Save & continue", onClick: submit, disabled: submitting, loading: submitting }}
      />
    </>
  );
}

function cryptoRandomId() {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
