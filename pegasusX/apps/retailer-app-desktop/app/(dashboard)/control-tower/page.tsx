"use client";

import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import { apiFetch } from "@/lib/auth";
import { Loader2, Activity, Package, ShoppingCart, Clock, HandHelping } from "lucide-react";

type Pulse = {
  retailer_id: string;
  generated_at: string;
  open_orders: number;
  active_fulfillments: number;
  dock_pending: number;
  pos_open_sessions: number;
  open_shifts: number;
  open_assist_tickets: number;
  low_stock_sku_bins: number;
  shift_variances_7d: number;
  sales_minor_7d: number;
  capabilities: string[];
  empty: boolean;
};

type Tile = {
  label: string;
  value: string | number;
  href?: string;
  icon: React.ElementType;
};

function formatMoney(minor: number) {
  return (minor / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export default function ControlTowerPage() {
  const t = usePortalT();
  const [pulse, setPulse] = useState<Pulse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [reconcileEpoch, setReconcileEpoch] = useState(0);

  useRetailerSessionReconcile(() => {
    setReconcileEpoch((epoch) => epoch + 1);
  });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch("/v1/retailer/control-tower/pulse");
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error((body as { error?: string }).error || `pulse_${res.status}`);
      }
      setPulse((await res.json()) as Pulse);
      setError(null);
    } catch {
      setError("control_tower_pulse_failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, reconcileEpoch]);

  const tiles: Tile[] = pulse
    ? [
        { label: t("retailer_desktop.residual.text.open_orders"), value: pulse.open_orders, href: "/orders", icon: Package },
        {
          label: t("retailer_desktop.residual.text.active_fulfillment"),
          value: pulse.active_fulfillments,
          href: "/tracking",
          icon: Activity,
        },
        { label: t("retailer_desktop.residual.text.dock_pending"), value: pulse.dock_pending, href: "/dock", icon: Package },
        {
          label: t("retailer_desktop.residual.text.pos_open_sessions"),
          value: pulse.pos_open_sessions,
          href: "/pos",
          icon: ShoppingCart,
        },
        { label: t("retailer_desktop.residual.text.open_shifts"), value: pulse.open_shifts, href: "/shifts", icon: Clock },
        {
          label: t("retailer_desktop.residual.text.assist_tickets"),
          value: pulse.open_assist_tickets,
          href: "/assist",
          icon: HandHelping,
        },
        { label: t("retailer_desktop.residual.text.low_stock_bins"), value: pulse.low_stock_sku_bins, href: "/stock", icon: Package },
        {
          label: t("retailer_desktop.residual.text.shift_variances_closed"),
          value: pulse.shift_variances_7d,
          href: "/shifts",
          icon: Clock,
        },
        {
          label: t("retailer_desktop.residual.text.sales_7d"),
          value: formatMoney(pulse.sales_minor_7d),
          href: "/reports",
          icon: Activity,
        },
      ]
    : [];

  return (
    <div className="relative flex min-h-[calc(100vh-64px)] flex-col gap-6 overflow-hidden bg-[#0a0a0a] p-6 text-white">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("retailer_desktop.control_tower.text.retailer_ops_pulse")}</h1>
          <p className="text-sm text-gray-400">
            Live counts from your shop — empty when quiet, never demo data.
          </p>
          {pulse && (
            <p className="mt-1 text-xs text-gray-500">
              Updated {pulse.generated_at.slice(0, 19)} · packs:{" "}
              {(pulse.capabilities || []).join(", ") || "CORE"}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={() => void load()}
          className="inline-flex items-center gap-2 rounded-lg border border-white/15 bg-white/5 px-3 py-2 text-sm hover:bg-white/10"
        >
          {loading && <Loader2 className="h-4 w-4 animate-spin" />}
          Refresh
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {error}
        </div>
      )}

      {loading && !pulse && (
        <div className="flex flex-1 items-center justify-center text-gray-400">
          <Loader2 className="mr-2 h-5 w-5 animate-spin" />
          Loading pulse…
        </div>
      )}

      {pulse?.empty && !loading && !error && (
        <div className="flex flex-1 flex-col items-center justify-center rounded-xl border border-dashed border-white/15 bg-white/[0.03] px-6 py-16 text-center">
          <Activity className="mb-4 h-10 w-10 text-emerald-400/80" />
          <h2 className="text-lg font-semibold">{t("retailer_desktop.control_tower.text.no_live_ops_signals_yet")}</h2>
          <p className="mt-2 max-w-md text-sm text-gray-400">
            Place wholesale orders, enable store stock or POS, open a shift, or create an assist
            ticket. This surface stays empty until real activity exists — it never shows mock
            charts.
          </p>
          <div className="mt-6 flex flex-wrap justify-center gap-2">
            <Link
              href="/orders"
              className="rounded-lg border border-white/15 px-3 py-1.5 text-sm hover:bg-white/10"
            >
              Orders
            </Link>
            <Link
              href="/stock"
              className="rounded-lg border border-white/15 px-3 py-1.5 text-sm hover:bg-white/10"
            >
              Store stock
            </Link>
            <Link
              href="/pos"
              className="rounded-lg border border-white/15 px-3 py-1.5 text-sm hover:bg-white/10"
            >
              POS
            </Link>
          </div>
        </div>
      )}

      {pulse && !pulse.empty && (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {tiles.map((tile) => {
            const Icon = tile.icon;
            const inner = (
              <div className="flex h-full flex-col rounded-xl border border-white/10 bg-white/[0.04] p-4 transition hover:border-emerald-500/40 hover:bg-white/[0.06]">
                <div className="mb-3 flex items-center gap-2 text-xs uppercase tracking-wide text-gray-400">
                  <Icon className="h-3.5 w-3.5 text-emerald-400" />
                  {tile.label}
                </div>
                <div className="text-2xl font-semibold tabular-nums">{tile.value}</div>
              </div>
            );
            return tile.href ? (
              <Link key={tile.label} href={tile.href}>
                {inner}
              </Link>
            ) : (
              <div key={tile.label}>{inner}</div>
            );
          })}
        </div>
      )}

      <p className="text-xs text-gray-600">
        Retailer Control Tower is an ops digest for your org, not the supplier telematics
        playbook. Network graph / H3 maps are omitted unless a real retailer feed is wired.
      </p>
    </div>
  );
}
