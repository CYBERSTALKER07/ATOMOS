"use client";

import { AnimatePresence } from "framer-motion";
import { PackageOpen, ArrowUpRight } from "lucide-react";
import { Chip } from "@heroui/react";
import { VirtualScrollList } from "@pegasusx/ui-kit/desktop";
import EmptyState from "../EmptyState";
import { ListRowSkeleton } from "../Skeleton";
import type { Order } from "../../lib/types";

interface OrderListProps {
  loading: boolean;
  filtered: Order[];
  listEmptyState: {
    headline: string;
    body: string;
    variant: "no-results" | "restricted" | "offline" | "error" | "no-orders";
    action: string;
    onAction: () => void;
  };
  selectedId: string | null;
  setSelectedId: (id: string) => void;
  chipCfg: Record<string, { color: "warning" | "success" | "default" | "danger"; label: string }>;
  list: Order[];
}

export function OrderList({
  loading,
  filtered,
  listEmptyState,
  selectedId,
  setSelectedId,
  chipCfg,
  list,
}: OrderListProps) {
  return (
    <AnimatePresence mode="popLayout">
      {loading ? (
        <div className="flex flex-col gap-2">
          <ListRowSkeleton count={4} />
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          headline={listEmptyState.headline}
          body={listEmptyState.body}
          variant={listEmptyState.variant}
          action={listEmptyState.action}
          onAction={listEmptyState.onAction}
        />
      ) : (
        <VirtualScrollList
          height="calc(100vh - 440px)"
          items={filtered}
          itemKey={(order) => order.order_id}
          renderItem={(order) => {
            const isSelected = (selectedId ?? list[0]?.order_id) === order.order_id;
            const c = chipCfg[order.state] || chipCfg.PENDING;
            return (
              <button
                type="button"
                onClick={() => setSelectedId(order.order_id)}
                className={`mb-2 flex w-full items-center gap-4 rounded-2xl border p-4 text-left transition-all group ${
                  isSelected
                    ? "bg-[var(--desk-surface)] border-[var(--desk-accent)] shadow-md ring-2 ring-[var(--desk-accent-soft)]"
                    : "bg-[var(--desk-surface)] border-[var(--desk-border)] hover:border-[var(--desk-border-strong)]"
                }`}
              >
                <div
                  className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-xl transition-colors ${isSelected ? "bg-[var(--desk-accent-soft)] text-[var(--desk-accent)]" : "bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)] group-hover:text-[var(--desk-text-secondary)]"}`}
                >
                  <PackageOpen size={20} />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="mb-1 flex items-center justify-between">
                    <span className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                      #{order.order_id.slice(-8)}
                    </span>
                    <span
                      className={`rounded-md px-2 py-0.5 text-[10px] font-light uppercase tracking-widest ${c.color === "success" ? "bg-green-100 text-green-700" : c.color === "warning" ? "bg-orange-100 text-orange-700" : "bg-gray-100 text-gray-700"}`}
                    >
                      {c.label}
                    </span>
                  </div>
                  <p className="md-typescale-body-small truncate text-[var(--desk-text-tertiary)]">
                    {order.payment_gateway || "UNSPECIFIED"}
                  </p>
                  {(order.preorder_badge === "REVIEW_DELIVERY" ||
                    order.confirmation_status === "PENDING_WAREHOUSE") && (
                    <Chip size="sm" color="warning" variant="soft" className="mt-1">
                      Review Delivery
                    </Chip>
                  )}
                </div>
                <div className="text-right">
                  <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                    {order.amount.toLocaleString()}
                  </p>
                  <ArrowUpRight
                    size={14}
                    className={`ml-auto transition-opacity ${isSelected ? "opacity-100 text-[var(--desk-accent)]" : "opacity-20 group-hover:opacity-100"}`}
                  />
                </div>
              </button>
            );
          }}
        />
      )}
    </AnimatePresence>
  );
}
