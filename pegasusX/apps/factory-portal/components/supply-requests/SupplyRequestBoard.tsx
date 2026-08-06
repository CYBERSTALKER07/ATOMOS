"use client";

import { usePortalT } from "@/lib/i18n";
import { PageSection } from '@/components/PageSection';
import type { SupplyRequest } from './types';
import { ACTIONS } from './constants';

interface SupplyRequestBoardProps {
  filtered: SupplyRequest[];
  transitioning: string | null;
  handleTransition: (request: SupplyRequest, action: string) => void;
}

export function SupplyRequestBoard({
  filtered,
  transitioning,
  handleTransition,
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
                  <div className="font-medium">{request.warehouse_name || request.warehouse_id.slice(0, 8)}</div>
                  <div className="text-xs opacity-70 mt-1">{request.priority} · {request.item_count ?? request.items?.length ?? 0} items</div>
                  <div className="text-xs font-mono opacity-60 mt-1">
                    {request.requested_delivery_date ? new Date(request.requested_delivery_date).toLocaleDateString() : 'No delivery date'}
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
