import Link from 'next/link';
import { motion } from 'framer-motion';
import { PageSection } from '@/components/PageSection';
import Icon from '@/components/Icon';

type TransferState = 'APPROVED' | 'LOADING' | 'DISPATCHED';

interface Transfer {
  id: string;
  warehouse_name: string;
  total_items: number;
  total_volume_m3: number;
  state: string;
  created_at: string;
  updated_at: string;
}

interface LoadingBayGridProps {
  grouped: {
    key: TransferState;
    label: string;
    css: string;
    items: Transfer[];
  }[];
}

export default function LoadingBayGrid({ grouped }: LoadingBayGridProps) {
  return (
    <div className="mt-6 grid gap-4 xl:grid-cols-3">
      {grouped.map((column) => (
        <PageSection
          key={column.key}
          title={column.label}
          description={
            column.key === 'APPROVED'
              ? 'Approved transfers waiting for bay operators.'
              : column.key === 'LOADING'
                ? 'Transfers currently being loaded or sealed for dispatch.'
                : 'Transfers that already left the loading bay this cycle.'
          }
          actions={<span className={`status-chip ${column.css}`}>{column.items.length}</span>}
        >
          <motion.div
            className="space-y-3"
            initial="hidden"
            animate="show"
            variants={{
              hidden: { opacity: 0 },
              show: { opacity: 1, transition: { staggerChildren: 0.05 } },
            }}
          >
            {column.items.map((transfer) => (
              <Link key={transfer.id} href={`/transfers/${transfer.id}`}>
                <motion.div
                  variants={{
                    hidden: { opacity: 0, y: 10 },
                    show: { opacity: 1, y: 0 },
                  }}
                  whileTap={{ scale: 0.98 }}
                  className="block rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4 transition-colors hover:border-[var(--accent)] hover-lift active-press"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-base font-semibold text-[var(--foreground)]">{transfer.warehouse_name}</p>
                      <p className="mt-1 text-xs font-mono text-[var(--muted)]">{transfer.id}</p>
                    </div>
                    <Icon name="chevronR" size={16} className="text-[var(--muted)]" />
                  </div>

                  <div className="mt-4 grid grid-cols-2 gap-3">
                    <div className="rounded-xl bg-[var(--background)] p-3">
                      <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">Items</p>
                      <p className="mt-2 text-lg font-semibold tabular-nums text-[var(--foreground)]">{transfer.total_items}</p>
                    </div>
                    <div className="rounded-xl bg-[var(--background)] p-3">
                      <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">Volume</p>
                      <p className="mt-2 text-lg font-semibold tabular-nums text-[var(--foreground)]">{transfer.total_volume_m3.toFixed(1)} m³</p>
                    </div>
                  </div>

                  <div className="mt-4 flex items-center justify-between gap-3 text-xs text-[var(--muted)]">
                    <span>Created {new Date(transfer.created_at).toLocaleDateString()}</span>
                    <span>Updated {new Date(transfer.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                  </div>
                </motion.div>
              </Link>
            ))}

            {column.items.length === 0 && (
              <div className="rounded-2xl border border-dashed border-[var(--border)] bg-[var(--surface)] px-4 py-10 text-center text-sm text-[var(--muted)]">
                No transfers in this stage.
              </div>
            )}
          </motion.div>
        </PageSection>
      ))}
    </div>
  );
}
