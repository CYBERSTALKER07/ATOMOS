"use client";

import { useEffect, useState } from "react";
import { ApiClient } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import type { SupplierPricingRule } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function PricingPage() {
  const [rule, setRule] = useState<SupplierPricingRule | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierPricingRule()
      .then(setRule)
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load pricing"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PortalSurface
      title="Pricing authority"
      description="Supplier-controlled markup and retailer discount rules."
      loading={loading}
      error={error}
      empty={!rule}
    >
      {rule ? (
        <div className="md-card p-6 grid grid-cols-1 md:grid-cols-3 gap-4">
          <Metric label="Base markup (bps)" value={String(rule.base_markup_bps)} />
          <Metric label="Retailer discount (bps)" value={String(rule.retailer_discount_bps)} />
          <Metric label="Min margin (bps)" value={String(rule.min_margin_bps)} />
          <Metric label="Currency" value={rule.currency} />
          <Metric label="Version" value={String(rule.rule_version)} />
        </div>
      ) : null}
    </PortalSurface>
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
