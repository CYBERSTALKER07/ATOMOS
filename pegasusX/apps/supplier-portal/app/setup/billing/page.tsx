"use client";

import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { supplierFetch } from "@/lib/auth";
import {
  SelectionOption,
  SetupCallout,
  SetupField,
  SetupFooter,
  SetupInput,
  SetupPageHeader,
  SetupSection,
} from "@/components/setup/SetupPrimitives";

// /setup/billing — bank + payment-gateway configuration.
// Decoupled from the 4-step registration wizard to reduce friction.
// Hard product invariant: this page must remain a post-registration step
// that suppliers reach AFTER /auth/register. Do not merge into the wizard.

const GATEWAYS = [
  {
    id: "GLOBAL_PAY",
    label: t("supplier_portal.billing_setup.gateway.global_pay.label"),
    description: t("supplier_portal.residual.text.card_payments_via_the_pegasusx_global_rail"),
    icon: "global",
  },
  {
    id: "ADYEN",
    label: t("supplier_portal.residual.text.adyen"),
    description: t("supplier_portal.residual.text.enterprise_card_acquiring_with_local_payment_methods"),
    icon: "payment",
  },
  {
    id: "AIRWALLEX",
    label: t("supplier_portal.residual.text.airwallex"),
    description: t("supplier_portal.residual.text.cross_border_payouts_and_multi_currency_settlement"),
    icon: "treasury",
  },
  {
    id: "CASH",
    label: t("supplier_portal.residual.text.cash_on_delivery"),
    description: t("supplier_portal.residual.text.retailers_collect_cash_reconcile_manually"),
    icon: "pricing",
  },
] as const;

type GatewayId = (typeof GATEWAYS)[number]["id"];

type BillingState = {
  bankName: string;
  accountHolder: string;
  accountNumber: string;
  swiftBic: string;
  iban: string;
  selectedGateways: GatewayId[];
  paymentAcceptor: "SUPPLIER" | "WAREHOUSE";
};

const INITIAL: BillingState = {
  bankName: "",
  accountHolder: "",
  accountNumber: "",
  swiftBic: "",
  iban: "",
  selectedGateways: ["GLOBAL_PAY", "CASH"],
  paymentAcceptor: "SUPPLIER",
};

const ACCEPTOR_OPTIONS = [
  {
    id: "SUPPLIER" as const,
    label: t("supplier_portal.residual.text.supplier_accepts_payments"),
    description: t("supplier_portal.residual.text.card_revenue_settles_to_your_supplier_treasury_account"),
    icon: "supplier",
  },
  {
    id: "WAREHOUSE" as const,
    label: t("supplier_portal.residual.text.warehouse_accepts_payments"),
    description: t("supplier_portal.residual.text.fulfilling_nodes_collect_payment_per_dispatch_lane"),
    icon: "warehouse",
  },
];

