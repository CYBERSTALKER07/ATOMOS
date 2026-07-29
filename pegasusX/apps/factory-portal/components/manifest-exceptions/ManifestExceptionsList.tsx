import { PageSection } from '@/components/PageSection';

export interface ManifestException {
  exception_id: string;
  manifest_id: string;
  transfer_id: string;
  reason: string;
  metadata?: string;
  attempt_count: number;
  escalated: boolean;
  created_at: string;
  correlation_id?: string;
}

const REASON_COLORS: Record<string, string> = {
  OVERFLOW: 'var(--color-md-warning)',
  DAMAGED: 'var(--color-md-error)',
  MANUAL: 'var(--color-md-info)',
  NO_CAPACITY: 'var(--color-md-error)',
};

export function shortId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}…` : id;
}

export function reasonBadge(reason: string) {
  return (
    <span
      className="md-typescale-label-small md-shape-sm px-2 py-0.5 inline-block"
      style={{
        background: REASON_COLORS[reason] || 'var(--color-md-outline)',
        color: '#fff',
      }}
    >
      {reason}
    </span>
  );
}

interface ManifestExceptionsListProps {
  exceptions: ManifestException[];
}

export function ManifestExceptionsList({ exceptions }: ManifestExceptionsListProps) {
  return (
    <PageSection title="Exception inbox" description="Rows highlighted when attempt count reaches DLQ threshold." className="mt-6">
      <div className="overflow-x-auto -mx-5 px-5">
        <table className="desk-table w-full text-sm">
          <thead>
            <tr className="border-b" style={{ borderColor: 'var(--desk-border)' }}>
              {['Transfer', 'Manifest', 'Reason', 'Attempts', 'Escalated', 'Time'].map((h) => (
                <th key={h} className="px-4 py-3 text-left font-medium" style={{ color: 'var(--desk-text-secondary)' }}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {exceptions.map((ex) => (
              <tr
                key={ex.exception_id}
                className="border-t"
                style={{
                  borderColor: 'var(--desk-border)',
                  background: ex.attempt_count >= 3 ? 'var(--color-md-error-container)' : undefined,
                }}
              >
                <td className="px-4 py-3 font-mono">{shortId(ex.transfer_id)}</td>
                <td className="px-4 py-3 font-mono">{shortId(ex.manifest_id)}</td>
                <td className="px-4 py-3">{reasonBadge(ex.reason)}</td>
                <td className="px-4 py-3">
                  <span
                    className={ex.attempt_count >= 3 ? 'font-light' : ''}
                    style={{ color: ex.attempt_count >= 3 ? 'var(--color-md-error)' : 'var(--desk-text-primary)' }}
                  >
                    {ex.attempt_count}
                    {ex.attempt_count >= 3 && ' — DLQ'}
                  </span>
                </td>
                <td className="px-4 py-3">{ex.escalated ? 'Yes' : 'No'}</td>
                <td className="px-4 py-3" style={{ color: 'var(--desk-text-secondary)' }}>
                  {ex.created_at ? new Date(ex.created_at).toLocaleString() : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </PageSection>
  );
}
