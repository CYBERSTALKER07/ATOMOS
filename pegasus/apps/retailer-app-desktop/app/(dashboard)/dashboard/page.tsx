"use client";

import { useMemo } from "react";
import Link from "next/link";
import {
  ShoppingCart,
  PackageSearch,
  Inbox,
  Truck,
  Brain,
  Package,
  RefreshCcw,
  ArrowRight,
  ArrowUpRight,
  Layers3,
} from "lucide-react";
import { Chip, Skeleton } from "@heroui/react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import EmptyState from "../../../components/EmptyState";
import PageTransition from "../../../components/PageTransition";
import { useLiveData } from "../../../lib/hooks";
import { useCart } from "../../../lib/cart";
import type { Order, Prediction, Product } from "../../../lib/types";

const EMPTY_ORDERS: Order[] = [];
const EMPTY_PREDICTIONS: Prediction[] = [];
const EMPTY_PRODUCTS: Product[] = [];

export default function DashboardPage() {
  const getProfileId = () => {
    if (typeof localStorage === "undefined") return "";
    try {
      const parsed = JSON.parse(localStorage.getItem("retailer_profile") || "null") as { id?: string } | null;
      return parsed?.id ?? "";
    } catch {
      return "";
    }
  };

  const retailerID = getProfileId();
  const ordersPath = retailerID ? `/v1/retailers/${retailerID}/orders` : "/v1/orders";
  const { data: orders, loading: loadingOrders } = useLiveData<Order[]>(ordersPath, 30000);
  const { data: predictions, loading: loadingPred } = useLiveData<Prediction[]>("/v1/ai/predictions");
  const { data: products } = useLiveData<Product[]>("/v1/catalog/products");
  const { addToCart, items } = useCart();

  const orderList = orders ?? EMPTY_ORDERS;
  const predictionList = predictions ?? EMPTY_PREDICTIONS;
  const productList = products ?? EMPTY_PRODUCTS;
  const cartQuantity = items.reduce((total, item) => total + item.quantity, 0);

  const activeOrders = useMemo(
    () => orderList.filter((order) => order.state !== "COMPLETED" && order.state !== "CANCELLED"),
    [orderList],
  );
  const completedOrders = useMemo(
    () => orderList.filter((order) => order.state === "COMPLETED"),
    [orderList],
  );
  const reorderProducts = useMemo(() => productList.slice(0, 8), [productList]);
  const loading = loadingOrders || loadingPred;

  if (loading) {
    return (
      <div className="min-h-full p-6 md:p-8">
        <Skeleton className="mb-2 h-8 w-56 rounded-lg" />
        <Skeleton className="mb-8 h-4 w-96 rounded-lg" />
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {[0, 1, 2, 3].map((item) => (
            <Skeleton key={item} className="h-32 rounded-2xl" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <PageTransition className="min-h-full p-6 md:p-8">
      <header className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="md-typescale-headline-large">Retailer operations hub</h1>
          <p className="mt-1 md-typescale-body-medium" style={{ color: "var(--muted)" }}>
            Review active deliveries, restock signals, and fast reorder actions from one workspace.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Link href="/orders" className="md-btn md-btn-outlined md-typescale-label-large inline-flex h-10 items-center px-5">
            Review orders
          </Link>
          <Link href="/catalog" className="md-btn md-btn-filled md-typescale-label-large inline-flex h-10 items-center px-5">
            Open catalog
          </Link>
        </div>
      </header>

      <BentoGrid className="mb-8">
        <BentoCard span={2} className="lg:min-h-[240px]">
          <div className="flex h-full flex-col justify-between gap-6">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="max-w-2xl">
                <p className="md-typescale-label-small uppercase tracking-[0.16em] text-muted">Today&apos;s focus</p>
                <h2 className="mt-2 text-2xl font-semibold tracking-tight text-foreground">Keep replenishment predictable and deliveries confirmed.</h2>
                <p className="mt-2 max-w-xl md-typescale-body-medium text-muted">
                  {activeOrders.length} active deliveries and {predictionList.length} AI restock signals are currently shaping your next procurement run.
                </p>
              </div>
              <Chip size="sm" color="default" variant="soft" className="font-semibold">
                {productList.length} catalog SKUs available
              </Chip>
            </div>

            <div className="grid gap-3 sm:grid-cols-3">
              <WorkspaceActionCard
                href="/catalog"
                icon={PackageSearch}
                title="Supplier catalog"
                subtitle="Search SKUs, compare partners, and add products to the cart."
              />
              <WorkspaceActionCard
                href="/orders"
                icon={ShoppingCart}
                title="Order desk"
                subtitle="Verify inbound deliveries and review order exceptions."
              />
              <WorkspaceActionCard
                href="/insights"
                icon={Brain}
                title="AI planning"
                subtitle="Inspect demand predictions before the next procurement cycle."
              />
            </div>
          </div>
        </BentoCard>

        <KpiCard
          label="Active deliveries"
          value={activeOrders.length}
          supporting={`${completedOrders.length} completed`}
          icon={<Truck size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />}
        />
        <KpiCard
          label="Restock signals"
          value={predictionList.length}
          supporting="AI recommendations ready"
          icon={<Brain size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />}
        />
        <KpiCard
          label="Cart quantity"
          value={cartQuantity}
          supporting="Items staged for checkout"
          icon={<ShoppingCart size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />}
        />
        <KpiCard
          label="Supplier coverage"
          value={new Set(productList.map((product) => product.supplier_id)).size}
          supporting="Partners represented in catalog"
          icon={<Layers3 size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />}
        />
      </BentoGrid>

      <div className="grid gap-8 xl:grid-cols-[1.2fr_0.8fr]">
        <section>
          <div className="mb-4 flex items-center gap-2">
            <RefreshCcw size={16} style={{ color: "var(--muted)" }} />
            <h2 className="md-typescale-title-large font-semibold text-foreground">Quick reorder</h2>
          </div>

          {reorderProducts.length === 0 ? (
            <div className="py-10">
              <EmptyState 
                imageUrl="/images/empty-products.png"
                headline="No products available"
                body="Your catalog is currently empty."
              />
            </div>
          ) : (
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {reorderProducts.map((product) => (
                <button
                  key={product.id}
                  onClick={() => addToCart(product)}
                  className="bento-card flex cursor-pointer flex-col gap-4 text-left transition-all duration-150 hover:ring-1 hover:ring-accent"
                >
                  <div className="flex items-center justify-between gap-3">
                    <div className="flex min-w-0 items-center gap-3">
                      <div
                        className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl"
                        style={{ background: "var(--surface)" }}
                      >
                        <Package size={20} style={{ color: "var(--muted)" }} />
                      </div>
                      <div className="min-w-0">
                        <p className="truncate md-typescale-title-small font-semibold text-foreground">{product.name}</p>
                        <p className="truncate md-typescale-body-small text-muted">{product.supplier_name}</p>
                      </div>
                    </div>
                    <ArrowUpRight size={16} style={{ color: "var(--muted)" }} />
                  </div>
                  <div className="flex items-end justify-between gap-3">
                    <div>
                      <p className="md-typescale-label-small uppercase tracking-[0.14em] text-muted">Price</p>
                      <p className="md-typescale-title-medium font-semibold tabular-nums text-foreground">
                        {product.price.toLocaleString()}
                      </p>
                    </div>
                    <span className="rounded-full bg-[var(--surface)] px-3 py-1 md-typescale-label-small font-semibold text-foreground">
                      Add item
                    </span>
                  </div>
                </button>
              ))}
            </div>
          )}
        </section>

        <section>
          <div className="mb-4 flex items-center gap-2">
            <Brain size={16} style={{ color: "var(--accent)" }} />
            <h2 className="md-typescale-title-large font-semibold text-foreground">AI restock queue</h2>
            <Chip size="sm" color="default" variant="soft">{predictionList.length}</Chip>
          </div>

          {predictionList.length === 0 ? (
            <div className="py-10">
              <EmptyState 
                imageUrl="/images/empty-predictions.png"
                headline="No predictions yet"
                body="AI recommendations will appear here."
              />
            </div>
          ) : (
            <div className="flex max-h-[420px] flex-col gap-3 overflow-y-auto pr-1">
              {predictionList.slice(0, 6).map((prediction) => (
                <PredictionCard key={prediction.id} forecast={prediction} />
              ))}
              <Link href="/insights" className="inline-flex items-center gap-2 px-2 md-typescale-label-large font-semibold text-accent hover:underline">
                Review all predictions
                <ArrowRight size={16} />
              </Link>
            </div>
          )}
        </section>
      </div>

      {activeOrders.length > 0 && (
        <section className="mt-10">
          <div className="mb-4 flex items-center gap-2">
            <Inbox size={16} style={{ color: "var(--muted)" }} />
            <h2 className="md-typescale-title-large font-semibold text-foreground">Active deliveries</h2>
            <Chip size="sm" color="warning" variant="soft">{activeOrders.length}</Chip>
          </div>
          <div className="grid gap-3 xl:grid-cols-3">
            {activeOrders.slice(0, 6).map((order) => (
              <Link key={order.order_id} href="/orders" className="bento-card transition-all duration-150 hover:ring-1 hover:ring-accent">
                <div className="flex items-center gap-3">
                  <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl" style={{ background: "var(--surface)" }}>
                    <Truck size={18} style={{ color: "var(--muted)" }} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate md-typescale-title-small font-semibold text-foreground">#{order.order_id.slice(-8)}</p>
                    <p className="truncate md-typescale-body-small text-muted">{order.state.replace(/_/g, " ")}</p>
                  </div>
                  <span className="md-typescale-label-large font-semibold tabular-nums text-foreground">
                    {order.amount.toLocaleString()}
                  </span>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}
    </PageTransition>
  );
}

function WorkspaceActionCard({
  href,
  icon: Icon,
  title,
  subtitle,
}: {
  href: string;
  icon: React.ElementType;
  title: string;
  subtitle: string;
}) {
  return (
    <Link
      href={href}
      className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4 transition-all duration-150 hover:border-[var(--color-md-outline)] hover:bg-[var(--color-md-surface-container-low)]"
    >
      <div className="flex items-center justify-between gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-[var(--color-md-surface-container)]">
          <Icon size={18} strokeWidth={1.6} className="text-foreground" />
        </div>
        <ArrowRight size={16} style={{ color: "var(--muted)" }} />
      </div>
      <p className="mt-4 md-typescale-title-small font-semibold text-foreground">{title}</p>
      <p className="mt-1 md-typescale-body-small text-muted">{subtitle}</p>
    </Link>
  );
}

function KpiCard({
  label,
  value,
  supporting,
  icon,
}: {
  label: string;
  value: number;
  supporting: string;
  icon: React.ReactNode;
}) {
  return (
    <BentoCard>
      <div className="md-kpi-card">
        <div className="flex items-center justify-between">
          <span className="md-kpi-label">{label}</span>
          {icon}
        </div>
        <CountUp end={value} className="md-kpi-value" />
        <span className="md-kpi-sub">{supporting}</span>
      </div>
    </BentoCard>
  );
}

function PredictionCard({ forecast }: { forecast: Prediction }) {
  const confidenceColor =
    forecast.confidence >= 0.8
      ? "var(--success)"
      : forecast.confidence >= 0.6
        ? "var(--warning)"
        : "var(--danger)";
  const confidence = Math.round(forecast.confidence * 100);

  return (
    <div className="bento-card">
      <div className="flex items-center gap-3">
        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-[var(--border)] text-sm font-bold" style={{ color: confidenceColor }}>
          {confidence}%
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate md-typescale-title-small font-semibold text-foreground">
            {forecast.productName || `Prediction ${forecast.id.slice(-6)}`}
          </p>
          <p className="line-clamp-2 md-typescale-body-small text-muted">
            {forecast.reasoning || "AI recommendation"}
          </p>
        </div>
        <div className="shrink-0 text-right">
          <p className="md-typescale-title-small font-bold tabular-nums text-foreground">{forecast.predictedQuantity}</p>
          <p className="md-typescale-label-small text-muted">units</p>
        </div>
      </div>
    </div>
  );
}
