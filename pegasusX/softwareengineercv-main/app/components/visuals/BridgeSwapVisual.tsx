'use client';

export default function BridgeSwapVisual() {
  return (
    <div className="bridge-swap" aria-hidden>
      <svg className="bridge-swap__svg" viewBox="0 0 400 500" fill="none">
        <defs>
          <linearGradient id="wireTop" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="rgba(255,255,255,0.5)" />
            <stop offset="100%" stopColor="rgba(255,255,255,0.05)" />
          </linearGradient>
          <linearGradient id="wireBottom" x1="0" y1="1" x2="0" y2="0">
            <stop offset="0%" stopColor="rgba(255,255,255,0.5)" />
            <stop offset="100%" stopColor="rgba(255,255,255,0.05)" />
          </linearGradient>
        </defs>
        <ellipse cx="200" cy="120" rx="80" ry="40" stroke="url(#wireTop)" strokeWidth="1" />
        <ellipse cx="200" cy="380" rx="80" ry="40" stroke="url(#wireBottom)" strokeWidth="1" />
        <path d="M120 120 Q200 250 120 380" stroke="rgba(255,255,255,0.2)" strokeWidth="1" />
        <path d="M280 120 Q200 250 280 380" stroke="rgba(255,255,255,0.2)" strokeWidth="1" />
        <circle cx="200" cy="250" r="28" stroke="#fff" strokeWidth="2" />
        <text x="168" y="238" fill="rgba(255,255,255,0.5)" fontSize="10" fontFamily="monospace">swap</text>
        <text x="228" y="238" fill="rgba(255,255,255,0.5)" fontSize="10" fontFamily="monospace">swap</text>
        <line x1="280" y1="100" x2="340" y2="60" stroke="rgba(255,255,255,0.4)" strokeWidth="1" />
        <text x="300" y="55" fill="#fff" fontSize="11" fontFamily="monospace">supplier lane</text>
        <line x1="120" y1="400" x2="60" y2="440" stroke="rgba(255,255,255,0.4)" strokeWidth="1" />
        <text x="20" y="448" fill="#fff" fontSize="11" fontFamily="monospace">warehouse lane</text>
      </svg>
      <div className="bridge-swap__token">P</div>
    </div>
  );
}
