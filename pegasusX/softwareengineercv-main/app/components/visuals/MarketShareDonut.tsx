'use client';

const SEGMENTS = [
  { label: 'QuickSync', value: 26, amount: '$26B' },
  { label: 'DataPulse', value: 22, amount: '$22B' },
  { label: 'CloudNest', value: 12, amount: '$12B' },
  { label: 'TaskFlow', value: 22, amount: '$22B' },
  { label: 'InsightHub', value: 7, amount: '$7B' },
  { label: 'StreamlinePro', value: 11, amount: '$11B' },
] as const;

const GRAYS = ['#ffffff', '#d4d4d4', '#a3a3a3', '#737373', '#525252', '#262626'];

export default function MarketShareDonut() {
  let cumulative = 0;
  const gradientParts = SEGMENTS.map((s, i) => {
    const start = cumulative;
    cumulative += s.value;
    return `${GRAYS[i]} ${start}% ${cumulative}%`;
  });

  return (
    <div className="market-share-card">
      <div className="market-share-card__head">
        <h3 className="market-share-card__title">Market Share</h3>
        <span className="market-share-card__menu" aria-hidden>⋯</span>
      </div>

      <div className="market-share-card__chart-wrap">
        <div
          className="market-share-card__donut"
          style={{ background: `conic-gradient(${gradientParts.join(', ')})` }}
          role="img"
          aria-label="Market share distribution across network modules"
        >
          <div className="market-share-card__donut-hole">
            <p className="market-share-card__total">$100B</p>
            <p className="market-share-card__total-label">Globe Ecosystem</p>
          </div>
        </div>
      </div>

      <div className="market-share-card__legend">
        {SEGMENTS.map((s, i) => (
          <div key={s.label} className="market-share-card__legend-item">
            <span className="market-share-card__swatch" style={{ background: GRAYS[i] }} />
            <span className="market-share-card__legend-name">{s.label}</span>
            <span className="market-share-card__legend-val">{s.amount} ({s.value}%)</span>
          </div>
        ))}
      </div>
    </div>
  );
}
