import { Fragment, useState } from 'react';
import { motion } from 'framer-motion';
import { PageSection } from '@/components/PageSection';
import type { SupplyRequest } from './types';
import { ACTIONS, PRIORITY_COLORS, STATE_COLORS } from './constants';

interface SupplyRequestListProps {
  pageItems: SupplyRequest[];
  selectedIds: Set<string>;
  transitioning: string | null;
  handleToggleAll: () => void;
  handleToggleOne: (id: string, e: React.ChangeEvent<HTMLInputElement>) => void;
  handleTransition: (request: SupplyRequest, action: string) => void;
}

export function SupplyRequestList({
  pageItems,
  selectedIds,
  transitioning,
  handleToggleAll,
  handleToggleOne,
  handleTransition,
}: SupplyRequestListProps) {
  const [expandedRequestId, setExpandedRequestId] = useState<string | null>(null);

  return (
    <PageSection title="Demand queue" description="Advance requests through ACK → production → ready → fulfill.">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="overflow-x-auto -mx-5 px-5"
      >
        <table className="desk-table w-full text-sm">
          <thead>
            <tr style={{ background: 'var(--color-md-surface-container)' }}>
              <th className="text-left px-4 py-3 font-medium w-10">
                <input 
                  type="checkbox" 
                  className="rounded border-(--color-md-outline) bg-transparent"
                  checked={pageItems.length > 0 && selectedIds.size === pageItems.length}
                  onChange={handleToggleAll}
                />
              </th>
              <th className="text-left px-4 py-3 font-medium">Warehouse</th>
              <th className="text-left px-4 py-3 font-medium">Priority</th>
              <th className="text-left px-4 py-3 font-medium">State</th>
              <th className="text-left px-4 py-3 font-medium">Items</th>
              <th className="text-left px-4 py-3 font-medium">Notes</th>
              <th className="text-left px-4 py-3 font-medium">Volume (VU)</th>
              <th className="text-left px-4 py-3 font-medium">Delivery Date</th>
              <th className="text-left px-4 py-3 font-medium">Created</th>
              <th className="text-right px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {pageItems.map((request, index) => (
              <Fragment key={request.request_id}>
              <motion.tr 
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.05 }}
                className="border-t hover:bg-[var(--default)]/50 transition-colors cursor-pointer" 
                style={{ borderColor: 'var(--color-md-outline-variant)' }}
                onClick={() => setExpandedRequestId(expandedRequestId === request.request_id ? null : request.request_id)}
              >
                <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                  <input 
                    type="checkbox" 
                    className="rounded border-(--color-md-outline) bg-transparent"
                    checked={selectedIds.has(request.request_id)}
                    onChange={(e) => handleToggleOne(request.request_id, e)}
                    onClick={(e) => e.stopPropagation()}
                  />
                </td>
                <td className="px-4 py-3">
                  <div className="font-medium">{request.warehouse_name || request.warehouse_id.slice(0, 8)}</div>
                  <div className="text-xs font-mono" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                    {request.request_id.slice(0, 8)}
                  </div>
                </td>
                <td className="px-4 py-3">
                  <span className="px-2 py-0.5 rounded-full text-[10px] font-light uppercase tracking-wider" style={{ border: `1px solid ${PRIORITY_COLORS[request.priority]}`, color: PRIORITY_COLORS[request.priority] || 'inherit' }}>
                    {request.priority}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className="px-2 py-0.5 rounded text-[10px] font-light uppercase tracking-wider" style={{ background: STATE_COLORS[request.state], color: 'white' }}>
                    {request.state.replace(/_/g, ' ')}
                  </span>
                </td>
                <td className="px-4 py-3 tabular-nums font-mono">
                  {request.item_count ?? request.items?.length ?? 0}
                </td>
                <td className="px-4 py-3 text-xs max-w-[180px] truncate" title={request.notes || undefined}>
                  {request.notes || '—'}
                </td>
                <td className="px-4 py-3 tabular-nums font-mono">{request.total_volume_vu.toLocaleString()}</td>
                <td className="px-4 py-3 tabular-nums font-mono text-xs">
                  {request.requested_delivery_date ? new Date(request.requested_delivery_date).toLocaleDateString() : '—'}
                </td>
                <td className="px-4 py-3 text-xs tabular-nums font-mono" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  {new Date(request.created_at).toLocaleDateString()}
                </td>
                <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                  <div className="flex gap-2 justify-end">
                    {(ACTIONS[request.state] || []).map((action) => (
                      <motion.button
                        whileHover={{ scale: 1.05 }}
                        whileTap={{ scale: 0.95 }}
                        key={action.action}
                        onClick={() => void handleTransition(request, action.action)}
                        disabled={transitioning === request.request_id}
                        className="px-3 py-1 rounded-lg text-xs font-medium transition-opacity disabled:opacity-50 hover-lift active-press"
                        style={{ background: action.color, color: 'white' }}
                      >
                        {transitioning === request.request_id ? '...' : action.label}
                      </motion.button>
                    ))}
                  </div>
                </td>
              </motion.tr>
              {expandedRequestId === request.request_id && (request.items?.length ?? 0) > 0 && (
                <tr key={`${request.request_id}-items`} className="border-t" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                  <td colSpan={10} className="px-4 py-3 bg-[var(--color-md-surface-container-low)]">
                    <div className="text-xs font-light uppercase tracking-wider mb-2" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                      Requested SKUs
                    </div>
                    <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                      {request.items?.map((item) => (
                        <div key={item.item_id} className="rounded-lg border px-3 py-2 text-sm" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                          <div className="font-mono text-xs">{item.product_id}</div>
                          <div className="tabular-nums">Qty {item.requested_quantity.toLocaleString()}</div>
                        </div>
                      ))}
                    </div>
                  </td>
                </tr>
              )}
              </Fragment>
            ))}
          </tbody>
        </table>
      </motion.div>
    </PageSection>
  );
}
