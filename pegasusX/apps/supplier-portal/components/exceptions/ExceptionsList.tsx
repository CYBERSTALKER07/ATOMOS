import React from 'react';
import type { SupplierExceptionRow } from '@pegasusx/types';
import StatusBadge from '@/components/StatusBadge';

interface ExceptionsListProps {
  exceptions: SupplierExceptionRow[];
}

export function ExceptionsList({ exceptions }: ExceptionsListProps) {
  if (exceptions.length === 0) return null;
  return (
    <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
      {exceptions.map((row) => (
        <li key={row.order_id} className="p-4 md-typescale-body-medium">
          <div className="flex flex-wrap items-center gap-2">
            <span className="md-chip h-6 text-xs">{row.kind}</span>
            <span className="font-mono text-[var(--color-md-primary)]">{row.order_id}</span>
            <StatusBadge state={row.status} />
          </div>
          {row.note ? <p className="mt-2 text-[var(--color-md-outline)]">{row.note}</p> : null}
          <p className="mt-1 text-sm text-[var(--color-md-outline)]">
            Updated {new Date(row.updated_at).toLocaleString()}
          </p>
        </li>
      ))}
    </ul>
  );
}
