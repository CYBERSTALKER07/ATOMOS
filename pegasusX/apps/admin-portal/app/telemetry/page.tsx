"use client";

import React, { useEffect, useState, useMemo } from "react";
import { api, type AuditRow } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

type SeverityLevel = "ALL" | "CRITICAL" | "ERROR" | "WARN" | "AUDIT" | "INFO";
type RoleScope = "ALL" | "PLATFORM_ADMIN" | "SUPPLIER" | "WAREHOUSE" | "FLEET_DRIVER" | "DEPOT_CASHIER" | "RETAILER" | "YARD_SECURITY";

interface OperationalEvent {
  id: string;
  timestamp: string;
  severity: "CRITICAL" | "ERROR" | "WARN" | "AUDIT" | "INFO";
  role: string;
  entityType: "SUPPLIER" | "WAREHOUSE" | "RETAILER" | "DRIVER" | "VEHICLE" | "ORDER" | "SYSTEM";
  entityId: string;
  action: string;
  summary: string;
  traceId: string;
  rawPayload: Record<string, unknown>;
}

const INITIAL_EVENTS: OperationalEvent[] = [
  {
    id: "evt_dvir_intercept_01",
    timestamp: new Date(Date.now() - 45000).toISOString(),
    severity: "CRITICAL",
    role: "YARD_SECURITY",
    entityType: "VEHICLE",
    entityId: "veh_isuzu_01",
    action: "ALPR_OUTBOUND_GATE_INTERCEPT",
    summary: "Outbound yard gate barrier locked down on ALPR scan: Vehicle veh_isuzu_01 is grounded under RED_TAG_QUARANTINE (Failed Brakes inspection).",
    traceId: "tr_7f99a812_gate_sec",
    rawPayload: { vehicle_id: "veh_isuzu_01", inspection_id: "insp_88921", status: "RED_TAG_QUARANTINE", defects: ["brakes_service_leak"], gate: "GATE_01_OUT" },
  },
  {
    id: "evt_aml_warn_02",
    timestamp: new Date(Date.now() - 120000).toISOString(),
    severity: "WARN",
    role: "RETAILER",
    entityType: "RETAILER",
    entityId: "ret_baraka_market",
    action: "AML_CEILING_APPROACHING",
    summary: "Retailer TIN 305544332 accumulated 24,200,000 UZS daily cash under uncapped multi-tender policy. Audit logged.",
    traceId: "tr_4e12c981_aml_guard",
    rawPayload: { tin: "305544332", current_exposure_minor: 2420000000, policy: "UNCAPPED_MULTI_TENDER" },
  },
  {
    id: "evt_blind_count_ratchet_03",
    timestamp: new Date(Date.now() - 240000).toISOString(),
    severity: "WARN",
    role: "DEPOT_CASHIER",
    entityType: "WAREHOUSE",
    entityId: "depot_tashkent_central",
    action: "BLIND_RECOUNT_RATCHET_TRIGGERED",
    summary: "Driver drv_tashkent_404 Attempt #1 blind count failed variance tolerance. Mandated blind recount #2 without revealing expected sum.",
    traceId: "tr_991b2234_cashier_ratchet",
    rawPayload: { session_id: "sess_cash_441", driver_id: "drv_tashkent_404", attempt: 1, counted_minor: 119500000, tolerance_minor: 500000 },
  },
  {
    id: "evt_kinetic_action_04",
    timestamp: new Date(Date.now() - 360000).toISOString(),
    severity: "AUDIT",
    role: "WAREHOUSE",
    entityType: "SUPPLIER",
    entityId: "sup_samarkand_dairy",
    action: "ACTION_RELEASE_PICK_WAVE",
    summary: "Pick wave released for Order ord_retailer_9912. Pre-execution rubrics evaluated: PKCS#12 cert valid (120h), 100% SKUs IKPU verified, FX corridor in bounds.",
    traceId: "tr_3389cd11_kinetic_exec",
    rawPayload: { action: "ACTION_RELEASE_PICK_WAVE", cert_hours: 120, ikpu_valid: true, fx_delta_bps: 45, status: "COMMITTED" },
  },
  {
    id: "evt_crushability_guard_05",
    timestamp: new Date(Date.now() - 480000).toISOString(),
    severity: "ERROR",
    role: "WAREHOUSE",
    entityType: "ORDER",
    entityId: "ord_bulk_flour_332",
    action: "GUARD_CRUSHABILITY_VIOLATION",
    summary: "Manifest sealing blocked: Heavy pallet SKU-FLOUR-50KG (1200kg) staged on top of fragile SKU-EGGS-30PK (max crush 150kg). 3D load re-optimization required.",
    traceId: "tr_a8813ef0_crush_guard",
    rawPayload: { layer_above_kg: 1200, layer_below_max_crush_kg: 150, guard: "Guard6_PalletCrushability" },
  },
  {
    id: "evt_soliq_factura_06",
    timestamp: new Date(Date.now() - 600000).toISOString(),
    severity: "INFO",
    role: "PLATFORM_ADMIN",
    entityType: "SUPPLIER",
    entityId: "sup_tashkent_fmcg",
    action: "SOLIQ_E_FACTURA_CONFIRMED",
    summary: "State Tax Committee accepted electronic factura for Order ord_tashkent_8819. Fiscal sign hash verified with PKCS#12 certificate.",
    traceId: "tr_21fbb990_soliq_tax",
    rawPayload: { invoice_id: "inv_991823", soliq_status: "REGISTERED_SOLIQ", vat_collected_minor: 14400000 },
  },
];

