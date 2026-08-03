import { ForecastConfidenceView } from '@/components/ForecastConfidenceView';

export function ForecastSkuTable({ products }: { products: any[] }) {
  return (
    <div className="border border-[var(--border)] rounded-xl overflow-hidden">
      <table className="desk-table w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
            <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Product</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Stock</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Recommended</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Stockout</th>
            <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Priority</th>
            <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Confidence</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Incoming</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">AI Pred</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Pre-Orders</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Burn/Day</th>
          </tr>
        </thead>
        <tbody>
          {products.map(p => (
            <tr key={p.product_id} className="border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--surface)] transition-colors">
              <td className="px-4 py-3">{p.product_name || p.product_id.slice(0, 8)}</td>
              <td className="px-4 py-3 text-right font-mono">{p.current_stock}</td>
              <td className="px-4 py-3 text-right font-mono font-semibold">{p.recommended_qty}</td>
              <td className="px-4 py-3 text-right">
                <span className={
                  p.days_until_stockout < 2 ? 'text-[var(--danger)] font-semibold' :
                  p.days_until_stockout < 5 ? 'text-[var(--warning)]' : ''
                }>
                  {p.days_until_stockout.toFixed(1)}d
                </span>
              </td>
              <td className="px-4 py-3">
                <span className={`text-xs font-semibold ${
                  p.priority === 'CRITICAL' ? 'text-[var(--danger)]' :
                  p.priority === 'URGENT' ? 'text-[var(--warning)]' : 'text-[var(--muted)]'
                }`}>{p.priority}</span>
              </td>
              <td className="px-4 py-3">
                {p.confidence ? (
                  <ForecastConfidenceView confidence={p.confidence} compact />
                ) : (
                  <span className="text-xs text-[var(--muted)]">—</span>
                )}
              </td>
              <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.incoming_orders || 0}</td>
              <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.ai_prediction || 0}</td>
              <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.pre_orders || 0}</td>
              <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.burn_rate?.toFixed(1) || '0.0'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
