"use client";

import { useCallback, useMemo } from "react";
import Link from "next/link";
import {
  ShoppingCart,
  PackageSearch,
  Inbox,
  Truck,
  Brain,
  Package,
  ArrowRight,
  ArrowUpRight,
  Layers3,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import EmptyState from "../../../components/EmptyState";
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
      const parsed = JSON.parse(
        localStorage.getItem("retailer_profile") || "null",
      ) as { id?: string } | null;
      return parsed?.id ?? "";
    } catch {
      return "";
    }
  };

  const retailerID = getProfileId();
  const ordersPath = retailerID
    ? `/v1/retailers/${retailerID}/orders`
    : "/v1/orders";

  const {
    data: orders,
    loading: loadingOrders,
    mutate: refreshOrders,
  } = useLiveData<Order[]>(ordersPath, 30000);
  const {
    data: predictions,
    loading: loadingPred,
    mutate: refreshPredictions,
  } = useLiveData<Prediction[]>("/v1/ai/predictions");
  const {
    data: products,
    loading: loadingProducts,
    mutate: refreshProducts,
  } = useLiveData<Product[]>("/v1/catalog/products");
  const { addToCart, items } = useCart();

  const orderList = orders ?? EMPTY_ORDERS;
  const predictionList = predictions ?? EMPTY_PREDICTIONS;
  const productList = products ?? EMPTY_PRODUCTS;
  const cartQuantity = items.reduce((total, item) => total + item.quantity, 0);

  const activeOrders = useMemo(
    () =>
      orderList.filter(
        (order) => order.state !== "COMPLETED" && order.state !== "CANCELLED",
      ),
    [orderList],
  );
  const completedOrders = useMemo(
    () => orderList.filter((order) => order.state === "COMPLETED"),
    [orderList],
  );
  const reorderProducts = useMemo(() => productList.slice(0, 8), [productList]);

  const loading = loadingOrders || loadingPred || loadingProducts;

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <AnimatePresence mode="popLayout">
        {loading ? (
          <motion.div
            key="loading"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <Skeleton className="mb-2 h-8 w-64 rounded-lg" />
            <Skeleton className="mb-8 h-4 w-96 rounded-lg" />
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
              {[0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="h-32 rounded-2xl animate-pulse bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]"
                />
              ))}
            </div>
          </motion.div>
        ) : (
          <motion.div
            key="content"
            layout
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
          >
            <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
              <div>
                <h1 className="md-typescale-display-small font-bold tracking-tight text-[var(--desk-text-primary)]">
                  Operations Hub
                </h1>
                <p className="mt-1 md-typescale-body-large text-[var(--desk-text-secondary)]">
                  Active deliveries, restock signals, and fleet telemetry.
                </p>
              </div>
              <div className="flex items-center gap-3">
                <Link href="/orders">
                  <Button
                    variant="flat"
                    className="h-10 px-5 rounded-xl font-bold text-[var(--desk-text-secondary)]"
                  >
                    Review Orders
                  </Button>
                </Link>
                <Link href="/catalog">
                  <Button
                    variant="solid"
                    className="h-10 px-5 rounded-xl font-bold shadow-[var(--shadow-sm)]"
                    style={{ background: "var(--desk-accent)", color: "white" }}
                  >
                    Open Catalog
                  </Button>
                </Link>
              </div>
            </header>

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
                    <h2 className="mt-2 text-2xl font-bold tracking-tight text-[var(--desk-text-primary)] leading-tight">
                      Predictive Replenishment
                    </h2>
                    <p className="mt-2 md-typescale-body-medium text-[var(--desk-text-secondary)]">
                      {activeOrders.length} inbound nodes and{" "}
                      {predictionList.length} AI signals are shaping your next
                      run.
                    </p>
                  </div>
                  <div className="px-3 py-1 rounded-lg bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] font-bold text-xs">
                    {productList.length} SKUS
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-3 mt-6">
                  <QuickAction
                    href="/catalog"
                    icon={PackageSearch}
                    label="Catalog"
                  />
                  <QuickAction
                    href="/orders"
                    icon={ShoppingCart}
                    label="Orders"
                  />
                  <QuickAction href="/insights" icon={Brain} label="Planning" />
                </div>
              </BentoCard>

              <KpiCard
                label="Active Nodes"
                value={activeOrders.length}
                sub={`${completedOrders.length} archived`}
                icon={
                  <Truck size={18} style={{ color: "var(--desk-accent)" }} />
                }
              />
              <KpiCard
                label="AI Signals"
                value={predictionList.length}
                sub="Restock Ready"
                icon={<Brain size={18} style={{ color: "var(--desk-info)" }} />}
              />
              <KpiCard
                label="Staged Cart"
                value={cartQuantity}
                sub="Items in queue"
                icon={
                  <ShoppingCart
                    size={18}
                    style={{ color: "var(--desk-success)" }}
                  />
                }
              />
              <KpiCard
                label="Suppliers"
                value={new Set(productList.map((p) => p.supplier_id)).size}
                sub="Active partners"
                icon={
                  <Layers3 size={18} style={{ color: "var(--desk-warning)" }} />
                }
              />
            </BentoGrid>

            <div className="grid gap-8 xl:grid-cols-[1.2fr_0.8fr]">
              <section>
                <h3 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)] mb-4">
                  Quick Reorder
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {reorderProducts.map((p) => (
                    <motion.button
                      key={p.id}
                      layout
                      whileHover={{ y: -2 }}
                      whileTap={{ scale: 0.98 }}
                      onClick={() => addToCart(p)}
                      className="flex items-center gap-4 p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl text-left hover:shadow-md transition-shadow group"
                    >
                      <div className="w-12 h-12 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-text-tertiary)] group-hover:bg-[var(--desk-accent-soft)] group-hover:text-[var(--desk-accent)] transition-colors">
                        <Package size={20} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="md-typescale-title-small font-bold truncate text-[var(--desk-text-primary)]">
                          {p.name}
                        </p>
                        <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] truncate uppercase tracking-widest">
                          {p.supplier_name}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
                          {p.price.toLocaleString()}
                        </p>
                        <ArrowUpRight
                          size={14}
                          className="ml-auto opacity-20 group-hover:opacity-100 transition-opacity"
                        />
                      </div>
                    </motion.button>
                  ))}
                </div>
              </section>

              <section>
                <div className="flex items-center justify-between mb-4">
                  <h3 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)]">
                    AI Restock
                  </h3>
                  <Link
                    href="/insights"
                    className="text-[var(--desk-accent)] md-typescale-label-small font-bold uppercase tracking-widest hover:underline"
                  >
                    View All
                  </Link>
                </div>
                <div className="flex flex-col gap-3">
                  {predictionList.slice(0, 5).map((forecast) => (
                    <div
                      key={forecast.id}
                      className="p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl flex items-center gap-4"
                    >
                      <div
                        className={`w-10 h-10 rounded-full border-2 flex items-center justify-center text-xs font-bold ${forecast.confidence > 0.8 ? "border-[var(--desk-success)] text-[var(--desk-success)]" : "border-[var(--desk-warning)] text-[var(--desk-warning)]"}`}
                      >
                        {Math.round(forecast.confidence * 100)}%
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="md-typescale-title-small font-bold truncate text-[var(--desk-text-primary)]">
                          {forecast.productName}
                        </p>
                        <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] line-clamp-1">
                          {forecast.reasoning}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
                          {forecast.predictedQuantity}
                        </p>
                        <p className="text-[10px] text-[var(--desk-text-tertiary)] uppercase font-bold tracking-tighter">
                          Units
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </section>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

function QuickAction({
  href,
  icon: Icon,
  label,
}: {
  href: string;
  icon: any;
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
      <span className="md-typescale-label-small font-bold uppercase tracking-widest">
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
  icon: any;
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
