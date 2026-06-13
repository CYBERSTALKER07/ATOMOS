"use client";

import { useEffect, useState } from "react";
import type { SupplierPricingRule, SupplierPricingRuleUpdateRequest } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

type Draft = {
  base_markup_bps: string;
  retailer_discount_bps: string;
  min_margin_bps: string;
  currency: string;
};

function draftFromRule(rule: SupplierPricingRule): Draft {
  return {
    base_markup_bps: String(rule.base_markup_bps),
    retailer_discount_bps: String(rule.retailer_discount_bps),
    min_margin_bps: String(rule.min_margin_bps),
    currency: rule.currency,
  };
}

function parseBps(value: string, field: string): number | null {
  const parsed = Number.parseInt(value.trim(), 10);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return null;
  }
  if (parsed > 100_000) {
    throw new Error(`${field} exceeds maximum`);
  }
  return parsed;
}

export default function PricingPage() {
  const [rule, setRule] = useState<SupplierPricingRule | null>(null);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadRule = () => {
    setLoading(true);
    setError(null);
    api
      .getSupplierPricingRule()
      .then((loaded) => {
        setRule(loaded);
        setDraft(draftFromRule(loaded));
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load pricing"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadRule();
  }, []);

  const saveRule = async () => {
    if (!draft) return;
    setSaving(true);
    setError(null);
    try {
      const body: SupplierPricingRuleUpdateRequest = {};
      const baseMarkup = parseBps(draft.base_markup_bps, "base_markup_bps");
      const retailerDiscount = parseBps(draft.retailer_discount_bps, "retailer_discount_bps");
      const minMargin = parseBps(draft.min_margin_bps, "min_margin_bps");
      if (baseMarkup === null || retailerDiscount === null || minMargin === null) {
        throw new Error("Markup and margin values must be non-negative integers");
      }
      body.base_markup_bps = baseMarkup;
      body.retailer_discount_bps = retailerDiscount;
      body.min_margin_bps = minMargin;
      const currency = draft.currency.trim().toUpperCase();
      if (currency.length === 3) {
        body.currency = currency;
      }
      const updated = await api.updateSupplierPricingRule(body);
      setRule(updated);
      setDraft(draftFromRule(updated));
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_failed");
    } finally {
      setSaving(false);
    }
  };

  return (
    <PortalSurface
      title="Pricing authority"
      description="Supplier-controlled markup and retailer discount rules."
      loading={loading}
      error={error}
      empty={!rule || !draft}
    >
      {rule && draft ? (
        <div className="md-card p-6 space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Field
              label="Base markup (bps)"
              value={draft.base_markup_bps}
              onChange={(value) => setDraft((prev) => (prev ? { ...prev, base_markup_bps: value } : prev))}
            />
            <Field
              label="Retailer discount (bps)"
              value={draft.retailer_discount_bps}
              onChange={(value) => setDraft((prev) => (prev ? { ...prev, retailer_discount_bps: value } : prev))}
            />
            <Field
              label="Min margin (bps)"
              value={draft.min_margin_bps}
              onChange={(value) => setDraft((prev) => (prev ? { ...prev, min_margin_bps: value } : prev))}
            />
            <Field
              label="Currency"
              value={draft.currency}
              onChange={(value) => setDraft((prev) => (prev ? { ...prev, currency: value.toUpperCase() } : prev))}
            />
            <Metric label="Version" value={String(rule.rule_version)} />
            <Metric label="Updated" value={rule.updated_at ? new Date(rule.updated_at).toLocaleString() : "—"} />
          </div>
          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              className="md-btn md-btn-filled md-typescale-label-large px-6 py-2"
              disabled={saving}
              onClick={() => void saveRule()}
            >
              {saving ? "Saving…" : "Save pricing rule"}
            </button>
            <button
              type="button"
              className="md-btn md-btn-outlined md-typescale-label-large px-6 py-2"
              disabled={saving}
              onClick={() => setDraft(draftFromRule(rule))}
            >
              Reset
            </button>
          </div>
        </div>
      ) : null}
    </PortalSurface>
  );
}

function Field({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label className="block">
      <div className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</div>
      <input
        className="md-input-outlined mt-1 w-full px-3 py-2"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</div>
      <div className="md-typescale-headline-small font-semibold mt-1">{value}</div>
    </div>
  );
}
