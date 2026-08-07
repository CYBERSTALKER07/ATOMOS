'use client';

import { useLanguage } from '@/app/context/LanguageContext';

import type { CSSProperties } from 'react';

const NODES = [
  { x: 18, y: 35 },
  { x: 18, y: 65 },
  { x: 82, y: 30 },
  { x: 82, y: 70 },
] as const;

export default function AgentNetworkHero() {
  const { language } = useLanguage();

  return (
    <div className="agent-network-hero">
      <div className="agent-network-hero__diagram" aria-hidden>
        {NODES.map((n, i) => (
          <svg
            key={i}
            className="agent-network-hero__connector"
            style={{ '--nx': `${n.x}%`, '--ny': `${n.y}%` } as CSSProperties}
            viewBox="0 0 200 100"
            preserveAspectRatio="none"
          >
            <path
              d="M0 50 Q100 20 200 50"
              stroke="rgba(255,255,255,0.15)"
              strokeWidth="1"
              fill="none"
            />
          </svg>
        ))}
        {NODES.map((n, i) => (
          <div
            key={`node-${i}`}
            className="agent-network-hero__node"
            style={{ left: `${n.x}%`, top: `${n.y}%` }}
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
              <circle cx="12" cy="8" r="4" stroke="currentColor" strokeWidth="1.5" />
              <path d="M6 20c0-3.3 2.7-6 6-6s6 2.7 6 6" stroke="currentColor" strokeWidth="1.5" />
            </svg>
          </div>
        ))}
        <div className="agent-network-hero__hub">
          <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
            <path d="M24 8 L32 16 L32 32 L24 40 L16 32 L16 16 Z" stroke="#fff" strokeWidth="1.5" />
            <path d="M24 16 L28 20 L28 28 L24 32 L20 28 L20 20 Z" stroke="#fff" strokeWidth="1" opacity="0.5" />
          </svg>
        </div>
        <div className="agent-network-hero__glow-line" />
      </div>
      <div className="agent-network-hero__copy">
        <h3 className="agent-network-hero__title">{language === 'ru' ? 'Агенты, которые работают, пока вы отдыхаете.' : 'Agents who work while you dream.'}</h3>
        <p className="agent-network-hero__mono">
          From depot to remote networks,<br />
          dispatch assist never sleeps.<br />
          Plan. Load. Automate.
        </p>
      </div>
    </div>
  );
}
