'use client';

import type { HandoffCardMetadata } from '@pegasusx/types';

export type HandoffCardProps = {
  metadata: HandoffCardMetadata;
  className?: string;
  onAction?: (link: string) => void;
};

export function HandoffCard({ metadata, className = '', onAction }: HandoffCardProps) {
  const link = metadata.primary_link ?? '';
  return (
    <article className={`handoff-card ${className}`.trim()}>
      <header className="handoff-card__header">
        <h4 className="handoff-card__title">{metadata.title}</h4>
        {metadata.subtitle ? <p className="handoff-card__subtitle">{metadata.subtitle}</p> : null}
      </header>
      {metadata.fields && Object.keys(metadata.fields).length > 0 ? (
        <dl className="handoff-card__fields">
          {Object.entries(metadata.fields).map(([key, value]) => (
            <div key={key} className="handoff-card__field">
              <dt>{key.replace(/_/g, ' ')}</dt>
              <dd>{value}</dd>
            </div>
          ))}
        </dl>
      ) : null}
      {link ? (
        <button type="button" className="handoff-card__cta" onClick={() => onAction?.(link)}>
          {metadata.primary_cta ?? 'Open'}
        </button>
      ) : null}
    </article>
  );
}
