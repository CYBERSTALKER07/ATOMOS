"use client";

import { useMemo, useState, useCallback } from "react";
import {
  Building2,
  Plus,
  ChevronRight,
  Package,
  HandCoins,
  ArrowUpRight,
  TrendingDown,
  FileText,
  AlertTriangle,
  X,
  Search,
  Loader2,
  Trash2,
  RefreshCw,
} from "lucide-react";
import { Button, Chip } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
import EmptyState from "../../../components/EmptyState";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { useOptionalWebSocket } from "../../../lib/ws";
import type { Supplier, RetailerAnalytics } from "../../../lib/types";

type LoadIssue = "restricted" | "offline" | "error";

export default function ProcurementPage() {
  const {
    data: suppliers,
    loading: loadingSuppliers,
    error: suppliersError,
    isRefreshing: isSuppliersRefreshing,
    mutate: mutateSuppliers,
  } = useLiveData<Supplier[]>("/v1/retailer/suppliers");
  const {
    data: analytics,
    error: analyticsError,
    isRefreshing: isAnalyticsRefreshing,
    mutate: mutateAnalytics,
  } = useLiveData<RetailerAnalytics>("/v1/retailer/analytics/expenses");
  const ws = useOptionalWebSocket();

  const supplierList = suppliers ?? [];
  const totalSpend = analytics?.total_this_month ?? 0;
  const lastMonthSpend = analytics?.total_last_month ?? 0;
  const topSuppliers = analytics?.top_suppliers ?? [];
  const isRefreshing = isSuppliersRefreshing || isAnalyticsRefreshing;

  const refreshAll = useCallback(() => {
    void mutateSuppliers();
    void mutateAnalytics();
  }, [mutateAnalytics, mutateSuppliers]);

  const loadIssue = useMemo<LoadIssue | null>(() => {
    const errors = [suppliersError, analyticsError].filter(Boolean) as Array<
      Error & { status?: number }
    >;
    if (errors.length === 0) return null;
    if (errors.some((err) => err.status === 401 || err.status === 403)) {
      return "restricted";
    }
    if (
      (typeof navigator !== "undefined" && !navigator.onLine) ||
      errors.some((err) =>
        /Failed to fetch|NetworkError|Load failed/i.test(err.message),
      )
    ) {
      return "offline";
    }
    return "error";
  }, [analyticsError, suppliersError]);

  const syncBanner = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Procurement access is partially restricted for this account.",
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Offline mode active. Showing latest procurement data.",
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Procurement sync degraded. Auto-retry is active.",
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Live socket reconnecting. Updates may arrive with delay.",
      };
    }
    if (isRefreshing && !loadingSuppliers) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: "Syncing procurement feeds...",
      };
    }
    return null;
  }, [isRefreshing, loadIssue, loadingSuppliers, ws]);

  const suppliersEmptyState = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        headline: "Procurement access restricted",
        body: "Your account cannot load vendor connections right now.",
        variant: "restricted" as const,
        action: "Retry",
        onAction: refreshAll,
      };
    }
    if (loadIssue === "offline") {
      return {
        headline: "Procurement is offline",
        body: "Reconnect to refresh vendor and spend feeds.",
        variant: "offline" as const,
        action: "Retry",
        onAction: refreshAll,
      };
    }
    if (loadIssue === "error") {
      return {
        headline: "Vendor feed unavailable",
        body: "Connected vendor data could not be loaded right now.",
        variant: "error" as const,
        action: "Retry",
        onAction: refreshAll,
      };
    }
    return {
      headline: "No vendors connected",
      body: undefined,
      variant: "no-data" as const,
      action: "Connect Vendor",
      onAction: () => setShowAddModal(true),
    };
  }, [loadIssue, refreshAll]);

  const sparkSpend = useMemo(() => {
    const monthly = analytics?.monthly_expenses ?? [];
    if (monthly.length > 0) return monthly.map((m) => m.total);
    return Array.from(
      { length: 12 },
      (_, i) => 800 + i * 45 + Math.sin(i * 0.9) * 120,
    );
  }, [analytics]);

  const sparkOrders = useMemo(
    () =>
      Array.from({ length: 12 }, (_, i) => 20 + i * 3 + Math.cos(i * 0.6) * 8),
    [],
  );
  const totalOrders = supplierList.reduce((s, v) => s + v.order_count, 0);

  const [showAddModal, setShowAddModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<Supplier[]>([]);
  const [searching, setSearching] = useState(false);
  const [addingId, setAddingId] = useState<string | null>(null);
  const [removingId, setRemovingId] = useState<string | null>(null);

  const searchSuppliers = useCallback(
    async (q: string) => {
      setSearchQuery(q);
      if (q.length < 2) {
        setSearchResults([]);
        return;
      }
      setSearching(true);
      try {
        const res = await apiFetch(
          `/v1/catalog/suppliers/search?q=${encodeURIComponent(q)}`,
        );
        if (res.ok) {
          const data = await res.json();
          const existing = new Set(supplierList.map((s) => s.id));
          setSearchResults(
            (data ?? []).filter((s: Supplier) => !existing.has(s.id)),
          );
        }
      } catch {
        /* swallow */
      } finally {
        setSearching(false);
      }
    },
    [supplierList],
  );

  const addSupplier = useCallback(
    async (supplierId: string) => {
      setAddingId(supplierId);
      try {
        const res = await apiFetch(`/v1/retailer/suppliers/${supplierId}/add`, {
          method: "POST",
          headers: { "Idempotency-Key": `retailer-supplier-add:${supplierId}` },
        });
        if (res.ok) {
          mutateSuppliers();
          setSearchResults((prev) => prev.filter((s) => s.id !== supplierId));
        }
      } catch {
        /* swallow */
      } finally {
        setAddingId(null);
      }
    },
    [mutateSuppliers],
  );

  const removeSupplier = useCallback(
    async (supplierId: string) => {
      setRemovingId(supplierId);
      try {
        const res = await apiFetch(
          `/v1/retailer/suppliers/${supplierId}/remove`,
          {
            method: "POST",
            headers: {
              "Idempotency-Key": `retailer-supplier-remove:${supplierId}`,
            },
          },
        );
        if (res.ok) mutateSuppliers();
      } catch {
        /* swallow */
      } finally {
        setRemovingId(null);
      }
    },
    [mutateSuppliers],
  );

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="md-typescale-display-small font-light tracking-tight text-[var(--desk-text-primary)]">
            Vendor Operations
          </h1>
          <p className="mt-1 md-typescale-body-large text-[var(--desk-text-secondary)]">
            Lifecycle management for connected supply nodes and trade
            settlements.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="secondary"
            isDisabled={isRefreshing}
            onPress={refreshAll}
            className="h-11 px-5 rounded-xl font-light text-[var(--desk-text-secondary)]"
          >
            <RefreshCw
              size={16}
              className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`}
            />
            {isRefreshing ? "Syncing" : "Sync"}
          </Button>
          <Button
            variant="primary"
            onPress={() => setShowAddModal(true)}
            className="h-11 px-6 rounded-xl font-light transition-all shadow-[var(--shadow-sm)]"
            style={{ background: "var(--desk-accent)", color: "white" }}
          >
            <Plus size={18} className="mr-2" /> Connect Vendor
          </Button>
        </div>
      </header>

      {syncBanner && (
        <motion.div
          initial={{ opacity: 0, y: -6 }}
          animate={{ opacity: 1, y: 0 }}
          className={`mb-6 flex items-center justify-between gap-3 rounded-2xl border px-4 py-3 ${
            syncBanner.kind === "refreshing"
              ? "border-[var(--desk-info)]/30 bg-[var(--desk-info)]/5 text-[var(--desk-info)]"
              : "border-[var(--desk-warning)]/30 bg-[var(--desk-warning)]/10 text-[var(--desk-warning)]"
          }`}
        >
          <div className="flex items-center gap-2">
            <syncBanner.icon
              size={16}
              className={syncBanner.kind === "refreshing" ? "animate-spin" : ""}
            />
            <span className="md-typescale-body-small font-light uppercase tracking-wide">
              {syncBanner.message}
            </span>
          </div>
          {syncBanner.kind !== "refreshing" && (
            <button
              onClick={refreshAll}
              className="rounded-lg border border-current/30 px-3 py-1 text-[11px] font-light uppercase tracking-wide hover:bg-current/10"
            >
              Retry
            </button>
          )}
        </motion.div>
      )}

      <BentoGrid className="mb-8">
        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Trade Spend
              </span>
              <HandCoins size={18} style={{ color: "var(--desk-accent)" }} />
            </div>
            <div className="flex items-end justify-between">
              <CountUp
                end={totalSpend}
                className="md-typescale-metric text-[var(--desk-text-primary)]"
              />
              <MiniSparkline data={sparkSpend} width={80} height={32} />
            </div>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Network Orders
              </span>
              <Package size={18} style={{ color: "var(--desk-info)" }} />
            </div>
            <div className="flex items-end justify-between">
              <CountUp
                end={totalOrders}
                className="md-typescale-metric text-[var(--desk-text-primary)]"
              />
              <MiniSparkline data={sparkOrders} width={80} height={32} />
            </div>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Baseline
              </span>
              <AlertTriangle
                size={18}
                style={{ color: "var(--desk-warning)" }}
              />
            </div>
            <CountUp
              end={lastMonthSpend}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Previous period
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Concentration
              </span>
              <TrendingDown
                size={18}
                style={{ color: "var(--desk-success)" }}
              />
            </div>
            <CountUp
              end={topSuppliers.length}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Primary trade partners
            </p>
          </div>
        </BentoCard>
      </BentoGrid>

      <div className="flex gap-8 min-h-[520px]">
        {/* Main: Vendor List */}
        <div className="flex-1 flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)]">
              Connected Supply Nodes
            </h2>
            <span className="md-typescale-label-small font-light uppercase tracking-widest text-[var(--desk-text-tertiary)]">
              {supplierList.length} ACTIVE
            </span>
          </div>

          <AnimatePresence mode="popLayout">
            {loadingSuppliers ? (
              [0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="h-24 rounded-2xl animate-pulse bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]"
                />
              ))
            ) : supplierList.length === 0 ? (
              <EmptyState
                headline={suppliersEmptyState.headline}
                body={suppliersEmptyState.body}
                variant={suppliersEmptyState.variant}
                action={suppliersEmptyState.action}
                onAction={suppliersEmptyState.onAction}
              />
            ) : (
              <div className="flex flex-col gap-2">
                {supplierList.map((vendor) => (
                  <motion.div
                    key={vendor.id}
                    layout
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="flex items-center gap-4 p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl hover:border-[var(--desk-border-strong)] hover:shadow-md transition-all group"
                  >
                    <div className="w-12 h-12 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center shrink-0 border border-[var(--desk-border)] overflow-hidden">
                      {vendor.logo_url ? (
                        <img
                          src={vendor.logo_url}
                          alt={vendor.name}
                          className="w-full h-full object-cover"
                        />
                      ) : (
                        <Building2
                          size={20}
                          className="text-[var(--desk-text-tertiary)]"
                        />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-3 mb-1">
                        <span className="md-typescale-title-small font-light text-[var(--desk-text-primary)] truncate">
                          {vendor.name}
                        </span>
                        <Chip
                          size="sm"
                          color={vendor.is_active ? "success" : "default"}
                          variant="secondary"
                          className="font-light text-[9px]"
                        >
                          {vendor.is_active ? "ACTIVE" : "INACTIVE"}
                        </Chip>
                      </div>
                      <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] uppercase tracking-widest font-light">
                        {vendor.category} · {vendor.order_count} ORDERS
                      </p>
                    </div>
                    <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                      <button
                        onClick={() => removeSupplier(vendor.id)}
                        className="w-9 h-9 rounded-lg hover:bg-red-50 text-[var(--desk-danger)] flex items-center justify-center transition-colors"
                      >
                        {removingId === vendor.id ? (
                          <Loader2 size={16} className="animate-spin" />
                        ) : (
                          <Trash2 size={16} />
                        )}
                      </button>
                    </div>
                    <ChevronRight
                      size={18}
                      className="text-[var(--desk-text-tertiary)]"
                    />
                  </motion.div>
                ))}
              </div>
            )}
          </AnimatePresence>
        </div>

        {/* Sidebar: Spend Logic */}
        <aside className="w-[360px] shrink-0 hidden lg:flex flex-col gap-6">
          <div className="p-8 bg-[var(--desk-text-primary)] rounded-3xl text-white shadow-2xl relative overflow-hidden">
            <FileText className="absolute top-[-10px] right-[-10px] w-32 h-32 opacity-10 rotate-12" />
            <span className="md-typescale-label-small uppercase tracking-[0.2em] opacity-60 mb-4 block">
              MTD Settlement
            </span>
            <CountUp
              end={totalSpend}
              className="md-typescale-display-small font-light tabular-nums"
            />
            <div className="mt-8 pt-6 border-t border-white/10 flex items-center justify-between">
              <span className="text-xs font-light opacity-60 uppercase tracking-widest">
                Efficiency
              </span>
              <div className="flex items-center gap-2 text-[var(--desk-success)] font-light">
                <ArrowUpRight size={16} />
                <span>+8.2%</span>
              </div>
            </div>
          </div>

          <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
            <h3 className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-6">
              Trade Breakdown
            </h3>
            {topSuppliers.length === 0 ? (
              <div className="rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] p-4 text-center">
                <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] uppercase font-light tracking-widest">
                  No spend breakdown data yet
                </p>
              </div>
            ) : (
              <div className="space-y-4">
                {topSuppliers.slice(0, 5).map((item, i) => (
                  <div key={item.supplier_id} className="flex items-center gap-4">
                    <div className="w-8 h-8 rounded-lg bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[10px] font-black text-[var(--desk-text-tertiary)]">
                      {i + 1}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="md-typescale-body-medium font-light text-[var(--desk-text-primary)] truncate">
                        {item.supplier_name}
                      </p>
                      <p className="text-[10px] text-[var(--desk-text-tertiary)] font-light uppercase">
                        {item.order_count} TRADES
                      </p>
                    </div>
                    <span className="md-typescale-body-small font-light text-[var(--desk-text-primary)]">
                      {item.total.toLocaleString()}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </aside>
      </div>

      {/* Add Vendor Modal */}
      <AnimatePresence>
        {showAddModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="absolute inset-0 bg-[#0a0a0a]/60 backdrop-blur-md"
              onClick={() => setShowAddModal(false)}
            />
            <motion.div
              initial={{ scale: 0.9, opacity: 0, y: 20 }}
              animate={{ scale: 1, opacity: 1, y: 0 }}
              exit={{ scale: 0.9, opacity: 0, y: 20 }}
              className="relative w-full max-w-lg bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-3xl shadow-2xl overflow-hidden"
            >
              <div className="p-6 border-b border-[var(--desk-border)] flex items-center justify-between">
                <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)]">
                  Connect Supply Node
                </h2>
                <button
                  onClick={() => setShowAddModal(false)}
                  className="w-10 h-10 rounded-full hover:bg-[var(--desk-surface-subtle)] flex items-center justify-center transition-colors"
                >
                  <X size={20} />
                </button>
              </div>
              <div className="p-6">
                <div className="relative mb-6">
                  <Search
                    className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)]"
                    size={18}
                  />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => void searchSuppliers(e.target.value)}
                    placeholder="Search network nodes..."
                    className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] transition-all md-typescale-body-medium"
                    autoFocus
                  />
                </div>
                <div className="space-y-2 max-h-[320px] overflow-y-auto pr-1">
                  {searching ? (
                    <div className="py-12 text-center">
                      <Loader2
                        size={24}
                        className="animate-spin mx-auto text-[var(--desk-accent)]"
                      />
                    </div>
                  ) : searchResults.length === 0 ? (
                    <div className="py-12 text-center opacity-40">
                      <Building2 size={48} className="mx-auto mb-4" />
                      <p>No available nodes found</p>
                    </div>
                  ) : (
                    searchResults.map((s) => (
                      <div
                        key={s.id}
                        className="p-4 bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)] rounded-xl flex items-center justify-between"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-lg bg-[var(--desk-surface)] flex items-center justify-center border border-[var(--desk-border)]">
                            {s.name.charAt(0)}
                          </div>
                          <div>
                            <p className="md-typescale-body-medium font-light text-[var(--desk-text-primary)]">
                              {s.name}
                            </p>
                            <p className="text-[10px] text-[var(--desk-text-tertiary)] font-light uppercase">
                              {s.category}
                            </p>
                          </div>
                        </div>
                        <Button
                          variant="primary"
                          size="sm"
                          onPress={() => addSupplier(s.id)}
                          isDisabled={addingId === s.id}
                          className="bg-[var(--desk-text-primary)] text-white font-light rounded-lg h-9 px-4"
                        >
                          {addingId === s.id ? (
                            <>
                              <Loader2 size={14} className="animate-spin mr-1" />
                              Connecting
                            </>
                          ) : (
                            "Connect"
                          )}
                        </Button>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </div>
  );
}
