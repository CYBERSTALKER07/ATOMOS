'use client';

import type { StatusExplain } from '@pegasusx/types';

export type ExplainStatusBannerProps = {
  errorCode?: string | null;
  explain?: StatusExplain | null;
  detail?: string | null;
  className?: string;
};

export function ExplainStatusBanner({ errorCode, explain, detail, className = '' }: ExplainStatusBannerProps) {
  if (!errorCode && !explain) return null;
  const title = explain?.title ?? errorCode ?? 'Something went wrong';
  const summary = explain?.summary ?? detail ?? '';
  return (
    <div className={`explain-status-banner ${className}`.trim()} role="alert">
      <strong className="explain-status-banner__title">{title}</strong>
      {summary ? <p className="explain-status-banner__summary">{summary}</p> : null}
      {explain?.next_steps && explain.next_steps.length > 0 ? (
        <ul className="explain-status-banner__steps">
          {explain.next_steps.map((step) => (
            <li key={step}>{step}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

export function parseExplainFromPayload(payload: unknown): StatusExplain | null {
  if (!payload || typeof payload !== 'object') return null;
  const explain = (payload as { explain?: StatusExplain }).explain;
  if (!explain || typeof explain.title !== 'string') return null;
  return explain;
}
