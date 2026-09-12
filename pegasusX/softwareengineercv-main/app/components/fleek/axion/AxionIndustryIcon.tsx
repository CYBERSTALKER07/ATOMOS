type AxionIndustryIconProps = {
  type: 'retail' | 'health' | 'tech' | 'manufacturing' | 'fleet' | 'warehouse';
};

export default function AxionIndustryIcon({ type }: AxionIndustryIconProps) {
  const common = { width: 20, height: 20, viewBox: '0 0 24 24', fill: 'none', 'aria-hidden': true as const };

  switch (type) {
    case 'retail':
      return (
        <svg {...common}>
          <path d="M4 7h16v12H4zM8 7V5a2 2 0 012-2h4a2 2 0 012 2v2" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      );
    case 'health':
      return (
        <svg {...common}>
          <path d="M12 8v8M8 12h8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          <rect x="4" y="4" width="16" height="16" rx="4" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      );
    case 'tech':
      return (
        <svg {...common}>
          <rect x="3" y="3" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="1.5" />
          <rect x="14" y="3" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="1.5" />
          <rect x="3" y="14" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="1.5" />
          <rect x="14" y="14" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      );
    case 'manufacturing':
      return (
        <svg {...common}>
          <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="1.5" />
          <path d="M12 2v3M12 19v3M2 12h3M19 12h3" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      );
    case 'fleet':
      return (
        <svg {...common}>
          <path d="M3 17h1M6 17h2M15 17h2M19 17h1M5 17V9h10v8" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      );
    default:
      return (
        <svg {...common}>
          <path d="M4 10h16v10H4zM8 10V7h8v3" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      );
  }
}
