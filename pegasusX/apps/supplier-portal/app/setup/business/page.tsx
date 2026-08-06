"use client";

import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierBusinessSetupKey } from "@pegasusx/api-client";
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
  const t = usePortalT();
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
      const body = JSON.stringify(state);
      const res = await supplierFetch("/v1/supplier/business/setup", {
        method: "POST",
        headers: {
          "Idempotency-Key": supplierBusinessSetupKey(supplierScopeId(), body),
        },
        body,
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
        title={t("supplier_portal.auth.register.header.business_title")}
        subtitle={t("supplier_portal.residual.text.legal_and_location_information_used_for_invoicing_compliance_and")}
      />

      <SetupCallout>
        This information appears on retailer invoices and tax documents. You can update it later in Profile settings.
      </SetupCallout>

      <SetupSection
        icon="kyc"
        title={t("supplier_portal.setup.business.text.tax_and_registration")}
        description={t("supplier_portal.residual.text.required_for_compliance_and_payout_verification")}
      >
        <SetupField id="taxId" label={t("supplier_portal.residual.text.tax_id_vat_tin")} error={errors.taxId} hint="Minimum 5 characters.">
          <SetupInput
            id="taxId"
            error={errors.taxId}
            value={state.taxId}
            autoComplete="off"
            placeholder={t("supplier_portal.setup.business.text.e_g_gb123456789")}
            onChange={(e) => setState((s) => ({ ...s, taxId: e.target.value }))}
          />
        </SetupField>
        <SetupField id="registrationNumber" label={t("supplier_portal.residual.text.company_registration_number")} optional error={errors.registrationNumber}>
          <SetupInput
            id="registrationNumber"
            value={state.registrationNumber}
            autoComplete="off"
            placeholder={t("supplier_portal.setup.business.text.companies_house_or_local_registry_id")}
            onChange={(e) => setState((s) => ({ ...s, registrationNumber: e.target.value }))}
          />
        </SetupField>
      </SetupSection>

      <SetupSection
        icon="pin"
        title={t("supplier_portal.setup.business.text.headquarters_location")}
        description={t("supplier_portal.residual.text.your_registered_business_address_not_a_warehouse_dispatch_point")}
      >
        <SetupField id="headquartersAddress" label={t("supplier_portal.residual.text.street_address")} error={errors.headquartersAddress}>
          <SetupInput
            id="headquartersAddress"
            error={errors.headquartersAddress}
            value={state.headquartersAddress}
            autoComplete="street-address"
            placeholder={t("supplier_portal.setup.business.text.building_street_suite")}
            onChange={(e) => setState((s) => ({ ...s, headquartersAddress: e.target.value }))}
          />
        </SetupField>
        <div className="setup-grid-2">
          <SetupField id="city" label={t("supplier_portal.analytics.demand.signals.text.city")} error={errors.city}>
            <SetupInput
              id="city"
              error={errors.city}
              value={state.city}
              autoComplete="address-level2"
              onChange={(e) => setState((s) => ({ ...s, city: e.target.value }))}
            />
          </SetupField>
          <SetupField id="postalCode" label={t("supplier_portal.residual.text.postal_code")} error={errors.postalCode}>
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
        skip={{ label: t("common.action.skip_for_now"), onClick: skip, disabled: submitting }}
        primary={{ label: t("supplier_portal.residual.text.save_and_continue"), onClick: submit, disabled: submitting, loading: submitting }}
      />
    </>
  );
}
