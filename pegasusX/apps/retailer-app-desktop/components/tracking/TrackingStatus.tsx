import { Package, MapPin, Users, Truck } from "lucide-react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import { PageSection } from "../../../components/PageSection";
import type { TrackingOrder } from "../../../lib/types";

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
          title="Recent receipts"
          description="Completed deliveries from the tracking feed."
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
              return (
              <div
                key={receipt.order_id}
                className="flex items-center justify-between rounded-xl border border-[var(--desk-border)] px-3 py-2"
              >
                <div>
                  <p className="text-sm font-light text-[var(--desk-text-primary)]">
                    {receipt.supplier_name || "Supplier"}
                  </p>
                  <p className="text-[10px] font-mono text-[var(--desk-text-tertiary)]">
                    #{receipt.order_id.slice(-8)} · {fiscalLabel}
                  </p>
                </div>
                <span className="text-sm font-light tabular-nums">
                  {formatAmount(receipt.total_amount)}
                </span>
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
