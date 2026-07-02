'use client';

import { useMemo } from 'react';
import { Bar, CartesianGrid, ComposedChart, Line, XAxis, YAxis } from 'recharts';
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart';
import { useReducedMotion } from '@/app/hooks/useDevice';

const QUARTERS = ['Q1', 'Q2', 'Q3', 'Q4'];

const chartConfig = {
  actual: {
    label: 'Actual',
    color: 'rgba(255,255,255,0.25)',
  },
  forecast: {
    label: 'Forecast',
    color: '#FFA500',
  },
  trend: {
    label: 'Trend',
    color: 'rgba(255,255,255,0.9)',
  },
} satisfies ChartConfig;

type LogisticsSolutionChartProps = {
  chartLabel: string;
  bars: number[];
  line: number[];
};

export default function LogisticsSolutionChart({
  chartLabel,
  bars,
  line,
}: LogisticsSolutionChartProps) {
  const prefersReducedMotion = useReducedMotion();

  const chartData = useMemo(
    () =>
      QUARTERS.map((quarter, index) => {
        const total = bars[index];
        return {
          quarter,
          actual: Math.round(total * 0.72),
          forecast: Math.round(total * 0.28),
          trend: line[index],
        };
      }),
    [bars, line]
  );

  return (
    <div className="flex h-full min-h-[18rem] flex-col justify-end border-l border-white/10 bg-[#141414] p-6 md:p-8">
      <p className="mb-4 font-mono text-[0.65rem] uppercase tracking-[0.18em] text-white/50">
        {chartLabel}
      </p>

      <ChartContainer
        config={chartConfig}
        className="aspect-auto h-44 w-full md:h-52 [&_.recharts-cartesian-axis-tick_text]:fill-white/35 [&_.recharts-cartesian-grid_line]:stroke-white/[0.07]"
      >
        <ComposedChart data={chartData} margin={{ top: 8, right: 8, bottom: 0, left: -16 }}>
          <CartesianGrid vertical={false} />
          <XAxis
            dataKey="quarter"
            tickLine={false}
            axisLine={false}
            tickMargin={12}
            tick={{ fontSize: 10, fontFamily: 'monospace' }}
          />
          <YAxis hide domain={[0, 'dataMax + 8']} />
          <ChartTooltip
            cursor={{ fill: 'rgba(255,255,255,0.04)' }}
            content={
              <ChartTooltipContent
                className="border-white/15 bg-[#1a1a1a] text-white"
                indicator="dot"
              />
            }
          />
          <Bar
            dataKey="actual"
            stackId="metrics"
            fill="var(--color-actual)"
            maxBarSize={52}
            isAnimationActive={!prefersReducedMotion}
          />
          <Bar
            dataKey="forecast"
            stackId="metrics"
            fill="var(--color-forecast)"
            radius={[2, 2, 0, 0]}
            maxBarSize={52}
            isAnimationActive={!prefersReducedMotion}
          />
          <Line
            type="monotone"
            dataKey="trend"
            stroke="var(--color-trend)"
            strokeWidth={2}
            dot={false}
            isAnimationActive={!prefersReducedMotion}
          />
        </ComposedChart>
      </ChartContainer>

      <div className="flex flex-wrap gap-4 border-t border-white/10 pt-2 font-mono text-[0.6rem] uppercase tracking-wider text-white/45">
        <span className="flex items-center gap-2">
          <span className="h-2 w-2 bg-[#FFA500]" /> Forecast
        </span>
        <span className="flex items-center gap-2">
          <span className="h-2 w-2 bg-white/25" /> Actual
        </span>
        <span className="flex items-center gap-2">
          <span className="h-2 w-2 border border-white/40" /> Trend
        </span>
      </div>
    </div>
  );
}
