"use client";

import { usePortalT } from "@/lib/i18n";
import { useMemo } from 'react';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import type { SupplyRequest } from './types';

interface SupplyRequestHeaderProps {
  requests: SupplyRequest[];
}

export function SupplyRequestHeader({ requests }: SupplyRequestHeaderProps) {
  const t = usePortalT();
  const submittedCount = useMemo(() => requests.filter((r) => r.state === 'SUBMITTED').length, [requests]);
  const inProductionCount = useMemo(() => requests.filter((r) => r.state === 'IN_PRODUCTION').length, [requests]);
  const readyCount = useMemo(() => requests.filter((r) => r.state === 'READY').length, [requests]);
  const totalVolume = useMemo(
    () => requests.reduce((sum, r) => sum + r.total_volume_vu, 0),
    [requests],
  );

  return (
    <KpiStatGrid columns={4}>
      <KpiStatCard label={t("factory_portal.residual.text.submitted")} value={submittedCount} sub="Awaiting factory ACK" />
      <KpiStatCard label={t("factory_portal.residual.text.in_production")} value={inProductionCount} sub="Active factory work" />
      <KpiStatCard label={t("factory_portal.residual.text.ready_to_fulfill")} value={readyCount} sub="Outbound handoff queue" />
      <KpiStatCard label={t("factory_portal.residual.text.total_volume_vu")} value={totalVolume.toLocaleString()} sub={`${requests.length} requests total`} />
    </KpiStatGrid>
  );
}
