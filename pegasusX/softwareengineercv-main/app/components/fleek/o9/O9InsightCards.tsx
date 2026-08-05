'use client';

import Link from 'next/link';
import { O9SectionLabel } from '@/app/components/page-sections/o9/O9Sections';

type InsightCard = {
  id: string;
  title: string;
  description: string;
  href: string;
  visual: 'line-dual' | 'line-metrics' | 'bars';
};

const CARDS: InsightCard[] = [
  {
    id: 'dispatch',
    title: 'Smart dispatch & load match',
    description:
      'Match orders to trucks from forecasted demand, capacity, and warehouse constraints — with human override on every assignment.',
    href: '/capabilities/smarter-dispatch',
    visual: 'line-dual',
  },
  {
    id: 'fleet',
    title: 'Live fleet & on-time delivery',
    description:
      'See planned vs actual routes across the network. Spot deviations before retailers call and keep every role on one ETA.',
    href: '/capabilities/live-fleet-tracking',
    visual: 'line-metrics',
  },
  {
    id: 'payments',
    title: 'Payment & gate confidence',
    description:
      'Reconcile COD, gate seals, and capacity utilization so treasury and payload teams close the day without spreadsheet fire drills.',
    href: '/capabilities/payment-confidence',
    visual: 'bars',
  },
];

function ChartLineDual() {
  return (
    <div className="o9-insight-viz" aria-hidden>
      <p className="o9-insight-viz__caption">Category: peak vs baseline load</p>
      <svg viewBox="0 0 320 140" className="o9-insight-viz__svg">
        <g stroke="rgba(255,255,255,0.08)" strokeWidth="1">
          {[28, 56, 84, 112].map((y) => (
            <line key={y} x1="24" y1={y} x2="304" y2={y} />
          ))}
        </g>
        <polyline
          fill="none"
          stroke="rgba(255,255,255,0.35)"
          strokeWidth="2"
          points="28,96 70,88 112,92 154,78 196,82 238,70 280,74"
        />
        <polyline
          fill="none"
          stroke="#f97316"
          strokeWidth="2.5"
          points="28,110 70,98 112,84 154,72 196,58 238,46 280,38"
        />
        {[
          [28, 110],
          [70, 98],
          [112, 84],
          [154, 72],
          [196, 58],
          [238, 46],
          [280, 38],
        ].map(([x, y], i) => (
          <circle key={i} cx={x} cy={y} r="4" fill="#f97316" />
        ))}
        <text x="24" y="134" fill="rgba(255,255,255,0.35)" fontSize="9" fontFamily="ui-monospace,monospace">
          OPTIONS
        </text>
        <text
          x="8"
          y="72"
          fill="rgba(255,255,255,0.35)"
          fontSize="9"
          fontFamily="ui-monospace,monospace"
          transform="rotate(-90 8 72)"
        >
          ORDERS
        </text>
      </svg>
    </div>
  );
}

function ChartLineMetrics() {
  return (
    <div className="o9-insight-viz" aria-hidden>
      <p className="o9-insight-viz__caption">Store / retailer traffic</p>
      <svg viewBox="0 0 320 88" className="o9-insight-viz__svg o9-insight-viz__svg--compact">
        <polyline
          fill="none"
          stroke="rgba(255,255,255,0.3)"
          strokeWidth="2"
          points="20,60 70,52 120,56 170,40 220,44 270,30 300,34"
        />
        <polyline
          fill="none"
          stroke="#f97316"
          strokeWidth="2.5"
          points="20,70 70,58 120,48 170,36 220,28 270,22 300,18"
        />
        {[
          [20, 70],
          [120, 48],
          [220, 28],
          [300, 18],
        ].map(([x, y], i) => (
          <circle key={i} cx={x} cy={y} r="4" fill="#f97316" />
        ))}
      </svg>
      <div className="o9-insight-viz__metrics">
        <div>
          <span>On-time %</span>
          <strong>94.2</strong>
        </div>
        <div>
          <span>Active trucks</span>
          <strong>128</strong>
        </div>
        <div>
          <span>Live stops</span>
          <strong>86%</strong>
        </div>
      </div>
    </div>
  );
}

function ChartBars() {
  const bars = [
    { total: 92, planned: 70 },
    { total: 78, planned: 62 },
    { total: 88, planned: 80 },
    { total: 70, planned: 48 },
    { total: 96, planned: 84 },
    { total: 82, planned: 66 },
  ];
  return (
    <div className="o9-insight-viz" aria-hidden>
      <p className="o9-insight-viz__caption">Gate / RDC capacity utilisation</p>
      <svg viewBox="0 0 320 140" className="o9-insight-viz__svg">
        <line x1="24" y1="40" x2="304" y2="40" stroke="rgba(255,255,255,0.45)" strokeDasharray="4 4" />
        {bars.map((b, i) => {
          const x = 36 + i * 44;
          const totalH = b.total * 0.9;
          const plannedH = b.planned * 0.9;
          return (
            <g key={i}>
              <rect x={x} y={120 - totalH} width="14" height={totalH} fill="rgba(255,255,255,0.28)" />
              <rect x={x + 16} y={120 - plannedH} width="14" height={plannedH} fill="#f97316" />
            </g>
          );
        })}
      </svg>
    </div>
  );
}

function Visual({ kind }: { kind: InsightCard['visual'] }) {
  if (kind === 'line-metrics') return <ChartLineMetrics />;
  if (kind === 'bars') return <ChartBars />;
  return <ChartLineDual />;
}

type O9InsightCardsProps = {
  eyebrow?: string;
  title?: string;
};

export default function O9InsightCards({
  eyebrow = 'Capabilities',
  title = 'Plan, dispatch, and settle on one picture',
}: O9InsightCardsProps) {
  return (
    <section className="o9-section o9-insight-cards">
      <O9SectionLabel>{eyebrow}</O9SectionLabel>
      <h2 className="o9-section__title">{title}</h2>
      <div className="o9-insight-cards__grid">
        {CARDS.map((card) => (
          <article key={card.id} className="o9-insight-card">
            <Visual kind={card.visual} />
            <h3 className="o9-insight-card__title">{card.title}</h3>
            <p className="o9-insight-card__copy">{card.description}</p>
            <Link href={card.href} className="o9-insight-card__link">
              Read more <span aria-hidden="true">&gt;</span>
            </Link>
          </article>
        ))}
      </div>
    </section>
  );
}
