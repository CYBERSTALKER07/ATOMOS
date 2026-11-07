"use client";

import { usePortalT } from "@/lib/i18n";
import { Package, MapPin, Users, Truck } from "lucide-react";
import { BentoGrid, BentoCard } from "../BentoGrid";
import CountUp from "../CountUp";
import { PageSection } from "../PageSection";
import type { TrackingOrder } from "../../lib/types";

function formatAmount(amount: number): string {
  return amount.toLocaleString("en-US").replace(/,/g, " ");
}

interface TrackingStatusProps {
  activeFulfillmentCount: number;
  approachingCount: number;
  suppliers: { id: string; name: string }[];
  avgItems: number;
  recentReceipts: TrackingOrder[];
  selectedSupplierIds: Set<string>;
  toggleSupplier: (id: string) => void;
}

export function TrackingStatus({
  activeFulfillmentCount,
  approachingCount,
  suppliers,
  avgItems,
  recentReceipts,
  selectedSupplierIds,
  toggleSupplier,
}: TrackingStatusProps) {
  const t = usePortalT();
  return (
    <>
      <BentoGrid className="mb-2">
        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Active Deliveries
              </span>
              <Package size={18} style={{ color: "var(--desk-accent)" }} />
            </div>
            <CountUp
              end={activeFulfillmentCount}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Inbound orders in motion
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Approaching
              </span>
              <MapPin size={18} style={{ color: "var(--desk-success)" }} />
            </div>
            <CountUp
              end={approachingCount}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Immediate vicinity
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Suppliers
              </span>
              <Users size={18} style={{ color: "var(--desk-info)" }} />
            </div>
            <CountUp
              end={suppliers.length}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Contracted partners
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Avg Items / Order
              </span>
              <Truck size={18} style={{ color: "var(--desk-warning)" }} />
            </div>
            <CountUp
              end={avgItems}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Basket density
            </p>
          </div>
        </BentoCard>
      </BentoGrid>

      {recentReceipts.length > 0 && (
        <PageSection
          title={t("retailer_desktop.tracking.tracking_status.text.recent_receipts")}
          description={t("retailer_desktop.residual.text.completed_deliveries_from_the_tracking_feed")}
        >
          <div className="space-y-2 max-h-40 overflow-y-auto !mt-0">
            {recentReceipts.slice(0, 6).map((receipt) => {
              const st = (receipt.state || "").toUpperCase();
              const fiscalLabel =
                st === "FISCALIZING"
                  ? "Pending fiscal"
                  : st === "FISCAL_FAILED"
                    ? "Fiscal failed"
                    : st === "COMPLETED"
                      ? "Fiscalized"
                      : receipt.state || "—";
              const canOpen =
                st === "COMPLETED" ||
                Boolean(receipt.fiscal_qr) ||
                Boolean(receipt.latest_fiscal_receipt_id);
              return (
              <div
                key={receipt.order_id}
                className="flex items-center justify-between gap-2 rounded-xl border border-[var(--desk-border)] px-3 py-2"
              >
                <div className="min-w-0">
                  <p className="text-sm font-light text-[var(--desk-text-primary)]">
                    {receipt.supplier_name || "Supplier"}
                  </p>
                  <p className="text-[10px] font-mono text-[var(--desk-text-tertiary)]">
                    #{receipt.order_id.slice(-8)} · {fiscalLabel}
                  </p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  {canOpen ? (
                    <>
                      <button
                        type="button"
                        className="text-[11px] text-[var(--desk-accent)] underline-offset-2 hover:underline"
                        onClick={() => {
                          void import("@/lib/order-receipt").then((m) =>
                            m.openTrackingReceipt(receipt).catch(() => undefined),
                          );
                        }}
                      >
                        View
                      </button>
                      <button
                        type="button"
                        className="text-[11px] text-[var(--desk-text-tertiary)] underline-offset-2 hover:underline"
                        onClick={() => {
                          void import("@/lib/order-receipt").then((m) =>
                            m
                              .openRetailerOrderReceipt(receipt.order_id, "pdf")
                              .catch(() => undefined),
                          );
                        }}
                      >
                        PDF
                      </button>
                    </>
                  ) : null}
                  <span className="text-sm font-light tabular-nums">
                    {formatAmount(receipt.total_amount)}
                  </span>
                </div>
              </div>
            );
            })}
          </div>
        </PageSection>
      )}

      {suppliers.length > 1 && (
        <div className="flex flex-wrap items-center gap-2">
          {suppliers.map((s) => {
            const active =
              selectedSupplierIds.size === 0 || selectedSupplierIds.has(s.id);
            return (
              <button
                key={s.id}
                onClick={() => toggleSupplier(s.id)}
                className={`px-5 py-2 rounded-full md-typescale-label-large font-light transition-all ${
                  active
                    ? "bg-[var(--desk-accent)] text-white shadow-[var(--shadow-sm)]"
                    : "bg-[var(--desk-surface)] text-[var(--desk-text-secondary)] border border-[var(--desk-border)] hover:bg-[var(--desk-surface-subtle)]"
                }`}
              >
                {s.name}
              </button>
            );
          })}
        </div>
      )}
    </>
  );
}
