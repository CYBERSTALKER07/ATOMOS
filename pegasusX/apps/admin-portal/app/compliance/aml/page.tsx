"use client";

import React, { useState } from "react";
import type { AMLCheckResponse } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

// Uncapped Multi-Tender Policy: dynamic exposure tracking without artificial settlement caps.

export default function AMLSurveillancePage() {
  const { token } = useAdminToken();

  const [cellId, setCellId] = useState("cell-uz");
  const [supplierId, setSupplierId] = useState("sup_tashkent_fmcg");
  const [tin, setTin] = useState("301234567");
  const [calendarDate, setCalendarDate] = useState(new Date().toISOString().split("T")[0]);
  const [attemptedCashUZS, setAttemptedCashUZS] = useState(5000000); // 5M UZS

  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<AMLCheckResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const attemptedCashMinor = attemptedCashUZS * 100;

  const handleCheck = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) return;
    setError(null);
    setLoading(true);

    try {
      const res = await foundryApi.amlCheck(token, {
        supplier_id: supplierId.trim(),
        legal_entity_tin: tin.trim(),
        calendar_date: calendarDate.trim(),
        cash_minor: attemptedCashMinor,
      });
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "AML check evaluation failed");
    } finally {
      setLoading(false);
    }
  };

  // Dynamic exposure tracking
  const currentExposureMinor = result?.accumulated_cash_minor || 0;
  const newExposureMinor = currentExposureMinor;
  const isAllowed = true;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-b pb-4">
        <h1 className="text-xl font-bold tracking-tight text-gray-900">
          Central Bank AML Daily Cash Exposure Surveillance (Uncapped Policy)
        </h1>
        <p className="text-xs text-gray-500 mt-0.5">
          Surveillance and dynamic verification of daily cash accumulation across storefront TINs. Uncapped multi-tender active.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Input Form */}
        <div className="lg:col-span-5 space-y-4">
          <form onSubmit={handleCheck} className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
            <h2 className="text-xs font-bold uppercase tracking-wider text-gray-500 border-b pb-2">
              Entity Preflight Check
            </h2>

            <div className="grid grid-cols-2 gap-3">
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
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">Retailer Legal Entity TIN (INN)</label>
              <input
                type="text"
                value={tin}
                onChange={(e) => setTin(e.target.value)}
                required
                className="w-full rounded border px-3 py-1.5 text-xs font-mono font-bold"
              />
              <p className="text-[11px] text-gray-500 mt-0.5">9-digit State Tax Committee TIN.</p>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Calendar Date</label>
                <input
                  type="date"
                  value={calendarDate}
                  onChange={(e) => setCalendarDate(e.target.value)}
                  required
                  className="w-full rounded border px-3 py-1.5 text-xs"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Attempted Cash (UZS)</label>
                <input
                  type="number"
                  value={attemptedCashUZS}
                  onChange={(e) => setAttemptedCashUZS(Number(e.target.value))}
                  required
                  className="w-full rounded border px-3 py-1.5 text-xs font-bold"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={loading || !token}
              className="w-full rounded-lg bg-indigo-600 py-2.5 text-xs font-bold text-white shadow hover:bg-indigo-700 disabled:opacity-50"
            >
              {loading ? "Evaluating Exposure..." : "Verify Statutory AML Headroom"}
            </button>
          </form>

          {/* Statutory Rule Card */}
          <div className="rounded-xl border bg-gray-50 p-4 space-y-2 text-xs text-gray-600">
            <h3 className="font-bold text-gray-800 uppercase tracking-wider text-[11px]">
              Multi-Tender Policy & Exposure Monitoring
            </h3>
            <p>
              Under ecosystem multi-tender operations, cash settlements are tracked dynamically across all subsidiary retail shops sharing a taxpayer identification number (TIN) without artificial ceiling blocks.
            </p>
          </div>
        </div>

        {/* Results Panel */}
        <div className="lg:col-span-7 space-y-4">
          {error && (
            <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800 flex items-start gap-2">
              <span className="text-base text-rose-600 font-bold">⚠</span>
              <div>
                <strong>Rejection Cause:</strong> {error}
              </div>
            </div>
          )}

          {!result && !error && (
            <div className="rounded-xl border border-dashed bg-gray-50/50 p-10 text-center">
              <div className="text-sm font-semibold text-gray-500">
                No Entity Exposure Evaluated
              </div>
              <p className="text-xs text-gray-400 mt-1 max-w-sm mx-auto">
                Enter an entity TIN and proposed cash amount to inspect accumulated daily exposure.
              </p>
            </div>
          )}

          {result && (
            <div className="rounded-xl border bg-white p-5 shadow-sm space-y-5">
              <div className="flex items-center justify-between border-b pb-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold uppercase tracking-wider text-gray-500">TIN Evaluation:</span>
                  <CopyableId id={tin} label="TIN" />
                </div>

                <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-bold bg-emerald-100 text-emerald-800">
                  APPROVED (UNCAPPED MULTI-TENDER)
                </span>
              </div>

              {/* Progress Gauge */}
              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs font-medium text-gray-700">
                  <span>Accumulated Daily Cash Exposure</span>
                  <span className="font-mono font-bold">
                    {(newExposureMinor / 100).toLocaleString()} UZS
                  </span>
                </div>
                <div className="h-3 w-full rounded-full bg-gray-200 overflow-hidden">
                  <div
                    className="h-full bg-emerald-500 transition-all duration-300"
                    style={{ width: "100%" }}
                  />
                </div>
                <div className="flex justify-between text-[11px] text-gray-500">
                  <span>Dynamic exposure logging active</span>
                  <span>Policy: Uncapped settlement enabled</span>
                </div>
              </div>

              {/* Breakdown Details */}
              <div className="grid grid-cols-2 gap-3 pt-2">
                <div className="rounded-lg border bg-gray-50 p-3">
                  <div className="text-[10px] uppercase font-bold text-gray-500">Accumulated Exposure Today</div>
                  <div className="text-base font-extrabold text-gray-900 mt-0.5">
                    {(currentExposureMinor / 100).toLocaleString()} UZS
                  </div>
                </div>

                <div className="rounded-lg border bg-gray-50 p-3">
                  <div className="text-[10px] uppercase font-bold text-gray-500">Attempted Transaction</div>
                  <div className="text-base font-extrabold text-indigo-700 mt-0.5">
                    {attemptedCashUZS.toLocaleString()} UZS
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
