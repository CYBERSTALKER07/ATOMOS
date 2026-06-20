"use client";

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
    label: "Global Pay",
    description: "Card payments via the pegasusX global rail.",
    icon: "global",
  },
  {
    id: "ADYEN",
    label: "Adyen",
    description: "Enterprise card acquiring with local payment methods.",
    icon: "payment",
  },
  {
    id: "AIRWALLEX",
    label: "Airwallex",
    description: "Cross-border payouts and multi-currency settlement.",
    icon: "treasury",
  },
  {
    id: "CASH",
    label: "Cash on delivery",
    description: "Retailers collect cash; reconcile manually.",
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
    label: "Supplier accepts payments",
    description: "Card revenue settles to your supplier treasury account.",
    icon: "supplier",
  },
  {
    id: "WAREHOUSE" as const,
    label: "Warehouse accepts payments",
    description: "Fulfilling nodes collect payment per dispatch lane.",
    icon: "warehouse",
  },
];

export default function BillingSetupPage() {
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
        title="Billing & payment gateways"
        subtitle="Configure where payouts land and which checkout rails retailers can use."
      />

      <SetupCallout>
        Bank details are transmitted securely. You can add or change gateways anytime under Treasury settings.
      </SetupCallout>

      <SetupSection
        icon="treasury"
        title="Payout bank account"
        description="Where supplier earnings are deposited after settlement."
      >
        <SetupField id="bankName" label="Bank name" error={errors.bankName}>
          <SetupInput
            id="bankName"
            error={errors.bankName}
            value={state.bankName}
            autoComplete="organization"
            placeholder="e.g. HSBC, Chase"
            onChange={(e) => setState((s) => ({ ...s, bankName: e.target.value }))}
          />
        </SetupField>
        <SetupField id="accountHolder" label="Account holder name" error={errors.accountHolder}>
          <SetupInput
            id="accountHolder"
            error={errors.accountHolder}
            value={state.accountHolder}
            autoComplete="name"
            onChange={(e) => setState((s) => ({ ...s, accountHolder: e.target.value }))}
          />
        </SetupField>
        <div className="setup-grid-2">
          <SetupField id="accountNumber" label="Account number" error={errors.accountNumber}>
            <SetupInput
              id="accountNumber"
              error={errors.accountNumber}
              value={state.accountNumber}
              inputMode="numeric"
              autoComplete="off"
              onChange={(e) => setState((s) => ({ ...s, accountNumber: e.target.value }))}
            />
          </SetupField>
          <SetupField id="swiftBic" label="SWIFT / BIC" error={errors.swiftBic}>
            <SetupInput
              id="swiftBic"
              error={errors.swiftBic}
              value={state.swiftBic}
              autoComplete="off"
              placeholder="e.g. CHASUS33"
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
        title="Payment acceptor"
        description="Who receives card revenue when an order is fulfilled."
      >
        <div className="setup-selection-grid setup-selection-grid--2" role="radiogroup" aria-label="Payment acceptor">
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
        title="Payment gateways"
        description="Retailers see these options at checkout. Select all that apply."
      >
        <div className="setup-selection-grid setup-selection-grid--2" role="group" aria-label="Payment gateways">
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
        back={{ label: "Back", onClick: goBack, disabled: submitting }}
        primary={{ label: "Complete setup", onClick: submit, disabled: submitting, loading: submitting }}
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
