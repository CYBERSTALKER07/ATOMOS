'use client';

import { useAnalytics } from '@/hooks/useAnalytics';
import { BentoGrid, BentoCard } from '@/components/BentoGrid';
import TransitHeatmap from '@/components/analytics/TransitHeatmap';
import ThroughputChart from '@/components/analytics/ThroughputChart';
import LoadDistribution from '@/components/analytics/LoadDistribution';
import NodeEfficiency from '@/components/analytics/NodeEfficiency';
import SLAHealth from '@/components/analytics/SLAHealth';

export default function IntelligencePage() {
  const {
    transitHeatmap,
    throughput,
    loadDistribution,
    nodeEfficiency,
    slaHealth,
    loading,
    error,
  } = useAnalytics();

  if (loading) {
    return (
      <div className="p-6">
        <h1 className="md-typescale-headline-small mb-6" style={{ color: 'var(--desk-text-primary)' }}>
          Intelligence Vector
        </h1>
        <div className="flex items-center justify-center h-64">
          <div className="md-typescale-body-medium" style={{ color: 'var(--desk-text-secondary)' }}>
            Initializing analytics...
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <h1 className="md-typescale-headline-small mb-6" style={{ color: 'var(--desk-text-primary)' }}>
          Intelligence Vector
        </h1>
        <div
          className="desk-card p-6 flex items-center justify-center h-48"
          style={{ background: 'var(--desk-danger-soft)', color: 'var(--desk-danger)' }}
        >
          <span className="md-typescale-body-medium">Analytics fault: {error}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <h1 className="md-typescale-headline-small mb-6" style={{ color: 'var(--desk-text-primary)' }}>
        Intelligence Vector
      </h1>

      <BentoGrid>
        {/* Row 1: Throughput (wide) + Transit Heatmap */}
        <BentoCard span={3} delay={0} className="min-h-85">
          <div
            className="desk-card p-5 h-full"
            style={{ background: 'var(--desk-surface)' }}
          >
            <ThroughputChart data={throughput} />
          </div>
        </BentoCard>

        <BentoCard span={1} delay={60} className="min-h-85">
          <div
            className="desk-card p-5 h-full"
            style={{ background: 'var(--desk-surface)' }}
          >
            <TransitHeatmap data={transitHeatmap} />
          </div>
        </BentoCard>

        {/* Row 2: Load Distribution + SLA Health */}
        <BentoCard span={2} delay={120} className="min-h-80">
          <div
            className="desk-card p-5 h-full"
            style={{ background: 'var(--desk-surface)' }}
          >
            <LoadDistribution data={loadDistribution} />
          </div>
        </BentoCard>

        <BentoCard span={2} delay={180} className="min-h-80">
          <div
            className="desk-card p-5 h-full"
            style={{ background: 'var(--desk-surface)' }}
          >
            <SLAHealth data={slaHealth} />
          </div>
        </BentoCard>

        {/* Row 3: Node Efficiency (full width) */}
        <BentoCard span={4} delay={240}>
          <div
            className="desk-card p-5"
            style={{ background: 'var(--desk-surface)' }}
          >
            <NodeEfficiency data={nodeEfficiency} />
          </div>
        </BentoCard>
      </BentoGrid>
    </div>
  );
}
