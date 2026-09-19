'use client';

export default function WireframeGlobe() {
  return (
    <div className="fleek-globe" aria-hidden>
      <svg className="fleek-globe__svg" viewBox="0 0 400 400" fill="none">
        <circle cx="200" cy="200" r="140" stroke="rgba(255,255,255,0.15)" strokeWidth="1" />
        <ellipse cx="200" cy="200" rx="140" ry="50" stroke="rgba(255,255,255,0.12)" strokeWidth="1" />
        <ellipse cx="200" cy="200" rx="50" ry="140" stroke="rgba(255,255,255,0.12)" strokeWidth="1" />
        <ellipse cx="200" cy="200" rx="100" ry="140" stroke="rgba(255,255,255,0.08)" strokeWidth="1" transform="rotate(30 200 200)" />
        <ellipse cx="200" cy="200" rx="100" ry="140" stroke="rgba(255,255,255,0.08)" strokeWidth="1" transform="rotate(-30 200 200)" />
        {[
          [120, 160],
          [280, 140],
          [300, 220],
          [180, 280],
          [100, 240],
          [220, 120],
        ].map(([x, y], i) => (
          <g key={i}>
            <circle cx={x} cy={y} r="6" fill="var(--fleek-accent)" />
            <circle cx={x} cy={y} r="12" stroke="var(--fleek-accent)" strokeWidth="1" opacity="0.4" />
          </g>
        ))}
        <line x1="120" y1="160" x2="220" y2="120" stroke="rgba(255,255,255,0.2)" strokeWidth="1" />
        <line x1="220" y1="120" x2="280" y2="140" stroke="rgba(255,255,255,0.2)" strokeWidth="1" />
        <line x1="280" y1="140" x2="300" y2="220" stroke="rgba(255,255,255,0.2)" strokeWidth="1" />
        <line x1="300" y1="220" x2="180" y2="280" stroke="rgba(255,255,255,0.2)" strokeWidth="1" />
        <line x1="180" y1="280" x2="100" y2="240" stroke="rgba(255,255,255,0.2)" strokeWidth="1" />
        <line x1="100" y1="240" x2="120" y2="160" stroke="rgba(255,255,255,0.2)" strokeWidth="1" />
      </svg>
      <div className="fleek-globe__bolt">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
          <path d="M13 2L4 14h7l-1 8 9-12h-7l1-8z" fill="var(--fleek-accent)" />
        </svg>
      </div>
    </div>
  );
}
