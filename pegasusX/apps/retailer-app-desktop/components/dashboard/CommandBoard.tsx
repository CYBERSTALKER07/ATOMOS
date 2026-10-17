"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { ORDER_STATUS_FUNNEL, type RetailerControlTowerPulse } from "@pegasusx/types";
import { SourceChip, StatusStack } from "@pegasusx/ui-kit/portal";
import { LoyaltyCard } from "../LoyaltyCard";
import { moneyCurrency } from "../../lib/payment-catalog";

type CommandBoardProps = {
  pulse: RetailerControlTowerPulse | null;
  loading: boolean;
  error?: string | null;
};

function salesLabel(minor: number): string {
  const currency = moneyCurrency();
  if (currency) {
    return `${minor} ${currency}`;
  }
  return `${minor} minor`;
}

export function CommandBoard({ pulse, loading, error }: CommandBoardProps) {
  const router = useRouter();
  const source = pulse?.source || "empty";
  const stack = pulse?.orders_by_status ?? null;
  const facets = pulse?.orders_by_supplier ?? [];

  if (loading && !pulse) {
    return <p className="text-sm text-[var(--desk-text-secondary)]">Loading retailer command…</p>;
  }

  if (error) {
    return (
      <div className="mb-10" data-testid="gs-u-retailer-command">
        <p
          role="alert"
          data-testid="gs-u-retailer-command-error"
          className="rounded-2xl border border-[var(--desk-warning)]/30 bg-[var(--desk-warning)]/10 px-4 py-3 text-sm text-[var(--desk-warning)]"
        >
          {error}
        </p>
      </div>
    );
  }

  return (
    <div className="mb-10 space-y-6" data-testid="gs-u-retailer-command">
      <div className="flex flex-wrap items-center gap-3" data-testid="gs-u-retailer-source">
        <SourceChip source={source} />
        <p className="text-sm text-[var(--desk-text-secondary)]">
          Child orders by supplier. A stuck FISCAL_FAILED at one supplier stays visible here.
        </p>
      </div>

      {pulse?.empty ? (
        <div
          data-testid="gs-u-retailer-pulse-empty"
          className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-5"
        >
          <p className="text-base font-medium text-[var(--desk-text-primary)]">No live ops signals yet</p>
          <p className="mt-1 text-sm text-[var(--desk-text-secondary)]">
            Empty pulse — not demo tiles. Place an order, open POS, or start a shift.
          </p>
        </div>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" data-testid="gs-u-retailer-pulse-tiles">
          <PulseTile href="/orders" label="Open orders" value={pulse?.open_orders ?? 0} />
          <PulseTile href="/tracking" label="Fulfillment" value={pulse?.active_fulfillments ?? 0} />
          <PulseTile href="/dock" label="Dock pending" value={pulse?.dock_pending ?? 0} />
          <PulseTile href="/pos" label="POS sessions" value={pulse?.pos_open_sessions ?? 0} />
          <PulseTile href="/shifts" label="Open shifts" value={pulse?.open_shifts ?? 0} />
          <PulseTile href="/stock" label="Low stock bins" value={pulse?.low_stock_sku_bins ?? 0} />
          <PulseTile href="/insights" label="Sales 7d" value={salesLabel(pulse?.sales_minor_7d ?? 0)} />
          <PulseTile href="/auto-order" label="Auto-order place" value="off" />
        </div>
      )}

      <section
        className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-5 space-y-4"
        data-testid="gs-u-retailer-stack"
      >
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--desk-text-tertiary)]">
            Incoming
          </p>
          <h2 className="mt-1 text-xl font-light text-[var(--desk-text-primary)]">Orders by status</h2>
        </div>
        <StatusStack
          dictionary={ORDER_STATUS_FUNNEL}
          counts={stack}
          source={source}
          onSelect={(key) => router.push(`/orders?status=${key}`)}
        />
      </section>

      {facets.length > 0 ? (
        <section className="space-y-4" data-testid="gs-u-retailer-supplier-facet">
          <h3 className="text-sm font-medium text-[var(--desk-text-primary)]">By supplier (child orders)</h3>
          <div className="grid gap-4 xl:grid-cols-2">
            {facets.map((facet) => (
              <div
                key={facet.supplier_id || "missing"}
                className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-4 space-y-3"
                data-supplier={facet.supplier_id}
              >
                <p className="text-xs uppercase tracking-wide text-[var(--desk-text-tertiary)]">
                  {facet.supplier_id || "missing supplier"}
                </p>
                <StatusStack
                  dictionary={ORDER_STATUS_FUNNEL}
                  counts={facet.orders_by_status}
                  source={source}
                  onSelect={(key) =>
                    router.push(
                      `/orders?status=${key}&supplier=${encodeURIComponent(facet.supplier_id)}`,
                    )
                  }
                />
              </div>
            ))}
          </div>
        </section>
      ) : null}

      <div className="grid gap-4 md:grid-cols-2">
        <Link
          href="/tracking"
          className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-4 text-sm text-[var(--desk-text-secondary)]"
        >
          Tracking map — open live fulfillments. No invented route series.
        </Link>
        <Link
          href="/insights"
          className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-4 text-sm text-[var(--desk-text-secondary)]"
        >
          Insights peek — sell-through lives on /insights, not a fake H-bar here.
        </Link>
      </div>

      <div data-testid="gs-u-retailer-loyalty">
        <LoyaltyCard />
      </div>
    </div>
  );
}

function PulseTile({
  href,
  label,
  value,
}: {
  href: string;
  label: string;
  value: string | number;
}) {
  return (
    <Link
      href={href}
      className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-4 py-3"
    >
      <p className="text-[10px] uppercase tracking-widest text-[var(--desk-text-tertiary)]">{label}</p>
      <p className="mt-1 text-lg font-light text-[var(--desk-text-primary)]">{value}</p>
    </Link>
  );
}