export default function OperationalTelemetryPage() {
  const { token } = useAdminToken();

  const [events, setEvents] = useState<OperationalEvent[]>(INITIAL_EVENTS);
  const [severityFilter, setSeverityFilter] = useState<SeverityLevel>("ALL");
  const [roleFilter, setRoleFilter] = useState<RoleScope>("ALL");
  const [searchTerm, setSearchTerm] = useState("");
  const [isLiveStreaming, setIsLiveStreaming] = useState(true);
  const [selectedEvent, setSelectedEvent] = useState<OperationalEvent | null>(null);

  // Ingest Spanner audit records into live operational stream
  useEffect(() => {
    if (!token) return;
    let cancelled = false;

    const pullAudits = async () => {
      try {
        const res = await api.listAudit(token, 20);
        if (cancelled || !res.audit) return;

        const auditEvents: OperationalEvent[] = res.audit.map((a: AuditRow) => ({
          id: `audit_${a.AuditID}`,
          timestamp: a.CreatedAt,
          severity: "AUDIT",
          role: "PLATFORM_ADMIN",
          entityType: (a.TenantType as any) || "SYSTEM",
          entityId: a.TenantID || "GLOBAL",
          action: a.Action,
          summary: `Audit mutation by ${a.ActorSubject}: ${a.Action} on ${a.TenantType}:${a.TenantID || "*"}`,
          traceId: `tr_${a.AuditID.substring(0, 8)}`,
          rawPayload: (() => {
            try {
              return JSON.parse(a.DetailJSON || "{}");
            } catch {
              return { detail: a.DetailJSON };
            }
          })(),
        }));

        setEvents((prev) => {
          const ids = new Set(prev.map((e) => e.id));
          const newUnique = auditEvents.filter((e) => !ids.has(e.id));
          return [...newUnique, ...prev].slice(0, 100);
        });
      } catch {
        // Non-fatal
      }
    };

    void pullAudits();
    return () => {
      cancelled = true;
    };
  }, [token]);

  const filteredEvents = useMemo(() => {
    return events.filter((ev) => {
      if (severityFilter !== "ALL" && ev.severity !== severityFilter) return false;
      if (roleFilter !== "ALL" && ev.role !== roleFilter) return false;
      if (searchTerm.trim()) {
        const q = searchTerm.toLowerCase();
        return (
          ev.action.toLowerCase().includes(q) ||
          ev.summary.toLowerCase().includes(q) ||
          ev.entityId.toLowerCase().includes(q) ||
          ev.traceId.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [events, severityFilter, roleFilter, searchTerm]);

  const exportDiagnostics = () => {
    const jsonStr = JSON.stringify(filteredEvents, null, 2);
    const blob = new Blob([jsonStr], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `pegasusx_diagnostic_bundle_${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Unified Operational Event Stream &amp; Audit Radar
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Real-time multi-tier event telemetry across all 9 roles, pre-execution kinetic guards, and Spanner audit logs.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setIsLiveStreaming(!isLiveStreaming)}
            className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-bold shadow-sm transition-colors ${
              isLiveStreaming
                ? "bg-emerald-600 text-white hover:bg-emerald-700"
                : "bg-gray-200 text-gray-700 hover:bg-gray-300"
            }`}
          >
            <span className={`h-2 w-2 rounded-full ${isLiveStreaming ? "bg-white animate-pulse" : "bg-gray-500"}`} />
            {isLiveStreaming ? "Live Streaming Active" : "Stream Paused"}
          </button>

          <button
            type="button"
            onClick={exportDiagnostics}
            className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 shadow-sm hover:bg-gray-50"
          >
            Export Diagnostic Bundle
          </button>
        </div>
      </div>

      {/* Filter Matrix */}
      <div className="grid grid-cols-1 md:grid-cols-12 gap-3 rounded-xl border bg-white p-4 shadow-sm">
        {/* Search Input */}
        <div className="md:col-span-5">
          <input
            type="text"
            placeholder="Search action, summary, entity ID (e.g. veh_isuzu_01), or trace ID..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-indigo-500"
          />
        </div>

        {/* Severity Filter */}
        <div className="md:col-span-3">
          <select
            value={severityFilter}
            onChange={(e) => setSeverityFilter(e.target.value as SeverityLevel)}
            className="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-semibold"
          >
            <option value="ALL">All Severities</option>
            <option value="CRITICAL">CRITICAL (Red-Tag / Intercept)</option>
            <option value="ERROR">ERROR (Precondition Violations)</option>
            <option value="WARN">WARN (AML / Tolerance Warnings)</option>
            <option value="AUDIT">AUDIT (Mutations &amp; Actions)</option>
            <option value="INFO">INFO (Normal Lifecycle Events)</option>
          </select>
        </div>

        {/* Role Filter */}
        <div className="md:col-span-4">
          <select
            value={roleFilter}
            onChange={(e) => setRoleFilter(e.target.value as RoleScope)}
            className="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-semibold"
          >
            <option value="ALL">All Ecosystem Roles</option>
            <option value="PLATFORM_ADMIN">Role 1: Platform Admin</option>
            <option value="SUPPLIER">Role 2: Supplier Enterprise</option>
            <option value="WAREHOUSE">Role 3: Warehouse / Depot</option>
            <option value="FLEET_DRIVER">Role 4: Fleet Driver</option>
            <option value="DEPOT_CASHIER">Role 7: Depot Cashier</option>
            <option value="RETAILER">Role 8: Retailer Shop</option>
            <option value="YARD_SECURITY">Role 9: Yard &amp; Gate Security</option>
          </select>
        </div>
      </div>

      {/* Live Operational Events Table */}
      <div className="overflow-hidden rounded-xl border bg-white shadow-sm">
        <div className="border-b bg-gray-50 px-4 py-2.5 flex items-center justify-between text-xs text-gray-500 font-bold uppercase tracking-wider">
          <span>Operational Feed ({filteredEvents.length} events)</span>
          <span>Buffer: Last 100 Transactions</span>
        </div>

        <div className="divide-y divide-gray-100 max-h-[600px] overflow-y-auto">
          {filteredEvents.length === 0 ? (
            <div className="p-8 text-center text-xs text-gray-400">
              No operational events matched your filter criteria.
            </div>
          ) : (
            filteredEvents.map((ev) => (
              <div
                key={ev.id}
                onClick={() => setSelectedEvent(ev)}
                className="p-4 hover:bg-gray-50/80 transition-colors cursor-pointer space-y-2"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    {/* Severity Pill */}
                    <span
                      className={`inline-flex items-center rounded px-2 py-0.5 text-[10px] font-extrabold ${
                        ev.severity === "CRITICAL"
                          ? "bg-rose-100 text-rose-800"
                          : ev.severity === "ERROR"
                          ? "bg-red-100 text-red-800"
                          : ev.severity === "WARN"
                          ? "bg-amber-100 text-amber-800"
                          : ev.severity === "AUDIT"
                          ? "bg-indigo-100 text-indigo-800"
                          : "bg-blue-100 text-blue-800"
                      }`}
                    >
                      {ev.severity}
                    </span>

                    {/* Role Pill */}
                    <span className="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-bold text-gray-700">
                      {ev.role}
                    </span>

                    {/* Action Name */}
                    <span className="font-mono font-bold text-xs text-gray-900">
                      {ev.action}
                    </span>
                  </div>

                  <div className="flex items-center gap-3 text-xs text-gray-400">
                    <span className="font-mono text-[11px]">{ev.traceId}</span>
                    <span>{new Date(ev.timestamp).toLocaleTimeString()}</span>
                  </div>
                </div>

                {/* Summary */}
                <p className="text-xs text-gray-700">{ev.summary}</p>

                {/* Entity & Context Line */}
                <div className="flex items-center justify-between text-[11px] text-gray-500 pt-1 border-t border-gray-50">
                  <div className="flex items-center gap-2">
                    <span className="uppercase font-semibold text-gray-400">{ev.entityType}:</span>
                    <CopyableId id={ev.entityId} label={ev.entityType} />
                  </div>

                  <span className="text-indigo-600 font-semibold hover:underline">
                    Inspect Trace &amp; Payload →
                  </span>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Slide-Over Event Drawer */}
      {selectedEvent && (
        <div className="fixed inset-0 z-50 flex justify-end bg-black/40 backdrop-blur-xs">
          <div className="w-full max-w-lg bg-white p-6 shadow-2xl flex flex-col justify-between overflow-y-auto">
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b pb-3">
                <div className="flex items-center gap-2">
                  <span
                    className={`rounded px-2 py-0.5 text-[10px] font-bold ${
                      selectedEvent.severity === "CRITICAL"
                        ? "bg-rose-100 text-rose-800"
                        : selectedEvent.severity === "ERROR"
                        ? "bg-red-100 text-red-800"
                        : selectedEvent.severity === "WARN"
                        ? "bg-amber-100 text-amber-800"
                        : "bg-indigo-100 text-indigo-800"
                    }`}
                  >
                    {selectedEvent.severity}
                  </span>
                  <h3 className="text-sm font-bold text-gray-900 font-mono">
                    {selectedEvent.action}
                  </h3>
                </div>

                <button
                  type="button"
                  onClick={() => setSelectedEvent(null)}
                  className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
                >
                  ✕
                </button>
              </div>

              <div className="space-y-2 text-xs">
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Trace ID:</span>
                  <CopyableId id={selectedEvent.traceId} label="Trace ID" />
                </div>
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Timestamp:</span>
                  <span>{new Date(selectedEvent.timestamp).toLocaleString()}</span>
                </div>
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Target Entity:</span>
                  <span className="font-mono font-semibold">{selectedEvent.entityType}:{selectedEvent.entityId}</span>
                </div>
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Originating Role:</span>
                  <span className="font-semibold text-indigo-700">{selectedEvent.role}</span>
                </div>
              </div>

              <div>
                <h4 className="text-xs font-bold text-gray-700 uppercase tracking-wider mb-1">
                  Plain-Language Summary
                </h4>
                <p className="rounded-lg border bg-gray-50 p-3 text-xs text-gray-800">
                  {selectedEvent.summary}
                </p>
              </div>

              <div>
                <h4 className="text-xs font-bold text-gray-700 uppercase tracking-wider mb-1">
                  Raw Operational Payload JSON
                </h4>
                <pre className="rounded-lg bg-gray-900 p-3 text-xs text-emerald-400 font-mono overflow-x-auto max-h-96">
                  {JSON.stringify(selectedEvent.rawPayload, null, 2)}
                </pre>
              </div>
            </div>

            <div className="pt-4 border-t">
              <button
                type="button"
                onClick={() => setSelectedEvent(null)}
                className="w-full rounded-lg border border-gray-300 py-2 text-xs font-bold text-gray-700 hover:bg-gray-50"
              >
                Close Drawer
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
