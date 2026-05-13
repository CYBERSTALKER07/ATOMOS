export type FactoryRuntimeTone = 'live' | 'refreshing' | 'warning' | 'offline' | 'error';

interface FactoryRuntimeBannerProps {
  tone: FactoryRuntimeTone;
  message: string;
}

const toneStyles: Record<FactoryRuntimeTone, { borderColor: string; background: string; color: string }> = {
  live: {
    borderColor: 'var(--color-md-outline-variant)',
    background: 'var(--color-md-surface-container-low)',
    color: 'var(--color-md-on-surface-variant)',
  },
  refreshing: {
    borderColor: 'var(--color-md-primary)',
    background: 'var(--color-md-surface-container)',
    color: 'var(--color-md-on-surface)',
  },
  warning: {
    borderColor: 'var(--color-md-warning)',
    background: 'var(--color-md-surface-container-high)',
    color: 'var(--color-md-on-surface)',
  },
  offline: {
    borderColor: 'var(--color-md-warning)',
    background: 'var(--color-md-surface-container-high)',
    color: 'var(--color-md-on-surface)',
  },
  error: {
    borderColor: 'var(--color-md-error)',
    background: 'var(--color-md-error-container)',
    color: 'var(--color-md-on-error-container)',
  },
};

export default function FactoryRuntimeBanner({ tone, message }: FactoryRuntimeBannerProps) {
  const style = toneStyles[tone];

  return (
    <div
      className="rounded-2xl border px-4 py-3 text-sm"
      style={{
        borderColor: style.borderColor,
        background: style.background,
        color: style.color,
      }}
    >
      {message}
    </div>
  );
}