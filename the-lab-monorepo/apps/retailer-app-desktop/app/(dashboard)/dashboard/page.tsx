"use client";

import { useMemo } from "react";
import Link from "next/link";
import {
  ShoppingCart, PackageSearch, Activity, BarChart3, Settings,
  Inbox, Clock, Search, User, Truck, Brain, ArrowUpRight,
  Package, RefreshCcw, AlertTriangle,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import { useLiveData } from "../../../lib/hooks";
import { useCart } from "../../../lib/cart";
import { apiFetch } from "../../../lib/auth";
import type { Order, Prediction, Product } from "../../../lib/types";

export default function DashboardPage() {
  const { data: orders, loading: loadingOrders } = useLiveData<Order[]>("/v1/orders", 30000);
  const { data: predictions, loading: loadingPred } = useLiveData<Prediction[]>("/v1/ai/predictions");
  const { data: products } = useLiveData<Product[]>("/v1/catalog/products");
  const { addToCart } = useCart();

  const orderList = orders ?? [];
  const predList = predictions ?? [];
  const productList = products ?? [];

  const activeOrders = useMemo(
    () => orderList.filter((o) => o.state !== "COMPLETED" && o.state !== "CANCELLED"),
    [orderList],
  );

  const reorderProducts = productList.slice(0, 8);

  const loading = loadingOrders || loadingPred;

  if (loading) {
    return (
      <div className="min-h-full p-6 md:p-8">
        <Skeleton className="h-8 w-48 rounded-lg mb-2" />
        <Skeleton className="h-4 w-80 rounded-lg mb-8" />
        <div className="grid grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
          {[0, 1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} className="h-32 rounded-2xl" />
          ))}
        </div>
        <div className="grid grid-cols-2 gap-6">
          <Skeleton className="h-64 rounded-2xl" />
          <Skeleton className="h-64 rounded-2xl" />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8">
      {/* Header */}
      <header className="mb-8">
        <h1 className="md-typescale-headline-large">Hub</h1>
        <p className="md-typescale-body-medium mt-1" style={{ color: "var(--muted)" }}>
          Quick access to everything. Your operations at a glance.
        </p>
      </header>

      {/* Service Grid — Yandex Go inspired */}
      <div className="grid grid-cols-2 lg:grid-cols-3 gap-4 mb-10">
        {/* Row 1: two big tiles */}
        <ServiceTile href="/catalog" icon={PackageSearch} title="Catalog" subtitle="Browse products" height="h-32" />
        <ServiceTile href="/insights" icon={Brain} title="AI Insights" subtitle={`${predList.length} predictions`} height="h-32" />

        {/* Row 2: orders wide + stacked small */}
        <ServiceTile href="/orders" icon={ShoppingCart} title="Orders" subtitle={`${activeOrders.length} active`} height="h-28" />
        <div className="flex flex-col gap-4">
          <ServiceTile href="/settings" icon={Inbox} title="Inbox" height="h-[calc(50%-8px)]" compact />
          <ServiceTile href="/orders" icon={Clock} title="History" height="h-[calc(50%-8px)]" compact />
        </div>

        {/* Row 3: three equal small tiles */}
        <ServiceTile href="/procurement" icon={Activity} title="Procurement" compact height="h-20" />
        <ServiceTile href="/catalog" icon={Search} title="Search" compact height="h-20" />
        <ServiceTile href="/settings" icon={User} title="Profile" compact height="h-20" />
      </div>

      {/* Quick Reorder + AI Predictions side by side */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Quick Reorder */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <RefreshCcw size={16} style={{ color: "var(--muted)" }} />
            <h2 className="md-typescale-title-large font-semibold text-foreground">Quick Reorder</h2>
          </div>

          {reorderProducts.length === 0 ? (
            <div className="bento-card flex flex-col items-center justify-center py-10 gap-2">
              <Package size={32} style={{ color: "var(--muted)" }} />
              <p className="md-typescale-body-medium text-muted">No products available</p>
            </div>
          ) : (
            <div className="flex gap-3 overflow-x-auto pb-2 scrollbar-hide">
              {reorderProducts.map((product) => (
                <button
                  key={product.id}
                  onClick={() => addToCart(product)}
                  className="flex flex-col items-center gap-2 p-3 shrink-0 cursor-pointer group"
                >
                  <div
                    className="w-16 h-16 rounded-2xl flex items-center justify-center transition-transform group-hover:scale-105"
                    style={{ background: "var(--surface)" }}
                  >
                    {product.image_url ? (
                      <img src={product.image_url} alt={product.name} className="w-full h-full rounded-2xl object-cover" />
                    ) : (
                      <Package size={24} style={{ color: "var(--muted)", opacity: 0.5 }} />
                    )}
                  </div>
                  <span className="md-typescale-label-small font-medium text-foreground text-center w-[72px] truncate">
                    {product.name}
                  </span>
                  <span className="md-typescale-label-small font-bold text-foreground tabular-nums">
                    {product.price.toLocaleString()}
                  </span>
                </button>
              ))}
            </div>
          )}
        </section>

        {/* AI Predictions */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Brain size={16} style={{ color: "var(--accent)" }} />
            <h2 className="md-typescale-title-large font-semibold text-foreground">AI Predictions</h2>
            <Chip size="sm" color="default" variant="soft" className="ml-1">{predList.length}</Chip>
          </div>

          {predList.length === 0 ? (
            <div className="bento-card flex flex-col items-center justify-center py-10 gap-2">
              <Brain size={32} style={{ color: "var(--muted)" }} />
              <p className="md-typescale-body-medium text-muted">No predictions yet</p>
            </div>
          ) : (
            <div className="flex flex-col gap-2 max-h-[360px] overflow-y-auto pr-1">
              {predList.slice(0, 6).map((forecast) => (
                <PredictionCard key={forecast.id} forecast={forecast} />
              ))}
              {predList.length > 6 && (
                <Link
                  href="/insights"
                  className="md-typescale-label-large text-accent font-semibold text-center py-2 hover:underline"
                >
                  View all {predList.length} predictions
                </Link>
              )}
            </div>
          )}
        </section>
      </div>

      {/* Active Deliveries */}
      {activeOrders.length > 0 && (
        <section className="mt-10">
          <div className="flex items-center gap-2 mb-4">
            <Truck size={16} style={{ color: "var(--muted)" }} />
            <h2 className="md-typescale-title-large font-semibold text-foreground">Active Deliveries</h2>
            <Chip size="sm" color="warning" variant="soft" className="ml-1">{activeOrders.length}</Chip>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
            {activeOrders.slice(0, 6).map((order) => (
              <Link key={order.order_id} href="/orders" className="bento-card hover:ring-1 hover:ring-accent transition-all">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0" style={{ background: "var(--surface)" }}>
                    <Truck size={18} style={{ color: "var(--muted)" }} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <span className="md-typescale-title-small font-semibold text-foreground truncate block">
                      #{order.order_id.slice(-8)}
                    </span>
                    <span className="md-typescale-body-small text-muted">{order.state.replace(/_/g, " ")}</span>
                  </div>
                  <span className="md-typescale-label-large font-semibold tabular-nums shrink-0">
                    {order.amount.toLocaleString()}
                  </span>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

/* ── Service Tile Component ── */

function ServiceTile({
  href,
  icon: Icon,
  title,
  subtitle,
  height = "h-28",
  compact = false,
}: {
  href: string;
  icon: React.ElementType;
  title: string;
  subtitle?: string;
  height?: string;
  compact?: boolean;
}) {
  return (
    <Link
      href={href}
      className={`bento-card flex flex-col justify-end cursor-pointer group hover:ring-1 hover:ring-accent transition-all ${height}`}
      style={{ padding: compact ? "12px 16px" : "16px 20px" }}
    >
      <Icon
        size={compact ? 22 : 28}
        strokeWidth={1.5}
        className="mb-2 text-foreground transition-transform group-hover:scale-110"
      />
      <span className={`font-semibold text-foreground ${compact ? "md-typescale-label-medium" : "md-typescale-title-small"}`}>
        {title}
      </span>
      {subtitle && (
        <span className="md-typescale-label-small text-muted mt-0.5">{subtitle}</span>
      )}
    </Link>
  );
}

/* ── Prediction Card ── */

function PredictionCard({ forecast }: { forecast: Prediction }) {
  const confidenceColor =
    forecast.confidence >= 0.8
      ? "var(--success)"
      : forecast.confidence >= 0.6
        ? "var(--warning)"
        : "var(--danger)";

  const pct = Math.round(forecast.confidence * 100);

  return (
    <div className="bento-card">
      <div className="flex items-center gap-3">
        {/* Confidence ring */}
        <div className="shrink-0 relative w-11 h-11">
          <svg width={44} height={44} className="rotate-[-90deg]">
            <circle cx={22} cy={22} r={19} fill="none" stroke="var(--border)" strokeWidth={3} />
            <circle
              cx={22} cy={22} r={19}
              fill="none"
              stroke={confidenceColor}
              strokeWidth={3}
              strokeLinecap="round"
              strokeDasharray={`${2 * Math.PI * 19}`}
              strokeDashoffset={`${2 * Math.PI * 19 * (1 - forecast.confidence)}`}
            />
          </svg>
          <span
            className="absolute inset-0 flex items-center justify-center text-[10px] font-bold"
            style={{ color: confidenceColor }}
          >
            {pct}%
          </span>
        </div>

        <div className="flex-1 min-w-0">
          <span className="md-typescale-title-small font-semibold text-foreground truncate block">
            {forecast.productName || `Prediction ${forecast.id.slice(-6)}`}
          </span>
          <span className="md-typescale-body-small text-muted line-clamp-1">
            {forecast.reasoning || "AI recommendation"}
          </span>
        </div>

        <div className="text-right shrink-0">
          <span className="md-typescale-title-small font-bold tabular-nums text-foreground block">
            {forecast.predictedQuantity}
          </span>
          <span className="md-typescale-label-small text-muted">units</span>
        </div>
      </div>
    </div>
  );
}