export default function BillingSetupPage() {
  const t = usePortalT();
  const [state, setState] = useState<BillingState>(INITIAL);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (state.bankName.trim().length < 2) e.bankName = "Bank name is required";
    if (state.accountHolder.trim().length < 2) e.accountHolder = "Account holder name is required";
    if (state.accountNumber.trim().length < 4) e.accountNumber = "Account number is required";
    if (state.swiftBic.trim().length < 4) e.swiftBic = "SWIFT / BIC is required";
    if (state.selectedGateways.length === 0) e.selectedGateways = "Choose at least one gateway";
    return e;
  }

  function toggleGateway(id: GatewayId) {
    setState((s) => {
      const set = new Set(s.selectedGateways);
      if (set.has(id)) set.delete(id);
      else set.add(id);
      return { ...s, selectedGateways: Array.from(set) as GatewayId[] };
    });
    setErrors((prev) => {
      const next = { ...prev };
      delete next.selectedGateways;
      return next;
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

  function skip() {
    window.location.href = "/?billing=skipped";
  }

  function goBack() {
    window.location.href = "/setup/business";
  }

  return (
    <>
      <SetupPageHeader
        icon="treasury"
        title={t("supplier_portal.setup.billing.text.billing_and_payment_gateways")}
        subtitle={t("supplier_portal.residual.text.configure_where_payouts_land_and_which_checkout_rails_retailers_")}
      />

      <SetupCallout>
        Bank details are transmitted securely. You can add or change gateways anytime under Treasury settings.
      </SetupCallout>

      <SetupSection
        icon="treasury"
        title={t("supplier_portal.setup.billing.text.payout_bank_account")}
        description={t("supplier_portal.residual.text.where_supplier_earnings_are_deposited_after_settlement")}
      >
        <SetupField id="bankName" label={t("supplier_portal.residual.text.bank_name")} error={errors.bankName}>
          <SetupInput
            id="bankName"
            error={errors.bankName}
            value={state.bankName}
            autoComplete="organization"
            placeholder={t("supplier_portal.setup.billing.text.e_g_hsbc_chase")}
            onChange={(e) => setState((s) => ({ ...s, bankName: e.target.value }))}
          />
        </SetupField>
        <SetupField id="accountHolder" label={t("supplier_portal.residual.text.account_holder_name")} error={errors.accountHolder}>
          <SetupInput
            id="accountHolder"
            error={errors.accountHolder}
            value={state.accountHolder}
            autoComplete="name"
            onChange={(e) => setState((s) => ({ ...s, accountHolder: e.target.value }))}
          />
        </SetupField>
        <div className="setup-grid-2">
          <SetupField id="accountNumber" label={t("supplier_portal.residual.text.account_number")} error={errors.accountNumber}>
            <SetupInput
              id="accountNumber"
              error={errors.accountNumber}
              value={state.accountNumber}
              inputMode="numeric"
              autoComplete="off"
              onChange={(e) => setState((s) => ({ ...s, accountNumber: e.target.value }))}
            />
          </SetupField>
          <SetupField id="swiftBic" label={t("supplier_portal.residual.text.swift_bic")} error={errors.swiftBic}>
            <SetupInput
              id="swiftBic"
              error={errors.swiftBic}
              value={state.swiftBic}
              autoComplete="off"
              placeholder={t("supplier_portal.setup.billing.text.e_g_chasus33")}
              onChange={(e) => setState((s) => ({ ...s, swiftBic: e.target.value }))}
            />
          </SetupField>
        </div>
        <SetupField id="iban" label="IBAN" optional hint="Optional for non-IBAN regions.">
          <SetupInput
            id="iban"
            value={state.iban}
            autoComplete="off"
            onChange={(e) => setState((s) => ({ ...s, iban: e.target.value }))}
          />
        </SetupField>
      </SetupSection>

      <SetupSection
        icon="payment"
        title={t("supplier_portal.setup.billing.text.payment_acceptor")}
        description={t("supplier_portal.residual.text.who_receives_card_revenue_when_an_order_is_fulfilled")}
      >
        <div className="setup-selection-grid setup-selection-grid--2" role="radiogroup" aria-label={t("supplier_portal.setup.billing.text.payment_acceptor")}>
          {ACCEPTOR_OPTIONS.map((option) => (
            <SelectionOption
              key={option.id}
              selected={state.paymentAcceptor === option.id}
              title={option.label}
              description={option.description}
              icon={option.icon}
              checkType="single"
              onClick={() => setState((s) => ({ ...s, paymentAcceptor: option.id }))}
            />
          ))}
        </div>
      </SetupSection>

      <SetupSection
        icon="global"
        title={t("supplier_portal.setup.billing.text.payment_gateways")}
        description={t("supplier_portal.residual.text.retailers_see_these_options_at_checkout_select_all_that_apply")}
      >
        <div className="setup-selection-grid setup-selection-grid--2" role="group" aria-label={t("supplier_portal.setup.billing.text.payment_gateways")}>
          {GATEWAYS.map((gateway) => (
            <SelectionOption
              key={gateway.id}
              selected={state.selectedGateways.includes(gateway.id)}
              title={gateway.label}
              description={gateway.description}
              icon={gateway.icon}
              checkType="multi"
              onClick={() => toggleGateway(gateway.id)}
            />
          ))}
        </div>
        {errors.selectedGateways ? (
          <p className="setup-helper setup-helper--error" role="alert" style={{ marginTop: "var(--space-3)" }}>
            {errors.selectedGateways}
          </p>
        ) : null}
      </SetupSection>

      {submitError ? <SetupCallout variant="error">{submitError}</SetupCallout> : null}

      <SetupFooter
        back={{ label: t("common.action.back"), onClick: goBack, disabled: submitting }}
        primary={{ label: t("supplier_portal.residual.text.complete_setup"), onClick: submit, disabled: submitting, loading: submitting }}
      />

      <div style={{ marginTop: "var(--space-4)", textAlign: "center" }}>
        <button
          type="button"
          className="setup-btn setup-btn--ghost"
          onClick={skip}
          disabled={submitting}
        >
          Skip for now — finish later
        </button>
      </div>
    </>
  );
}

function cryptoRandomId() {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
