"use client";

import React, { useState } from "react";
import type { CashierSessionView, DepotVaultLedgerEntry } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

const UZS_DENOMINATIONS = [200000, 100000, 50000, 20000, 10000, 5000, 2000, 1000] as const;

export default function CashierVaultPage() {
  const { token } = useAdminToken();

  // Active Session State
  const [activeTab, setActiveTab] = useState<"intake" | "count" | "cit">("intake");
  const [session, setSession] = useState<CashierSessionView | null>(null);

  // Form: Bag Intake
  const [cellId, setCellId] = useState("cell-uz");
  const [supplierId, setSupplierId] = useState("sup_tashkent_fmcg");
  const [depotId, setDepotId] = useState("depot_tashkent_central");
  const [driverId, setDriverId] = useState("drv_tashkent_404");
  const [routeId, setRouteId] = useState("route_daily_center_01");
  const [bagSealBarcode, setBagSealBarcode] = useState(`seal_bag_${Date.now()}`);
  const [expectedCashMinor, setExpectedCashMinor] = useState(125000000); // 1,250,000 UZS in tiyins

  // Form: Blind Count Banknote Multipliers
  const [counts, setCounts] = useState<Record<number, number>>({
    200000: 5,
    100000: 2,
    50000: 1,
    20000: 0,
    10000: 0,
    5000: 0,
    2000: 0,
    1000: 0,
  });
  const [toleranceMinor, setToleranceMinor] = useState(500000); // 5,000 UZS tolerance

  // Form: CIT Handover
  const [citCourierBadge, setCitCourierBadge] = useState("inkass_uz_badge_992");
  const [citSealBarcode, setCitSealBarcode] = useState(`cit_seal_${Date.now()}`);
  const [citAmountUZS, setCitAmountUZS] = useState(5000000);
  const [citReceipt, setCitReceipt] = useState<DepotVaultLedgerEntry | null>(null);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // Calculate live running total
  const totalCountedUZS = UZS_DENOMINATIONS.reduce((acc, denom) => {
    return acc + denom * (counts[denom] || 0);
  }, 0);
  const totalCountedMinor = totalCountedUZS * 100;

  const handleStartSession = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) return;
    setError(null);
    setSuccessMsg(null);
    setLoading(true);

    try {
      const res = await foundryApi.startCashierSession(token, {
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        depot_id: depotId.trim(),
        driver_id: driverId.trim(),
        route_id: routeId.trim(),
        bag_seal_barcode: bagSealBarcode.trim(),
        expected_cash_minor: expectedCashMinor,
      });
      setSession(res);
      setActiveTab("count");
      setSuccessMsg(`Session ${res.session_id} initialized. Proceed to physical blind count.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to initialize cashier intake session");
    } finally {
      setLoading(false);
    }
  };

  const handleCountChange = (denom: number, count: number) => {
    setCounts((prev) => ({
      ...prev,
      [denom]: Math.max(0, count),
    }));
  };

  const handleSubmitBlindCount = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token || !session) return;
    setError(null);
    setSuccessMsg(null);
    setLoading(true);

    try {
      const res = await foundryApi.submitBlindCount(token, session.session_id, {
        denominations_uzs: counts,
        tolerance_threshold_minor: toleranceMinor,
      });
      setSession(res);
      if (res.status === "BALANCED") {
        setSuccessMsg(`Session BALANCED! Driver cleared. Vault Clearance Hash: ${res.clearance_token_hash?.substring(0, 16)}...`);
      } else if (res.status === "TOLERANCE_VARIANCE") {
        setSuccessMsg(`Session cleared under TOLERANCE_VARIANCE. Variance recorded in Spanner audit.`);
      }
    } catch (err: any) {
      // Check for structured recount required conflict (UA-1)
      const msg = err instanceof Error ? err.message : "Blind count failed";
      if (msg.includes("recount") || msg.includes("RECOUNT_REQUIRED")) {
        setError(`BLIND RECOUNT REQUIRED (Attempt #2): Counted sum deviated beyond tolerance. Re-tally physical notes.`);
        // Increment attempt number locally
        setSession((prev) => prev ? { ...prev, attempt_number: 2, status: "BLIND_RECOUNT_REQUIRED" } : null);
      } else {
        setError(msg);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleCITHandover = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) return;
    setError(null);
    setSuccessMsg(null);
    setLoading(true);

    try {
      const res = await foundryApi.citVaultHandover(token, {
        depot_id: depotId.trim(),
        supplier_id: supplierId.trim(),
        session_ids: session ? [session.session_id] : ["session_manual_drop"],
        courier_badge_id: citCourierBadge.trim(),
        cit_seal_barcode: citSealBarcode.trim(),
        amount_minor: citAmountUZS * 100,
      });
      setCitReceipt(res);
      setSuccessMsg(`CIT Armored Car Outflow booked to Spanner Vault Ledger! Entry ID: ${res.vault_entry_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "CIT handover submission failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Depot Cashier & Vault Turn-In Oversight
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Role 7 Zero-Trust Dual-Blind Counting Ritual (Guard #11) with UZS Denomination Tally & CIT Handover.
          </p>
        </div>

        {/* Tab Switcher */}
        <div className="flex rounded-lg border bg-gray-100 p-0.5 text-xs font-semibold">
          <button
            type="button"
            onClick={() => setActiveTab("intake")}
            className={`rounded-md px-3 py-1 ${
              activeTab === "intake" ? "bg-white text-gray-900 shadow-sm" : "text-gray-600 hover:text-gray-900"
            }`}
          >
            1. Driver Intake
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("count")}
            className={`rounded-md px-3 py-1 ${
              activeTab === "count" ? "bg-white text-gray-900 shadow-sm" : "text-gray-600 hover:text-gray-900"
            }`}
          >
            2. Blind Denomination Count
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("cit")}
            className={`rounded-md px-3 py-1 ${
              activeTab === "cit" ? "bg-white text-gray-900 shadow-sm" : "text-gray-600 hover:text-gray-900"
            }`}
          >
            3. CIT Vault Handover
          </button>
        </div>
      </div>

      {/* Notifications */}
      {error && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800 flex items-start gap-2">
          <span className="text-base text-rose-600 font-bold">⚠</span>
          <div>
            <strong>Discrepancy / Action Required:</strong> {error}
          </div>
        </div>
      )}
      {successMsg && (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-xs text-emerald-800 flex items-start gap-2">
          <span className="text-base text-emerald-600 font-bold">✓</span>
          <div>{successMsg}</div>
        </div>
      )}

      {/* Active Session Status Card */}
      {session && (
        <div className="rounded-xl border bg-white p-4 shadow-sm flex flex-wrap items-center justify-between gap-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <span className="text-xs font-bold text-gray-500 uppercase tracking-wider">Active Session</span>
              <CopyableId id={session.session_id} label="Session" />
            </div>
            <div className="text-xs text-gray-700">
              Driver: <span className="font-mono font-semibold">{session.driver_id}</span> | Route: <span className="font-mono">{session.route_id}</span> | Seal: <span className="font-mono">{session.bag_seal_barcode}</span>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <div className="text-right">
              <div className="text-[10px] uppercase font-bold text-gray-500">Attempt</div>
              <div className="text-sm font-bold text-gray-900">#{session.attempt_number} of 2</div>
            </div>
            <div className="text-right">
              <div className="text-[10px] uppercase font-bold text-gray-500">Status</div>
              <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-bold ${
                session.status === "BALANCED"
                  ? "bg-emerald-100 text-emerald-800"
                  : session.status === "BLIND_RECOUNT_REQUIRED"
                  ? "bg-amber-100 text-amber-800"
                  : session.status === "DISCREPANT_QUARANTINE"
                  ? "bg-rose-100 text-rose-800"
                  : "bg-blue-100 text-blue-800"
              }`}>
                {session.status}
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Tab 1: Driver Bag Intake */}
      {activeTab === "intake" && (
        <div className="max-w-xl mx-auto rounded-xl border bg-white p-6 shadow-sm space-y-4">
          <h2 className="text-sm font-bold text-gray-900 border-b pb-2">
            Driver Cash Bag Check-In (Stage 1)
          </h2>
          <form onSubmit={handleStartSession} className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Depot ID</label>
                <input
                  type="text"
                  value={depotId}
                  onChange={(e) => setDepotId(e.target.value)}
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

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Driver ID</label>
                <input
                  type="text"
                  value={driverId}
                  onChange={(e) => setDriverId(e.target.value)}
                  required
                  className="w-full rounded border px-3 py-1.5 text-xs font-mono"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Route ID</label>
                <input
                  type="text"
                  value={routeId}
                  onChange={(e) => setRouteId(e.target.value)}
                  required
                  className="w-full rounded border px-3 py-1.5 text-xs font-mono"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">Tamper-Evident Bag Seal Barcode</label>
              <input
                type="text"
                value={bagSealBarcode}
                onChange={(e) => setBagSealBarcode(e.target.value)}
                required
                className="w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
              <p className="text-[11px] text-gray-500 mt-0.5">Scanned barcode from tamper-evident cash drop pouch.</p>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Route Expected Cash (Server-Enforced Tiyins)
              </label>
              <input
                type="number"
                value={expectedCashMinor}
                onChange={(e) => setExpectedCashMinor(Number(e.target.value))}
                required
                className="w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
              <p className="text-[11px] text-amber-600 mt-0.5">
                Note: In Zero-Trust blind mode, this figure is stored in Spanner and stripped from cashier terminal DTO payloads.
              </p>
            </div>

            <button
              type="submit"
              disabled={loading || !token}
              className="w-full rounded-lg bg-indigo-600 py-2.5 text-xs font-bold text-white shadow hover:bg-indigo-700 disabled:opacity-50"
            >
              {loading ? "Initializing..." : "Start Blind Count Session"}
            </button>
          </form>
        </div>
      )}

      {/* Tab 2: Blind Denomination Count */}
      {activeTab === "count" && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Denomination Counter Helper */}
          <div className="lg:col-span-8 rounded-xl border bg-white p-6 shadow-sm space-y-4">
            <div className="flex items-center justify-between border-b pb-3">
              <div>
                <h2 className="text-sm font-bold text-gray-900">
                  Banknote Denomination Counting Table
                </h2>
                <p className="text-xs text-gray-500">
                  Input physical note counts. The system computes running totals automatically.
                </p>
              </div>

              <div className="text-right">
                <span className="text-[10px] font-bold text-gray-500 uppercase">Running Som Total</span>
                <div className="text-lg font-extrabold text-indigo-700">
                  {totalCountedUZS.toLocaleString()} UZS
                </div>
              </div>
            </div>

            <form onSubmit={handleSubmitBlindCount} className="space-y-4">
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                {UZS_DENOMINATIONS.map((denom) => {
                  const count = counts[denom] || 0;
                  const subtotal = denom * count;
                  return (
                    <div key={denom} className="rounded-lg border bg-gray-50 p-3 space-y-1">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-bold text-gray-700">
                          {denom.toLocaleString()} UZS
                        </span>
                        <span className="text-[10px] font-mono text-gray-500">Note</span>
                      </div>
                      <input
                        type="number"
                        min="0"
                        value={count}
                        onChange={(e) => handleCountChange(denom, Number(e.target.value))}
                        className="w-full rounded border bg-white px-2 py-1 text-sm font-bold text-gray-900 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                      <div className="text-[11px] font-semibold text-gray-500 text-right">
                        = {subtotal.toLocaleString()} UZS
                      </div>
                    </div>
                  );
                })}
              </div>

              <div className="border-t pt-3 flex items-center justify-between">
                <div>
                  <label className="block text-[11px] font-medium text-gray-700">
                    Variance Tolerance Threshold (Tiyins)
                  </label>
                  <input
                    type="number"
                    value={toleranceMinor}
                    onChange={(e) => setToleranceMinor(Number(e.target.value))}
                    className="w-48 rounded border px-2.5 py-1 text-xs font-mono mt-1"
                  />
                  <span className="text-[11px] text-gray-500 ml-2">({(toleranceMinor / 100).toLocaleString()} UZS)</span>
                </div>

                <button
                  type="submit"
                  disabled={loading || !token || !session}
                  className="rounded-lg bg-emerald-600 px-6 py-2.5 text-xs font-bold text-white shadow hover:bg-emerald-700 disabled:opacity-50"
                >
                  {loading ? "Verifying..." : "Submit Blind Count"}
                </button>
              </div>
            </form>
          </div>

          {/* Blind Count Safeguards Explanation Panel */}
          <div className="lg:col-span-4 rounded-xl border bg-gray-50 p-5 space-y-3">
            <h3 className="text-xs font-bold text-gray-900 uppercase tracking-wider">
              Zero-Trust Blind Count Protocol
            </h3>
            <ul className="text-xs text-gray-600 space-y-2 list-disc pl-4">
              <li>
                <strong>Attempt #1:</strong> If physical count matches expected cash within tolerance, session immediately resolves to <span className="font-semibold text-emerald-700">BALANCED</span>.
              </li>
              <li>
                <strong>Attempt #2 Ratchet:</strong> If variance exceeds tolerance on Attempt #1, a blind recount is mandated without showing the expected figure.
              </li>
              <li>
                <strong>Discrepant Quarantine:</strong> If Attempt #2 still deviates, the session is quarantined and flagged for physical safe auditor inspection.
              </li>
              <li>
                <strong>Micro-Skimming Prevention:</strong> Driver history tracks cumulative micro-shortages across multiple routes to deter systematic shrinkage.
              </li>
            </ul>
          </div>
        </div>
      )}

      {/* Tab 3: CIT Armored Car Handover */}
      {activeTab === "cit" && (
        <div className="max-w-xl mx-auto rounded-xl border bg-white p-6 shadow-sm space-y-4">
          <h2 className="text-sm font-bold text-gray-900 border-b pb-2">
            CIT Armored Car Safe Outflow Handover (Stage 3)
          </h2>
          <form onSubmit={handleCITHandover} className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Depot ID</label>
                <input
                  type="text"
                  value={depotId}
                  onChange={(e) => setDepotId(e.target.value)}
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
              <label className="block text-xs font-medium text-gray-700 mb-1">Armored Courier Badge ID</label>
              <input
                type="text"
                value={citCourierBadge}
                onChange={(e) => setCitCourierBadge(e.target.value)}
                required
                className="w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
              <p className="text-[11px] text-gray-500 mt-0.5">Inkassatsia courier photo badge identity number.</p>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">CIT Armored Bag Seal Barcode</label>
              <input
                type="text"
                value={citSealBarcode}
                onChange={(e) => setCitSealBarcode(e.target.value)}
                required
                className="w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">Handover Sum (UZS)</label>
              <input
                type="number"
                value={citAmountUZS}
                onChange={(e) => setCitAmountUZS(Number(e.target.value))}
                required
                className="w-full rounded border px-3 py-1.5 text-xs font-bold text-gray-900"
              />
              <p className="text-[11px] text-gray-500 mt-0.5">= {(citAmountUZS * 100).toLocaleString()} tiyins</p>
            </div>

            <button
              type="submit"
              disabled={loading || !token}
              className="w-full rounded-lg bg-indigo-600 py-2.5 text-xs font-bold text-white shadow hover:bg-indigo-700 disabled:opacity-50"
            >
              {loading ? "Recording CIT Outflow..." : "Confirm & Sign CIT Vault Handover"}
            </button>
          </form>

          {citReceipt && (
            <div className="rounded-lg border bg-gray-50 p-4 space-y-2 mt-4">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold text-gray-700">CIT Handover Receipt</span>
                <CopyableId id={citReceipt.vault_entry_id} label="Vault Entry" />
              </div>
              <div className="text-xs text-gray-600 space-y-1">
                <div>Type: <span className="font-semibold">{citReceipt.entry_type}</span></div>
                <div>Amount: <span className="font-mono font-bold text-rose-600">-{(citReceipt.amount_minor / 100).toLocaleString()} UZS</span></div>
                <div>Safe Running Balance: <span className="font-mono font-bold text-emerald-700">{(citReceipt.running_balance_minor / 100).toLocaleString()} UZS</span></div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
