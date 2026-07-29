import { BarChart3, Brain, TrendingUp, Zap } from "lucide-react";
import { BentoGrid, BentoCard } from "../../components/BentoGrid";
import CountUp from "../../components/CountUp";
import MiniSparkline from "../../components/MiniSparkline";
import type { Prediction } from "../../lib/types";

interface InsightsSummaryProps {
  totalThisMonth: number;
  sparkRevenue: number[];
  predList: Prediction[];
  sparkOrders: number[];
  topProducts: any[];
}

export function InsightsSummary({
  totalThisMonth,
  sparkRevenue,
  predList,
  sparkOrders,
  topProducts,
}: InsightsSummaryProps) {
  return (
    <BentoGrid className="mb-8">
      <BentoCard interactive={false}>
        <div className="flex flex-col gap-1">
          <div className="flex items-center justify-between mb-2">
            <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
              Monthly Spend
            </span>
            <BarChart3 size={18} style={{ color: "var(--desk-accent)" }} />
          </div>
          <div className="flex items-end justify-between">
            <CountUp
              end={totalThisMonth}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <MiniSparkline data={sparkRevenue} width={80} height={32} />
          </div>
        </div>
      </BentoCard>

      <BentoCard interactive={false}>
        <div className="flex flex-col gap-1">
          <div className="flex items-center justify-between mb-2">
            <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
              AI Signals
            </span>
            <Brain size={18} style={{ color: "var(--desk-info)" }} />
          </div>
          <div className="flex items-end justify-between">
            <CountUp
              end={predList.length}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <MiniSparkline data={sparkOrders} width={80} height={32} />
          </div>
        </div>
      </BentoCard>

      <BentoCard interactive={false}>
        <div className="flex flex-col gap-1">
          <div className="flex items-center justify-between mb-2">
            <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
              Top Product
            </span>
            <TrendingUp size={18} style={{ color: "var(--desk-success)" }} />
          </div>
          <span className="md-typescale-title-medium font-light truncate text-[var(--desk-text-primary)]">
            {topProducts[0]?.product_name || "Calculating..."}
          </span>
          <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
            {topProducts[0]?.quantity || 0} units staged
          </p>
        </div>
      </BentoCard>

      <BentoCard interactive={false}>
        <div className="flex flex-col gap-1">
          <div className="flex items-center justify-between mb-2">
            <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
              Efficiency
            </span>
            <Zap size={18} style={{ color: "var(--desk-warning)" }} />
          </div>
          <CountUp
            end={94}
            className="md-typescale-metric text-[var(--desk-text-primary)]"
            suffix="%"
          />
          <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
            Prediction precision
          </p>
        </div>
      </BentoCard>
    </BentoGrid>
  );
}
