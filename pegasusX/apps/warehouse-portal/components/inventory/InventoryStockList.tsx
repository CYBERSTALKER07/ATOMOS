'use client';

import { motion } from 'framer-motion';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';

interface InventoryItem {
  product_id: string;
  product_name: string;
  quantity: number;
  reorder_threshold: number;
  sku: string;
  sku_id?: string;
  out_of_stock_policy?: string;
  effective_policy?: string;
  accepts_backorder?: boolean;
}

interface InventoryStockListProps {
  items: InventoryItem[];
  loading: boolean;
  search: string;
  lowOnly: boolean;
  adjusting: string | null;
  adjustVal: string;
  pulsedProductIds: string[];
  onAdjustValChange: (val: string) => void;
  onStartAdjust: (item: InventoryItem) => void;
  onReviewAdjust: (item: InventoryItem) => void;
  onCancelAdjust: () => void;
  onPolicyChange: (productId: string, policy: string) => void;
}

/**
 * Inventory stock table with product rows, quantity display,
 * OOS policy pickers, adjust controls, and pulse sync animations.
 */
export default function InventoryStockList({
  items,
  loading,
  search,
  lowOnly,
  adjusting,
  adjustVal,
  pulsedProductIds,
  onAdjustValChange,
  onStartAdjust,
  onReviewAdjust,
  onCancelAdjust,
  onPolicyChange,
}: InventoryStockListProps) {
  if (loading) {
    return (
      <div className="space-y-1">
        {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <EmptyState
        variant={search || lowOnly ? 'no-results' : 'no-data'}
        headline="No inventory items found"
        body={search || lowOnly ? "Try adjusting your search filters to find what you're looking for." : "There are no products in your inventory yet."}
      />
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="overflow-x-auto rounded-xl border border-[var(--border)] bg-[var(--surface)]"
    >
      <table className="desk-table w-full text-sm">
        <thead>
          <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
            <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Product</th>
            <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">SKU</th>
            <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Quantity</th>
            <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Reorder At</th>
            <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">OOS Policy</th>
            <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Status</th>
            <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Action</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item, index) => {
            const isLow = item.quantity <= item.reorder_threshold;
            return (
              <motion.tr
                key={item.product_id}
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.03 }}
                className={`table__row border-b border-[var(--border)] last:border-0 hover:bg-[var(--default)]/50 transition-colors ${pulsedProductIds.includes(item.product_id) ? 'warehouse-inventory-sync-pulse' : ''}`}
              >
                <td className="py-3 px-4 font-medium">{item.product_name}</td>
                <td className="py-3 px-4 font-mono text-xs text-[var(--muted)]">{item.sku || '\u2014'}</td>
                <td className="py-3 px-4 text-right font-mono tabular-nums">{item.quantity}</td>
                <td className="py-3 px-4 text-right font-mono text-[var(--muted)] tabular-nums">{item.reorder_threshold}</td>
                <td className="py-3 px-4">
                  <select
                    value={item.out_of_stock_policy || 'INHERIT'}
                    onChange={(e) => { onPolicyChange(item.product_id, e.target.value); }}
                    className="px-2 py-1 rounded border text-xs outline-none focus:ring-1 focus:ring-[var(--primary)]"
                    style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
                  >
                    <option value="INHERIT">Inherit warehouse</option>
                    <option value="REJECT">Reject when OOS</option>
                    <option value="ACCEPT_BACKORDER">Accept backorder</option>
                  </select>
                </td>
                <td className="py-3 px-4">
                  {isLow ? (
                    <span className="status-chip status-chip--critical">LOW</span>
                  ) : (
                    <span className="status-chip status-chip--stable">OK</span>
                  )}
                </td>
                <td className="py-3 px-4 text-right">
                  {adjusting === item.product_id ? (
                    <div className="flex items-center gap-1 justify-end">
                      <input
                        type="number"
                        value={adjustVal}
                        onChange={e => onAdjustValChange(e.target.value)}
                        placeholder="New qty"
                        className="w-20 px-2 py-1 rounded border text-xs outline-none focus:ring-1 focus:ring-[var(--primary)]"
                        style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
                      />
                      <motion.button
                        whileHover={{ scale: 1.1 }}
                        whileTap={{ scale: 0.9 }}
                        onClick={() => onReviewAdjust(item)}
                        disabled={Number.isNaN(parseInt(adjustVal, 10))}
                        className="px-2 py-1 text-xs button--primary rounded active-press disabled:opacity-50"
                      >
                        Review
                      </motion.button>
                      <motion.button
                        whileHover={{ scale: 1.1 }}
                        whileTap={{ scale: 0.9 }}
                        onClick={() => onCancelAdjust()}
                        className="px-2 py-1 text-xs button--secondary rounded active-press"
                      >
                        X
                      </motion.button>
                    </div>
                  ) : (
                    <motion.button
                      whileHover={{ scale: 1.05, x: -2 }}
                      onClick={() => onStartAdjust(item)}
                      className="text-xs text-[var(--link)] font-medium hover:underline flex items-center gap-1 ml-auto"
                    >
                      <Icon name="refresh" size={12} /> Adjust
                    </motion.button>
                  )}
                </td>
              </motion.tr>
            );
          })}
        </tbody>
      </table>
    </motion.div>
  );
}
