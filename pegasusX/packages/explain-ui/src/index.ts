import { ExplainStatusBanner, parseExplainFromPayload } from './ExplainStatusBanner';
import type { ExplainStatusBannerProps } from './ExplainStatusBanner';
import { HandoffCard } from './HandoffCard';
import type { HandoffCardProps } from './HandoffCard';
import type { StatusExplain } from '@pegasusx/types';

export { ExplainStatusBanner, parseExplainFromPayload };
export type { ExplainStatusBannerProps };
export { HandoffCard };
export type { HandoffCardProps };

export function explainFromApiError(body: unknown): StatusExplain | null {
  return parseExplainFromPayload(body);
}
