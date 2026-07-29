import { useMemo } from 'react';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import type { SupplyRequest } from './types';

interface SupplyRequestHeaderProps {
  requests: SupplyRequest[];
}

export function SupplyRequestHeader({ requests }: SupplyRequestHeaderProps) {
  const submittedCount = useMemo(() => requests.filter((r) => r.state === 'SUBMITTED').length, [requests]);
  const inProductionCount = useMemo(() => requests.filter((r) => r.state === 'IN_PRODUCTION').length, [requests]);
  const readyCount = useMemo(() => requests.filter((r) => r.state === 'READY').length, [requests]);
  const totalVolume = useMemo(
    () => requests.reduce((sum, r) => sum + r.total_volume_vu, 0),
    [requests],
  );

  return (
    <KpiStatGrid columns={4}>
      <KpiStatCard label="Submitted" value={submittedCount} sub="Awaiting factory ACK" />
      <KpiStatCard label="In production" value={inProductionCount} sub="Active factory work" />
      <KpiStatCard label="Ready to fulfill" value={readyCount} sub="Outbound handoff queue" />
      <KpiStatCard label="Total volume (VU)" value={totalVolume.toLocaleString()} sub={`${requests.length} requests total`} />
    </KpiStatGrid>
  );
}
