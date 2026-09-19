"use client";

import { usePortalT } from "@/lib/i18n";
import { PageSection } from '@/components/PageSection';
import type { SupplyRequest } from './types';
import { ACTIONS } from './constants';

interface SupplyRequestBoardProps {
  filtered: SupplyRequest[];
  transitioning: string | null;
  handleTransition: (request: SupplyRequest, action: string) => void;
  qcById: Record<string, string>;
  onQC: (request: SupplyRequest, result: "PASS" | "FAIL") => void;
}

export function SupplyRequestBoard({
  filtered,
  transitioning,
  handleTransition,
  qcById,
  onQC,
}: SupplyRequestBoardProps) {
  const t = usePortalT();
  return (
    <PageSection title={t("factory_portal.supply_requests.supply_request_board.text.production_lane_board")} description={t("factory_portal.residual.text.kanban_by_supply_request_lifecycle_state")}>
      <div className="grid gap-4 lg:grid-cols-4 overflow-x-auto">
        {(['SUBMITTED', 'ACKNOWLEDGED', 'IN_PRODUCTION', 'READY'] as const).map((lane) => (
          <div key={lane} className="min-w-[220px] rounded-xl border p-3" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
            <div className="text-xs font-light uppercase tracking-wider mb-3">{lane.replace(/_/g, ' ')}</div>
            <div className="space-y-2">
              {filtered.filter((r) => r.state === lane).map((request) => (
                <div key={request.request_id} className="rounded-lg border p-3 text-sm" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                  <div className="flex items-start justify-between gap-2">
                    <div className="font-medium">{request.warehouse_name || request.warehouse_id.slice(0, 8)}</div>
                    {request.sla_status && request.sla_status !== "N/A" && request.sla_status !== "MET" ? (
                      <span
                        className="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide"
                        style={{
                          background:
                            request.sla_status === "BREACHED"
                              ? "var(--color-md-error-container, #fdecea)"
                              : request.sla_status === "AT_RISK"
                                ? "var(--color-md-tertiary-container, #fff4e5)"
                                : "var(--color-md-secondary-container, #e8f5e9)",
                          color:
                            request.sla_status === "BREACHED"
                              ? "var(--color-md-on-error-container, #b71c1c)"
                              : request.sla_status === "AT_RISK"
                                ? "var(--color-md-on-tertiary-container, #e65100)"
                                : "var(--color-md-on-secondary-container, #1b5e20)",
                        }}
                        title={
                          request.sla_due_at
                            ? `SLA due ${new Date(request.sla_due_at).toLocaleString()}`
                            : "SLA"
                        }
                      >
                        {request.sla_status.replace(/_/g, " ")}
                      </span>
                    ) : null}
                  </div>
                  <div className="text-xs opacity-70 mt-1">
                    {request.priority} · {request.item_count ?? request.items?.length ?? 0} items
                    {qcById[request.request_id] ? ` · QC ${qcById[request.request_id]}` : ""}
                  </div>
                  <div className="text-xs font-mono opacity-60 mt-1">
                    {request.requested_delivery_date ? new Date(request.requested_delivery_date).toLocaleDateString() : 'No delivery date'}
                    {typeof request.sla_hours_remaining === "number"
                      ? ` · ${request.sla_hours_remaining > 0 ? `${request.sla_hours_remaining}h left` : `${Math.abs(request.sla_hours_remaining)}h overdue`}`
                      : null}
                  </div>
                  <div className="flex flex-wrap gap-1 mt-2">
                    {(ACTIONS[request.state] || []).map((action) => (
                      <button
                        key={action.action}
                        type="button"
                        className="px-2 py-1 rounded text-[10px] font-medium text-white"
                        style={{ background: action.color }}
                        disabled={transitioning === request.request_id}
                        onClick={() => void handleTransition(request, action.action)}
                      >
                        {action.label}
                      </button>
                    ))}
                    <button
                      type="button"
                      className="px-2 py-1 rounded text-[10px] font-medium text-white"
                      style={{ background: "var(--color-md-success, #2e7d32)" }}
                      disabled={transitioning === request.request_id}
                      onClick={() => void onQC(request, "PASS")}
                    >
                      PASS
                    </button>
                    <button
                      type="button"
                      className="px-2 py-1 rounded text-[10px] font-medium text-white"
                      style={{ background: "var(--color-md-error, #c62828)" }}
                      disabled={transitioning === request.request_id}
                      onClick={() => void onQC(request, "FAIL")}
                    >
                      FAIL
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </PageSection>
  );
}
