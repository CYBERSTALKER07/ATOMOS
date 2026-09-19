'use client';

import { useLanguage } from '@/app/context/LanguageContext';

const NODES = ['Dispatch', 'Fleet', 'Treasury', 'Tracking', 'Topology', 'Payments'] as const;

export default function IntegrationsHubVisual() {
  const { language } = useLanguage();

  return (
    <div className="integrations-hub">
      <div className="integrations-hub__stage" aria-hidden>
        <div className="integrations-hub__ring">
          <span>{language === 'ru' ? 'АВТОМАТИЗАЦИЯ БЕЗ ГРАНИЦ' : 'AUTOMATIONS UNLEASHED'}</span>
        </div>
        {NODES.map((label, i) => {
          const angle = (i / NODES.length) * Math.PI * 2 - Math.PI / 2;
          const x = 50 + Math.cos(angle) * 38;
          const y = 50 + Math.sin(angle) * 38;
          return (
            <div
              key={label}
              className="integrations-hub__hex"
              style={{ left: `${x}%`, top: `${y}%` }}
            >
              {label.slice(0, 2)}
            </div>
          );
        })}
        <svg className="integrations-hub__wires" viewBox="0 0 200 200">
          {NODES.map((_, i) => {
            const angle = (i / NODES.length) * Math.PI * 2 - Math.PI / 2;
            const x = 100 + Math.cos(angle) * 70;
            const y = 100 + Math.sin(angle) * 70;
            return (
              <line
                key={i}
                x1="100"
                y1="100"
                x2={x}
                y2={y}
                stroke="rgba(255,255,255,0.12)"
                strokeWidth="1"
              />
            );
          })}
        </svg>
      </div>
      <p className="integrations-hub__caption">
        Six role surfaces share one integration spine — portal, mobile, and desktop on the same contracts.
      </p>
    </div>
  );
}
