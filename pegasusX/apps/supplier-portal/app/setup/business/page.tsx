"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { persistSession, supplierFetch } from "@/lib/auth";
import { supplierScopeId } from "@/lib/supplier-scope";
import { createSupplierApi } from "@/lib/api";
import { supplierBusinessSetupKey } from "@pegasusx/api-core";
import type { INNLookupResponse, JoinRequestConflictPayload } from "@pegasusx/types";
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
  const api = useMemo(() => createSupplierApi(), []);
  const [state, setState] = useState<BusinessState>(INITIAL);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [verifyingInn, setVerifyingInn] = useState(false);
  const [innLookup, setInnLookup] = useState<INNLookupResponse | null>(null);
  const [innError, setInnError] = useState<string | null>(null);
  const [joinPending, setJoinPending] = useState<JoinRequestConflictPayload | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void api
      .getSupplierProfile()
      .then((profile) => {
        if (cancelled) return;
        const knownInn = (profile.normalized_tax_id || profile.tax_id || "").trim();
        if (!knownInn) return;
        setState((s) => (s.taxId.trim() ? s : { ...s, taxId: knownInn }));
        setInnLookup((prev) =>
          prev ?? {
            status: "FOUND",
            inn: knownInn,
            normalized_inn: knownInn,
            legal_name: profile.legal_name,
            cached: false,
            existing_join_eligible: false,
          },
        );
      })
      .catch(() => {
        // Profile fetch is best-effort prefill on /setup/business.
      });
    return () => {
      cancelled = true;
    };
  }, [api]);

  async function verifyInn() {
    const cleaned = state.taxId.replace(/\D/g, "");
    if (cleaned.length !== 9) {
      setErrors((prev) => ({ ...prev, taxId: t("supplier_portal.onboarding_inn.invalid_inn") }));
      return;
    }
    setVerifyingInn(true);
    setInnError(null);
    setErrors((prev) => ({ ...prev, taxId: "" }));
    try {
      const res = await api.lookupPlatformINN({
        inn: cleaned,
        tenant_type: "SUPPLIER",
      });
      setInnLookup(res);
      if (res.status === "UNAVAILABLE") {
        setInnError(t("supplier_portal.onboarding_inn.registry_unavailable"));
        return;
      }
      if (res.status !== "FOUND") {
        setInnError(t("supplier_portal.onboarding_inn.invalid_inn"));
        return;
      }
      if (res.address && !state.headquartersAddress.trim()) {
        setState((s) => ({ ...s, headquartersAddress: res.address ?? "" }));
      }
    } catch (err) {
      setInnError(
        err instanceof Error ? err.message : t("supplier_portal.onboarding_inn.registry_unavailable"),
      );
    } finally {
      setVerifyingInn(false);
    }
  }

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    const cleanedInn = state.taxId.replace(/\D/g, "");
    if (cleanedInn.length !== 9) e.taxId = t("supplier_portal.onboarding_inn.invalid_inn");
    if (innLookup?.status === "UNAVAILABLE") {
      e.taxId = t("supplier_portal.onboarding_inn.registry_unavailable");
    }
    if (state.headquartersAddress.trim().length < 5) e.headquartersAddress = "Street address is required";
    if (state.city.trim().length < 2) e.city = "City is required";
    return e;
  }

  async function submit() {
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length > 0) return;
    if (!innLookup || innLookup.status !== "FOUND") {
      await verifyInn();
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    setJoinPending(null);
    try {
      const body = JSON.stringify(state);
      const res = await supplierFetch("/v1/supplier/business/setup", {
        method: "POST",
        headers: {
          "Idempotency-Key": supplierBusinessSetupKey(supplierScopeId(), body),
        },
        body,
      });
      if (res.status === 409) {
        const conflict = (await res.json().catch(() => null)) as JoinRequestConflictPayload | null;
        if (conflict) {
          setJoinPending(conflict);
          return;
        }
      }
      if (!res.ok) {
        const errBody = await res.text();
        throw new Error(errBody || `business setup failed: ${res.status}`);
      }
      const data = (await res.json().catch(() => null)) as { token?: string } | null;
      if (data?.token) {
        persistSession(data.token);
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

      {joinPending && (
        <SetupCallout>
          <div className="flex flex-col gap-1">
            <strong>{t("supplier_portal.onboarding_inn.existing_company_title")}</strong>
            <span>
              {t("supplier_portal.onboarding_inn.existing_company_message", {
                legal_name: joinPending.legal_name || "Registered Supplier",
                inn: joinPending.normalized_inn || state.taxId,
              })}
            </span>
            <span className="text-xs font-mono">
              {t("supplier_portal.onboarding_inn.join_request_sent")} (#{joinPending.request_id})
            </span>
            <Link href="/auth/login" className="underline text-xs mt-1">
              {t("supplier_portal.join_requests.status_pending")}
            </Link>
          </div>
        </SetupCallout>
      )}

      <SetupSection
        icon="kyc"
        title={t("supplier_portal.setup.business.text.tax_and_registration")}
        description={t("supplier_portal.residual.text.required_for_compliance_and_payout_verification")}
      >
        <SetupField
          id="taxId"
          label={t("supplier_portal.onboarding_inn.tax_id_label")}
          error={errors.taxId || innError || undefined}
          hint={t("supplier_portal.onboarding_inn.tax_id_placeholder")}
        >
          <div className="flex flex-wrap gap-2 items-center">
            <SetupInput
              id="taxId"
              error={errors.taxId || innError || undefined}
              value={state.taxId}
              autoComplete="off"
              placeholder={t("supplier_portal.onboarding_inn.tax_id_placeholder")}
              onChange={(e) => {
                setInnLookup(null);
                setInnError(null);
                setState((s) => ({ ...s, taxId: e.target.value }));
              }}
            />
            <button
              type="button"
              disabled={verifyingInn}
              onClick={() => void verifyInn()}
              className="md-btn md-btn-outlined px-3 py-2 text-xs"
            >
              {verifyingInn
                ? t("supplier_portal.onboarding_inn.verifying")
                : t("supplier_portal.onboarding_inn.verify_button")}
            </button>
          </div>
        </SetupField>

        {innLookup?.status === "FOUND" && (
          <SetupCallout>
            <div className="flex flex-col gap-1 text-xs">
              <span className="font-semibold">
                {t("supplier_portal.onboarding_inn.verified_badge")}: {innLookup.legal_name} (
                {innLookup.normalized_inn})
              </span>
              {innLookup.existing_join_eligible && (
                <span>
                  {t("supplier_portal.onboarding_inn.existing_company_message", {
                    legal_name: innLookup.existing_legal_name || innLookup.legal_name || "",
                    inn: innLookup.normalized_inn,
                  })}
                </span>
              )}
            </div>
          </SetupCallout>
        )}

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
