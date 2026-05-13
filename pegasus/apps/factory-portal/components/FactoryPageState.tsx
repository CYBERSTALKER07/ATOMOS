import type { ReactNode } from 'react';

import EmptyState from './EmptyState';
import { PageSkeleton } from './Skeleton';

export type FactoryPageStateKind =
  | 'loading'
  | 'empty'
  | 'no-results'
  | 'error'
  | 'offline'
  | 'restricted'
  | 'auth-failure';

interface FactoryPageStateProps {
  kind: FactoryPageStateKind;
  title?: string;
  subtitle?: string;
  headline?: string;
  body?: string;
  actionLabel?: string;
  onAction?: () => void;
  imageUrl?: string;
  skeleton?: ReactNode;
  className?: string;
}

const variantMap: Record<Exclude<FactoryPageStateKind, 'loading'>, 'no-data' | 'no-results' | 'error' | 'offline' | 'restricted'> = {
  empty: 'no-data',
  'no-results': 'no-results',
  error: 'error',
  offline: 'offline',
  restricted: 'restricted',
  'auth-failure': 'restricted',
};

export default function FactoryPageState({
  kind,
  title,
  subtitle,
  headline,
  body,
  actionLabel,
  onAction,
  imageUrl,
  skeleton,
  className = '',
}: FactoryPageStateProps) {
  if (kind === 'loading') {
    return (
      <div className={`space-y-4 ${className}`.trim()}>
        {(title || subtitle) && (
          <div>
            {title && <h1 className="text-xl font-semibold">{title}</h1>}
            {subtitle && (
              <p className="mt-1 text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                {subtitle}
              </p>
            )}
          </div>
        )}
        {skeleton ?? <PageSkeleton />}
      </div>
    );
  }

  return (
    <div className={`space-y-4 ${className}`.trim()}>
      {(title || subtitle) && (
        <div>
          {title && <h1 className="text-xl font-semibold">{title}</h1>}
          {subtitle && (
            <p className="mt-1 text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              {subtitle}
            </p>
          )}
        </div>
      )}
      <EmptyState
        variant={variantMap[kind]}
        imageUrl={imageUrl}
        headline={headline ?? title ?? 'State unavailable'}
        body={body}
        action={actionLabel}
        onAction={onAction}
      />
    </div>
  );
}