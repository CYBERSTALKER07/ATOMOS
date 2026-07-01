"use client";

import React, { useCallback, useEffect, useState } from "react";
import { apiFetch } from "@/lib/auth";

interface SupplierOption {
  id: string;
  name: string;
}

interface BaselineRow {
  forecast_date: string;
  warehouse_id: string;
  product_id: string;
  product_name?: string;
  baseline_qty: number;
  confidence: number;
  source: string;
}

interface TenantBaseline {
  supplier_id: string;
  rows: BaselineRow[];
  workspace_snapshot_json?: string;
  generated_at: string;
}

interface MEIOWarehouseNode {
  warehouse_id: string;
  sku_count: number;
  critical_skus: number;
  warning_skus: number;
  total_stock: number;
  avg_days_cover: number;
}

interface TenantMEIO {
  supplier_id: string;
  warehouses_scanned: number;
  skus_analyzed: number;
  insights_generated: number;
  transfer_recommendations: number;
  warehouse_balances: MEIOWarehouseNode[];
  stub_source: string;
  generated_at: string;
}

export default function PlanningFederationPage() {
  const [tenantQuery, setTenantQuery] = useState("");
  const [supplierOptions, setSupplierOptions] = useState<SupplierOption[]>([]);
  const [selectedSupplierId, setSelectedSupplierId] = useState("");
  const [baseline, setBaseline] = useState<TenantBaseline | null>(null);
  const [meio, setMeio] = useState<TenantMEIO | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const q = tenantQuery.trim();
    if (q.length < 2) {
      setSupplierOptions([]);
      return;
    }
    const handle = setTimeout(async () => {
      try {
        const res = await apiFetch(
          `/v1/catalog/suppliers/search?q=${encodeURIComponent(q)}`,
        );
        if (!res.ok) return;
        const data = await res.json();
        const rows = Array.isArray(data) ? data : data?.suppliers ?? [];
        setSupplierOptions(
          rows.map((row: { id?: string; supplier_id?: string; name: string }) => ({
            id: row.id ?? row.supplier_id ?? "",
            name: row.name,
          })).filter((row: SupplierOption) => row.id),
        );
      } catch {
        setSupplierOptions([]);
      }
    }, 300);
    return () => clearTimeout(handle);
  }, [tenantQuery]);

  const loadTenant = useCallback(async (supplierId: string) => {
    const sid = supplierId.trim();
    if (!sid) {
      setError("Select or enter a supplier tenant.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const [baselineRes, meioRes] = await Promise.all([
        apiFetch(`/v1/admin/planning/tenants/${encodeURIComponent(sid)}/baseline`),
        apiFetch(`/v1/admin/planning/tenants/${encodeURIComponent(sid)}/meio`),
      ]);
      if (!baselineRes.ok || !meioRes.ok) {
        throw new Error("Planning federation read failed");
      }
      setBaseline(await baselineRes.json());
      setMeio(await meioRes.json());
      setSelectedSupplierId(sid);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
      setBaseline(null);
      setMeio(null);
    } finally {
      setLoading(false);
    }
  }, []);

  return (
    <div
      className="min-h-full p-6 md:p-10"
      style={{ background: "var(--background)", color: "var(--foreground)" }}
    >
      <header className="mb-8">
        <h1 className="text-2xl font-light tracking-tight mb-2">Planning Federation</h1>
        <p className="opacity-60 text-sm">
          Read-only tenant baseline and MEIO rollup across supplier workspaces.
        </p>
      </header>

      <section
        className="mb-8 p-4 rounded-xl border"
        style={{ borderColor: "var(--border)" }}
      >
        <label className="block text-xs uppercase tracking-widest opacity-50 mb-2">
          Tenant selector
        </label>
        <div className="flex flex-col md:flex-row gap-3">
          <input
            type="text"
            value={tenantQuery}
            onChange={(e) => setTenantQuery(e.target.value)}
            placeholder="Search supplier name…"
            className="flex-1 px-3 py-2 rounded-lg border bg-transparent"
            style={{ borderColor: "var(--border)" }}
          />
          <select
            value={selectedSupplierId}
            onChange={(e) => {
              setSelectedSupplierId(e.target.value);
              if (e.target.value) void loadTenant(e.target.value);
            }}
            className="px-3 py-2 rounded-lg border bg-transparent min-w-[220px]"
            style={{ borderColor: "var(--border)" }}
          >
            <option value="">Select tenant…</option>
            {supplierOptions.map((opt) => (
              <option key={opt.id} value={opt.id}>
                {opt.name} ({opt.id.slice(0, 8)}…)
              </option>
            ))}
          </select>
          <button
            type="button"
            disabled={loading}
            onClick={() => void loadTenant(selectedSupplierId || tenantQuery)}
            className="px-4 py-2 rounded-lg text-sm font-medium"
            style={{ background: "var(--primary)", color: "var(--primary-foreground)" }}
          >
            {loading ? "Loading…" : "Load tenant"}
          </button>
        </div>
        {error && (
          <p className="mt-3 text-sm" style={{ color: "var(--danger)" }}>
            {error}
          </p>
        )}
      </section>

      {baseline && (
        <section className="mb-8">
          <h2 className="text-lg font-medium mb-3">Demand baseline</h2>
          <p className="text-xs opacity-50 mb-3">
            {baseline.rows.length} rows · generated {new Date(baseline.generated_at).toLocaleString()}
          </p>
          {baseline.rows.length === 0 ? (
            <p className="text-sm opacity-60">No baseline rows for this tenant.</p>
          ) : (
            <div className="overflow-x-auto rounded-xl border" style={{ borderColor: "var(--border)" }}>
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left opacity-50 border-b" style={{ borderColor: "var(--border)" }}>
                    <th className="p-3">Date</th>
                    <th className="p-3">SKU</th>
                    <th className="p-3">Warehouse</th>
                    <th className="p-3">Qty</th>
                    <th className="p-3">Confidence</th>
                  </tr>
                </thead>
                <tbody>
                  {baseline.rows.slice(0, 50).map((row) => (
                    <tr key={`${row.forecast_date}-${row.product_id}-${row.warehouse_id}`} className="border-b" style={{ borderColor: "var(--border)" }}>
                      <td className="p-3 font-mono text-xs">{row.forecast_date}</td>
                      <td className="p-3">{row.product_name ?? row.product_id}</td>
                      <td className="p-3 font-mono text-xs">{row.warehouse_id.slice(0, 8)}…</td>
                      <td className="p-3">{row.baseline_qty}</td>
                      <td className="p-3">{(row.confidence * 100).toFixed(0)}%</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}

      {meio && (
        <section>
          <h2 className="text-lg font-medium mb-3">MEIO rollup (stub)</h2>
          <p className="text-xs opacity-50 mb-3">
            Source: {meio.stub_source} · {meio.warehouses_scanned} warehouses ·{" "}
            {meio.skus_analyzed} SKUs analyzed
          </p>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
            {[
              { label: "Insights", value: meio.insights_generated },
              { label: "Transfers", value: meio.transfer_recommendations },
              { label: "Warehouses", value: meio.warehouses_scanned },
              { label: "SKUs", value: meio.skus_analyzed },
            ].map((stat) => (
              <div
                key={stat.label}
                className="p-4 rounded-xl border"
                style={{ borderColor: "var(--border)" }}
              >
                <div className="text-xs uppercase opacity-50">{stat.label}</div>
                <div className="text-2xl font-light mt-1">{stat.value}</div>
              </div>
            ))}
          </div>
          {meio.warehouse_balances.length === 0 ? (
            <p className="text-sm opacity-60">No pending replenishment insights.</p>
          ) : (
            <div className="overflow-x-auto rounded-xl border" style={{ borderColor: "var(--border)" }}>
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left opacity-50 border-b" style={{ borderColor: "var(--border)" }}>
                    <th className="p-3">Warehouse</th>
                    <th className="p-3">SKUs</th>
                    <th className="p-3">Critical</th>
                    <th className="p-3">Warning</th>
                    <th className="p-3">Stock</th>
                    <th className="p-3">Avg cover (d)</th>
                  </tr>
                </thead>
                <tbody>
                  {meio.warehouse_balances.map((node) => (
                    <tr key={node.warehouse_id} className="border-b" style={{ borderColor: "var(--border)" }}>
                      <td className="p-3 font-mono text-xs">{node.warehouse_id.slice(0, 8)}…</td>
                      <td className="p-3">{node.sku_count}</td>
                      <td className="p-3">{node.critical_skus}</td>
                      <td className="p-3">{node.warning_skus}</td>
                      <td className="p-3">{node.total_stock}</td>
                      <td className="p-3">{node.avg_days_cover.toFixed(1)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}
    </div>
  );
}
