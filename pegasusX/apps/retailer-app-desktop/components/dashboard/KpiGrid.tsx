"use client";

import { usePortalT } from "@/lib/i18n";
import React from "react";
import Link from "next/link";
import { ShoppingCart, PackageSearch, Brain, Truck, Layers3 } from "lucide-react";
import { BentoGrid, BentoCard } from "../BentoGrid";
import CountUp from "../CountUp";

interface KpiGridProps {
  activeOrdersLength: number;
  predictionListLength: number;
  productListLength: number;
  cartQuantity: number;
  completedOrdersLength: number;
  blockedPredictionCount: number;
  uniqueSuppliersCount: number;
}

export function KpiGrid({
  activeOrdersLength,
  predictionListLength,
  productListLength,
  cartQuantity,
  completedOrdersLength,
  blockedPredictionCount,
  uniqueSuppliersCount,
}: KpiGridProps) {
  const t = usePortalT();
  return (
    <BentoGrid className="mb-10">
      <BentoCard
        span={2}
        interactive={false}
        className="flex flex-col justify-between"
      >
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
              Focus today
            </span>
            <h2 className="mt-2 text-2xl font-light tracking-tight text-[var(--desk-text-primary)] leading-tight">
              Predictive Replenishment
            </h2>
            <p className="mt-2 md-typescale-body-medium text-[var(--desk-text-secondary)]">
              {activeOrdersLength} inbound nodes and{" "}
              {predictionListLength} AI signals are shaping your next
              run.
            </p>
          </div>
          <div className="px-3 py-1 rounded-lg bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] font-light text-xs">
            {productListLength} SKUS
          </div>
        </div>

        <div className="grid grid-cols-3 gap-3 mt-6">
          <QuickAction href="/catalog" icon={PackageSearch} label={t("portal.nav.section.catalog")} />
          <QuickAction href="/orders" icon={ShoppingCart} label={t("portal.nav.orders")} />
          <QuickAction href="/insights" icon={Brain} label={t("supplier_portal.replenishment.suggestions.text.reorder_suggestions")} />
        </div>
      </BentoCard>

      <KpiCard
        label={t("retailer_desktop.residual.text.active_nodes")}
        value={activeOrdersLength}
        sub={`${completedOrdersLength} archived`}
        icon={<Truck size={18} style={{ color: "var(--desk-accent)" }} />}
      />
      <KpiCard
        label={t("retailer_desktop.residual.text.ai_signals")}
        value={predictionListLength}
        sub={
          blockedPredictionCount > 0
            ? `${blockedPredictionCount} blocked (sparse history)`
            : predictionListLength === 0
              ? "None pending"
              : "Pending confirm"
        }
        icon={<Brain size={18} style={{ color: "var(--desk-info)" }} />}
      />
      <KpiCard
        label={t("retailer_desktop.residual.text.staged_cart")}
        value={cartQuantity}
        sub="Items in queue"
        icon={
          <ShoppingCart size={18} style={{ color: "var(--desk-success)" }} />
        }
      />
      <KpiCard
        label={t("retailer_desktop.residual.text.suppliers")}
        value={uniqueSuppliersCount}
        sub="Active partners"
        icon={<Layers3 size={18} style={{ color: "var(--desk-warning)" }} />}
      />
    </BentoGrid>
  );
}

function QuickAction({
  href,
  icon: Icon,
  label,
}: {
  href: string;
  icon: React.ElementType;
  label: string;
}) {
  return (
    <Link
      href={href}
      className="flex flex-col items-center gap-2 p-3 bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)] rounded-xl hover:bg-[var(--desk-accent-soft)] hover:border-[var(--desk-accent)] hover:text-[var(--desk-accent)] transition-all active:scale-95 group"
    >
      <Icon
        size={20}
        strokeWidth={1.5}
        className="group-hover:scale-110 transition-transform"
      />
      <span className="md-typescale-label-small font-light uppercase tracking-widest">
        {label}
      </span>
    </Link>
  );
}

function KpiCard({
  label,
  value,
  sub,
  icon,
}: {
  label: string;
  value: number;
  sub: string;
  icon: React.ReactNode;
}) {
  return (
    <BentoCard interactive={false}>
      <div className="flex items-center justify-between mb-2">
        <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
          {label}
        </span>
        {icon}
      </div>
      <CountUp
        end={value}
        className="md-typescale-metric text-[var(--desk-text-primary)]"
      />
      <p className="md-typescale-body-small text-[var(--desk-text-secondary)] mt-1">
        {sub}
      </p>
    </BentoCard>
  );
}
