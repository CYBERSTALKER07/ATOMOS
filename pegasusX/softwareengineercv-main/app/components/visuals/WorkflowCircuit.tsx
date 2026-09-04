'use client';

import { useLanguage } from '@/app/context/LanguageContext';

export default function WorkflowCircuit() {
  const { language } = useLanguage();

  return (
    <div className="workflow-circuit" aria-hidden>
      <svg className="workflow-circuit__svg" viewBox="0 0 800 400" fill="none">
        <defs>
          <linearGradient id="lineFadeL" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="transparent" />
            <stop offset="30%" stopColor="rgba(255,255,255,0.5)" />
            <stop offset="100%" stopColor="rgba(255,255,255,0.15)" />
          </linearGradient>
          <linearGradient id="lineFadeR" x1="100%" y1="0%" x2="0%" y2="0%">
            <stop offset="0%" stopColor="transparent" />
            <stop offset="30%" stopColor="rgba(255,255,255,0.5)" />
            <stop offset="100%" stopColor="rgba(255,255,255,0.15)" />
          </linearGradient>
        </defs>
        <path d="M120 200 H280 Q320 200 320 160 V80" stroke="url(#lineFadeL)" strokeWidth="1.5" />
        <path d="M120 200 H280 Q320 200 320 240 V320" stroke="url(#lineFadeL)" strokeWidth="1.5" />
        <path d="M480 200 H640 Q680 200 680 160 V80" stroke="url(#lineFadeR)" strokeWidth="1.5" />
        <path d="M480 200 H640 Q680 200 680 240 V320" stroke="url(#lineFadeR)" strokeWidth="1.5" />
        <path d="M200 120 H600" stroke="rgba(255,255,255,0.08)" strokeWidth="1" />
        <circle cx="200" cy="120" r="4" fill="#fff" />
        <rect x="196" y="276" width="8" height="8" fill="#fff" />
        <polygon points="400,180 410,200 390,200" stroke="#fff" strokeWidth="1" fill="none" />
        <polygon points="560,180 570,200 550,200" stroke="#fff" strokeWidth="1" fill="none" />
        <rect x="130" y="188" width="48" height="48" rx="12" stroke="rgba(255,255,255,0.35)" strokeWidth="1.5" />
        <circle cx="154" cy="212" r="8" stroke="#fff" strokeWidth="1.5" />
        <line x1="178" y1="212" x2="220" y2="212" stroke="rgba(255,255,255,0.4)" strokeWidth="1.5" />
      </svg>
      <p className="workflow-circuit__tag">{language === 'ru' ? 'Low-code автоматизация операций' : 'Low-code ops automation'}</p>
      <p className="workflow-circuit__brand">pegasus</p>
    </div>
  );
}
