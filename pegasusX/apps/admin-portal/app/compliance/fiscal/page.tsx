"use client";

import React, { useState } from "react";
import type { FiscalPreflightResponse } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

interface LineItemRow {
  sku: string;
  ikpu_code: string;
  package_code: string;
}

export default function FiscalPreflightPage() {
  const { token } = useAdminToken();

  const [cellId, setCellId] = useState("cell-uz");
  const [supplierId, setSupplierId] = useState("sup_tashkent_fmcg");
  const [invoiceId, setInvoiceId] = useState(`inv_${Date.now()}`);
  const [certHours, setCertHours] = useState(120);

  // Line items for fiscal validation
  const [lines, setLines] = useState<LineItemRow[]>([
    { sku: "SKU-MILK-1L", ikpu_code: "04012011001000000", package_code: "PKG_TETRA" },
    { sku: "SKU-BUTTER-200G", ikpu_code: "04051011001000000", package_code: "PKG_WRAP" },
  ]);

  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<FiscalPreflightResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const addLine = () => {
    setLines((prev) => [
      ...prev,
      { sku: `SKU-ITEM-${prev.length + 1}`, ikpu_code: "04012011001000000", package_code: "PKG_CARTON" },
    ]);
  };

  const removeLine = (idx: number) => {
    setLines((prev) => prev.filter((_, i) => i !== idx));
  };

  const updateLine = (idx: number, field: keyof LineItemRow, val: string) => {
    setLines((prev) =>
      prev.map((l, i) => (i === idx ? { ...l, [field]: val } : l))
    );
  };

  const handlePreflight = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) return;
    setError(null);
    setLoading(true);

    try {
      // Calculate future expiry based on certHours
      const expiresAt = new Date(Date.now() + certHours * 3600 * 1000).toISOString();
      const res = await foundryApi.fiscalPreflight(token, {
        cert_expires_at: expiresAt,
        lines,
      });
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Fiscal preflight evaluation failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-b pb-4">
        <h1 className="text-xl font-bold tracking-tight text-gray-900">
          State Tax Committee (Soliq) Fiscal Preflight & PKCS#12 Monitor
        </h1>
        <p className="text-xs text-gray-500 mt-0.5">
          Edge-Case Guard #3: Automated verification of 17-digit IKPU/MXIK tax codes and PKCS#12 e-signature certificate validity window.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Form Panel */}
        <div className="lg:col-span-7 space-y-4">
          <form onSubmit={handlePreflight} className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
            <h2 className="text-xs font-bold uppercase tracking-wider text-gray-500 border-b pb-2">
              Invoice Fiscal Parameters
            </h2>

            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Cell ID</label>
                <input
                  type="text"
                  value={cellId}
                  onChange={(e) => setCellId(e.target.value)}
                  required
                  className="w-full rounded border px-3 py-1.5 text-xs font-mono"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Supplier ID</label>
                <input
                  type="text"
                  value={supplierId}
                  onChange={(e) => setSupplierId(e.target.value)}
                  required
                  className="w-full rounded border px-3 py-1.5 text-xs font-mono"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Invoice ID</label>
                <input
                  type="text"
                  value={invoiceId}
                  onChange={(e) => setInvoiceId(e.target.value)}
                  required
                  className="w-full rounded border px-3 py-1.5 text-xs font-mono"
                />
              </div>
            </div>

            {/* PKCS#12 Validity Hours */}
            <div className="rounded-lg border bg-gray-50 p-4 space-y-2">
              <div className="flex items-center justify-between">
                <label className="text-xs font-bold text-gray-800">
                  PKCS#12 Certificate Remaining Validity
                </label>
                <span
                  className={`text-xs font-bold ${
                    certHours >= 72
                      ? "text-emerald-700"
                      : certHours >= 24
                      ? "text-amber-700"
                      : "text-rose-700"
                  }`}
                >
                  {certHours >= 72
                    ? "✓ Healthy (> 72h window)"
                    : certHours >= 24
                    ? "⚠ Warning: Expiring Soon"
                    : "⛔ CRITICAL: Preflight Blocked (< 24h)"}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <input
                  type="number"
                  min="0"
                  max="8760"
                  value={certHours}
                  onChange={(e) => setCertHours(Number(e.target.value))}
                  className="w-32 rounded border bg-white px-3 py-1.5 text-xs font-bold font-mono focus:outline-none"
                />
                <span className="text-xs text-gray-500">hours remaining until certificate expiry</span>
              </div>
            </div>

            {/* Line Items Table */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-xs font-bold text-gray-800">
                  Invoice Line Items (IKPU Classification)
                </label>
                <button
                  type="button"
                  onClick={addLine}
                  className="rounded bg-indigo-50 px-2 py-1 text-xs font-semibold text-indigo-700 hover:bg-indigo-100"
                >
                  + Add SKU Line
                </button>
              </div>

              <div className="space-y-2">
                {lines.map((l, idx) => (
                  <div key={idx} className="flex items-center gap-2 rounded-lg border bg-gray-50 p-2 text-xs">
                    <input
                      type="text"
                      placeholder="SKU"
                      value={l.sku}
                      onChange={(e) => updateLine(idx, "sku", e.target.value)}
                      className="w-1/3 rounded border bg-white px-2 py-1 font-mono"
                    />
                    <input
                      type="text"
                      placeholder="17-digit IKPU Code"
                      value={l.ikpu_code}
                      onChange={(e) => updateLine(idx, "ikpu_code", e.target.value)}
                      className="w-1/2 rounded border bg-white px-2 py-1 font-mono"
                    />
                    <input
                      type="text"
                      placeholder="Package Code"
                      value={l.package_code}
                      onChange={(e) => updateLine(idx, "package_code", e.target.value)}
                      className="w-1/4 rounded border bg-white px-2 py-1 font-mono"
                    />
                    <button
                      type="button"
                      onClick={() => removeLine(idx)}
                      disabled={lines.length <= 1}
                      className="text-gray-400 hover:text-rose-600 disabled:opacity-30 px-1"
                    >
                      ✕
                    </button>
                  </div>
                ))}
              </div>
            </div>

            <button
              type="submit"
              disabled={loading || !token}
              className="w-full rounded-lg bg-indigo-600 py-2.5 text-xs font-bold text-white shadow hover:bg-indigo-700 disabled:opacity-50"
            >
              {loading ? "Running Soliq Preflight..." : "Execute Soliq Fiscal Preflight"}
            </button>
          </form>
        </div>

        {/* Results Panel */}
        <div className="lg:col-span-5 space-y-4">
          {error && (
            <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800 flex items-start gap-2">
              <span className="text-base text-rose-600 font-bold">⚠</span>
              <div>
                <strong>Preflight Failed:</strong> {error}
              </div>
            </div>
          )}

          {!result && !error && (
            <div className="rounded-xl border border-dashed bg-gray-50/50 p-10 text-center">
              <div className="text-sm font-semibold text-gray-500">
                Awaiting Fiscal Preflight
              </div>
              <p className="text-xs text-gray-400 mt-1 max-w-sm mx-auto">
                Validate IKPU codes and digital certificate health to ensure zero post-dispatch invoice rejection at the State Tax Committee portal.
              </p>
            </div>
          )}

          {result && (
            <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
              <div className="flex items-center justify-between border-b pb-3">
                <span className="text-xs font-bold uppercase tracking-wider text-gray-500">
                  Preflight Result
                </span>
                <span
                  className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-bold ${
                    result.fiscal_preflight_ok
                      ? "bg-emerald-100 text-emerald-800"
                      : "bg-rose-100 text-rose-800"
                  }`}
                >
                  {result.fiscal_preflight_ok ? "READY FOR SOLIQ REGISTRATION" : "PREFLIGHT BLOCKED"}
                </span>
              </div>

              <div className="space-y-2 text-xs">
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Invoice:</span>
                  <CopyableId id={invoiceId} label="Invoice" />
                </div>
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Certificate Status:</span>
                  <span className="font-semibold text-emerald-700">Valid ({certHours}h remaining)</span>
                </div>
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Lines Validated:</span>
                  <span className="font-semibold">{lines.length} SKUs</span>
                </div>
              </div>

              {result.fiscal_preflight_ok ? (
                <div className="rounded-lg bg-emerald-50 border border-emerald-200 p-3 text-xs text-emerald-800">
                  ✓ All items satisfy Soliq commodity taxonomy requirements. Ready for e-signature.
                </div>
              ) : (
                <div className="rounded-lg bg-rose-50 border border-rose-200 p-3 text-xs text-rose-800">
                  ⛔ Preflight blocked. Ensure all SKUs have valid 17-digit IKPU codes and certificate has &gt;= 72h validity.
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
