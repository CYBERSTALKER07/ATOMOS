export { ExplainStatusBanner, parseExplainFromPayload } from './ExplainStatusBanner';
export type { ExplainStatusBannerProps } from './ExplainStatusBanner';
export { HandoffCard } from './HandoffCard';
export type { HandoffCardProps } from './HandoffCard';

export function explainFromApiError(body: unknown): import('@pegasusx/types').StatusExplain | null {
  return parseExplainFromPayload(body);
}
