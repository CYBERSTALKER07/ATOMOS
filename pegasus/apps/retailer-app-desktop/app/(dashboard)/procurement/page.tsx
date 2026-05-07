"use client";

import { useMemo, useState, useCallback } from "react";
import {
  Building2, Plus, Clock, ChevronRight, Package,
  HandCoins, ArrowUpRight, TrendingDown, FileText, AlertTriangle,
  X, Search, Loader2, Trash2,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
import EmptyState from "../../../components/EmptyState";
import PageTransition from "../../../components/PageTransition";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import type { Supplier, RetailerAnalytics } from "../../../lib/types";

export default function ProcurementPage() {
  const { data: suppliers, loading: loadingSuppliers, error, mutate } = useLiveData<Supplier[]>("/v1/retailer/suppliers");
  const { data: analytics } = useLiveData<RetailerAnalytics>("/v1/retailer/analytics/expenses");

  const supplierList = suppliers ?? [];
  const totalSpend = analytics?.total_this_month ?? 0;
  const lastMonthSpend = analytics?.total_last_month ?? 0;
  const topSuppliers = analytics?.top_suppliers ?? [];

  const sparkSpend = useMemo(() => {
    const monthly = analytics?.monthly_expenses ?? [];
    if (monthly.length > 0) return monthly.map((m) => m.total);
    return Array.from({ length: 12 }, (_, i) => totalSpend * 0.05 + i * (totalSpend * 0.008));
  }, [analytics, totalSpend]);

  const sparkOrders = useMemo(() => {
    return Array.from({ length: 12 }, (_, i) => 20 + i * 3 + Math.cos(i * 0.6) * 8);
  }, []);

  const totalOrders = supplierList.reduce((s, v) => s + v.order_count, 0);

  const spendDelta = lastMonthSpend > 0
    ? (((totalSpend - lastMonthSpend) / lastMonthSpend) * 100).toFixed(0)
    : null;

  /* ── Supplier management state ── */
  const [showAddModal, setShowAddModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<Supplier[]>([]);
  const [searching, setSearching] = useState(false);
  const [addingId, setAddingId] = useState<string | null>(null);
  const [removingId, setRemovingId] = useState<string | null>(null);

  const searchSuppliers = useCallback(async (q: string) => {
    setSearchQuery(q);
    if (q.length < 2) { setSearchResults([]); return; }
    setSearching(true);
    try {
      const res = await apiFetch(`/v1/catalog/suppliers/search?q=${encodeURIComponent(q)}`);
      if (res.ok) {
        const data = await res.json();
        const existing = new Set(supplierList.map((s) => s.id));
        setSearchResults((data ?? []).filter((s: Supplier) => !existing.has(s.id)));
      }
    } catch { /* swallow */ } finally { setSearching(false); }
  }, [supplierList]);

  const addSupplier = useCallback(async (supplierId: string) => {
    setAddingId(supplierId);
    try {
      const res = await apiFetch(`/v1/retailer/suppliers/${supplierId}/add`, {
        method: "POST",
        headers: { "Idempotency-Key": `retailer-supplier-add:${supplierId}` },
      });
      if (res.ok) {
        mutate();
        setSearchResults((prev) => prev.filter((s) => s.id !== supplierId));
      }
    } catch { /* swallow */ } finally { setAddingId(null); }
  }, [mutate]);

  const removeSupplier = useCallback(async (supplierId: string) => {
    setRemovingId(supplierId);
    try {
      const res = await apiFetch(`/v1/retailer/suppliers/${supplierId}/remove`, {
        method: "POST",
        headers: { "Idempotency-Key": `retailer-supplier-remove:${supplierId}` },
      });
      if (res.ok) { mutate(); }
    } catch { /* swallow */ } finally { setRemovingId(null); }
  }, [mutate]);

  /* ── Loading skeleton ── */
  if (loadingSuppliers) {
    return (
      <div className="min-h-full p-6 md:p-8">
        <Skeleton className="h-8 w-64 rounded-lg mb-2" />
        <Skeleton className="h-4 w-96 rounded-lg mb-8" />
        <div className="grid grid-cols-4 gap-4 mb-8">
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-28 rounded-2xl" />)}
        </div>
        <div className="flex gap-6">
          <div className="flex-1 flex flex-col gap-2">
            {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-20 rounded-2xl" />)}
          </div>
          <Skeleton className="w-[400px] h-80 rounded-2xl" />
        </div>
      </div>
    );
  }

  /* ── Error state ── */
  if (error) {
    return (
      <div className="min-h-full p-6 md:p-8 flex flex-col items-center justify-center gap-4">
        <AlertTriangle size={32} style={{ color: "var(--danger)" }} />
        <p className="md-typescale-title-medium text-foreground">Failed to load suppliers</p>
        <p className="md-typescale-body-medium text-muted">{error.message}</p>
        <Button onPress={() => mutate()} className="md-btn md-btn-outlined">Retry</Button>
      </div>
    );
  }

  /* ── Empty state ── */
  if (supplierList.length === 0) {
    return (
      <PageTransition>
        <EmptyState 
          imageUrl="/images/empty-suppliers.png"
          headline="No suppliers connected"
          body="Add your first supplier to start procuring inventory."
          action="Add Vendor"
          onAction={() => setShowAddModal(true)}
        />
      </PageTransition>
    );
  }

  return (
    <PageTransition className="min-h-full p-6 md:p-8">
      {/* ── Header ── */}
      <header className="mb-6 flex items-end justify-between gap-4 flex-wrap">
        <div>
          <h1 className="md-typescale-headline-large">Procurement & Suppliers</h1>
          <p className="md-typescale-body-medium mt-1" style={{ color: "var(--muted)" }}>
            Manage vendor relationships, track recurring purchases, and monitor payables.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="primary" onPress={() => setShowAddModal(true)} className="md-btn md-btn-filled md-typescale-label-large px-5 h-10 flex items-center gap-2">
            <Plus size={18} /> Add Vendor
          </Button>
        </div>
      </header>

      {/* ── KPI Bento ── */}
      <BentoGrid className="mb-8">
        <BentoCard delay={0}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">This Month Spend</span>
              <HandCoins size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={totalSpend} className="md-kpi-value" suffix="" />
              <MiniSparkline data={sparkSpend} width={72} height={28} />
            </div>
            {spendDelta && (
              <div className="flex items-center gap-1.5">
                <ArrowUpRight size={14} strokeWidth={2} style={{ color: "var(--success)" }} />
                <span className="md-kpi-sub" style={{ color: "var(--success)" }}>{spendDelta}% vs last month</span>
              </div>
            )}
          </div>
        </BentoCard>

        <BentoCard delay={60}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Total Orders</span>
              <Package size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={totalOrders} className="md-kpi-value" />
              <MiniSparkline data={sparkOrders} width={72} height={28} />
            </div>
            <span className="md-kpi-sub">{supplierList.length} active vendors</span>
          </div>
        </BentoCard>

        <BentoCard delay={120}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Last Month</span>
              <AlertTriangle size={18} strokeWidth={1.5} style={{ color: "var(--warning)" }} />
            </div>
            <CountUp end={lastMonthSpend} className="md-kpi-value" suffix="" />
            <span className="md-kpi-sub">Previous period</span>
          </div>
        </BentoCard>

        <BentoCard delay={180}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Top Suppliers</span>
              <TrendingDown size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <CountUp end={topSuppliers.length} className="md-kpi-value" />
            <span className="md-kpi-sub">By spend volume</span>
          </div>
        </BentoCard>
      </BentoGrid>

      {/* ── Split: Vendor List + Ledger Panel ── */}
      <div className="flex gap-6 min-h-[420px]">

        {/* Left: Vendor List */}
        <motion.div 
          initial="hidden"
          animate="visible"
          variants={{
            hidden: { opacity: 0 },
            visible: {
              opacity: 1,
              transition: { staggerChildren: 0.05 }
            }
          }}
          className="flex-1 flex flex-col gap-2 overflow-y-auto max-h-[calc(100dvh-440px)] pr-1"
        >
          <div className="flex items-center justify-between mb-3">
            <h2 className="md-typescale-title-large font-semibold text-foreground">Contracted Suppliers</h2>
            <span className="md-typescale-label-large text-muted">{supplierList.length} vendors</span>
          </div>

          {supplierList.map((vendor) => {
            const topEntry = topSuppliers.find((t) => t.supplier_id === vendor.id);
            return (
              <motion.button 
                variants={{
                  hidden: { opacity: 0, y: 10 },
                  visible: { opacity: 1, y: 0 }
                }}
                key={vendor.id} 
                className="bento-card w-full text-left cursor-pointer hover-lift active-press transition-all duration-150"
              >
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 rounded-xl flex items-center justify-center shrink-0" style={{ background: "var(--surface)" }}>
                    {vendor.logo_url ? (
                      <img src={vendor.logo_url} alt={vendor.name} className="w-full h-full rounded-xl object-cover" />
                    ) : (
                      <Building2 size={20} style={{ color: "var(--muted)" }} />
                    )}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2.5">
                      <span className="md-typescale-title-small font-semibold text-foreground truncate">{vendor.name}</span>
                      <Chip size="sm" color={vendor.is_active ? "success" : "default"} variant="soft" className="shrink-0">
                        {vendor.is_active ? "Active" : "Inactive"}
                      </Chip>
                    </div>
                    <div className="flex items-center gap-3 mt-1.5">
                      <span className="md-typescale-body-small text-muted">{vendor.category}</span>
                      <span className="text-muted opacity-40">·</span>
                      <span className="md-typescale-body-small text-muted">{vendor.order_count} orders</span>
                    </div>
                  </div>
                  <div className="text-right shrink-0 mr-2">
                    {topEntry && (
                      <>
                        <span className="md-typescale-label-small uppercase tracking-widest font-semibold block" style={{ color: "var(--muted)" }}>Spend</span>
                        <span className="md-typescale-label-large font-semibold tabular-nums">{topEntry.total.toLocaleString()}</span>
                      </>
                    )}
                  </div>
                  <ChevronRight size={18} style={{ color: "var(--muted)" }} />
                  <button
                    onClick={(e) => { e.stopPropagation(); removeSupplier(vendor.id); }}
                    className="p-1.5 rounded-lg hover:bg-surface cursor-pointer transition-colors ml-1"
                    title="Remove supplier"
                  >
                    {removingId === vendor.id ? (
                      <Loader2 size={16} className="animate-spin text-muted" />
                    ) : (
                      <Trash2 size={16} style={{ color: "var(--danger)" }} />
                    )}
                  </button>
                </div>
              </motion.button>
            );
          })}
        </motion.div>

        {/* Right: Ledger Summary Panel */}
        <div className="w-full lg:w-[360px] xl:w-[400px] shrink-0 hidden lg:flex flex-col gap-4">
          <h2 className="md-typescale-title-large font-semibold text-foreground">Spend Breakdown</h2>

          <div className="bento-card" style={{ background: "var(--accent)", color: "var(--accent-foreground)" }}>
            <div className="md-kpi-card" style={{ minHeight: "auto", padding: 0 }}>
              <span className="md-typescale-label-small uppercase tracking-widest opacity-80 font-semibold">Current Month</span>
              <CountUp end={totalSpend} className="md-typescale-headline-small font-bold tabular-nums" suffix="" />
              <div className="flex justify-between items-center mt-2 opacity-80 md-typescale-body-small font-medium">
                <span>This period</span>
                <span className="font-bold">{supplierList.length} suppliers</span>
              </div>
            </div>
          </div>

          <div className="bento-card">
            <div className="flex flex-col gap-3">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-full flex items-center justify-center" style={{ background: "var(--surface)" }}>
                  <FileText size={16} style={{ color: "var(--muted)" }} />
                </div>
                <div>
                  <span className="md-typescale-title-small font-semibold text-foreground block">Top Suppliers by Spend</span>
                  <span className="md-typescale-body-small text-muted">This month</span>
                </div>
              </div>
              <div className="border-t border-[var(--border)] pt-3 space-y-2.5">
                {topSuppliers.slice(0, 5).map((ts) => (
                  <div key={ts.supplier_id} className="flex justify-between items-center">
                    <div>
                      <span className="md-typescale-body-medium font-medium text-foreground block">{ts.supplier_name}</span>
                      <span className="md-typescale-label-small text-muted">{ts.order_count} orders</span>
                    </div>
                    <span className="md-typescale-label-large font-semibold tabular-nums">{ts.total.toLocaleString()}</span>
                  </div>
                ))}
                {topSuppliers.length === 0 && (
                  <p className="md-typescale-body-small text-muted text-center py-2">No spend data yet</p>
                )}
              </div>
            </div>
          </div>

          {totalSpend > 0 && lastMonthSpend > 0 && (
            <div className="bento-card">
              <div className="flex flex-col gap-2">
                <span className="md-kpi-label">Month-over-Month</span>
                <div className="w-full h-2 rounded-full" style={{ background: "var(--surface)" }}>
                  <div
                    className="h-full rounded-full bg-accent"
                    style={{ width: `${Math.min(100, (totalSpend / lastMonthSpend) * 100)}%` }}
                  />
                </div>
                <div className="flex justify-between md-typescale-label-small text-muted">
                  <span>{((totalSpend / lastMonthSpend) * 100).toFixed(0)}% of last month</span>
                  <span className="font-semibold text-foreground tabular-nums">{totalSpend.toLocaleString()} / {lastMonthSpend.toLocaleString()}</span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ── Add Vendor Modal ── */}
      {showAddModal && (
        <>
          <div className="fixed inset-0 bg-black/50 z-40" onClick={() => setShowAddModal(false)} />
          <div
            className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-50 w-full max-w-lg rounded-2xl border border-[var(--border)] p-6 shadow-2xl flex flex-col gap-4"
            style={{ background: "var(--background)", maxHeight: "80dvh" }}
          >
            <div className="flex items-center justify-between">
              <h2 className="md-typescale-title-large font-semibold text-foreground">Add Supplier</h2>
              <button onClick={() => setShowAddModal(false)} className="p-2 rounded-full hover:bg-surface cursor-pointer">
                <X size={18} style={{ color: "var(--muted)" }} />
              </button>
            </div>

            <div className="relative">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => searchSuppliers(e.target.value)}
                placeholder="Search suppliers by name..."
                className="w-full pl-10 pr-4 py-2.5 rounded-xl border border-[var(--border)] bg-transparent text-foreground md-typescale-body-medium focus:outline-none focus:border-[var(--accent)]"
                autoFocus
              />
            </div>

            <div className="flex-1 overflow-y-auto flex flex-col gap-2 min-h-[200px]">
              {searching && (
                <div className="flex items-center justify-center py-8">
                  <Loader2 size={20} className="animate-spin text-muted" />
                </div>
              )}

              {!searching && searchQuery.length >= 2 && searchResults.length === 0 && (
                <div className="flex flex-col items-center justify-center py-8 gap-2">
                  <Building2 size={28} style={{ color: "var(--muted)" }} />
                  <p className="md-typescale-body-medium text-muted">No suppliers found</p>
                </div>
              )}

              {!searching && searchQuery.length < 2 && (
                <div className="flex flex-col items-center justify-center py-8 gap-2">
                  <Search size={28} style={{ color: "var(--muted)" }} />
                  <p className="md-typescale-body-medium text-muted">Type at least 2 characters to search</p>
                </div>
              )}

              {searchResults.map((s) => (
                <div key={s.id} className="flex items-center gap-3 p-3 rounded-xl border border-[var(--border)] hover:bg-surface transition-colors">
                  <div className="w-10 h-10 rounded-lg flex items-center justify-center shrink-0" style={{ background: "var(--surface)" }}>
                    {s.logo_url ? (
                      <img src={s.logo_url} alt={s.name} className="w-full h-full rounded-lg object-cover" />
                    ) : (
                      <Building2 size={18} style={{ color: "var(--muted)" }} />
                    )}
                  </div>
                  <div className="flex-1 min-w-0">
                    <span className="md-typescale-body-medium font-semibold text-foreground block truncate">{s.name}</span>
                    <span className="md-typescale-label-small text-muted">{s.category}</span>
                  </div>
                  <Button
                    variant="primary"
                    size="sm"
                    onPress={() => addSupplier(s.id)}
                    isDisabled={addingId === s.id}
                    className="md-btn md-btn-filled px-4 h-8 flex items-center gap-1.5"
                  >
                    {addingId === s.id ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}
                    Add
                  </Button>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </PageTransition>
  );
}
