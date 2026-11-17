import { fetchAuthSession, fiscalReceiptLabel, homeCellFromJwt, pinApiBaseUrl, type MarketPack } from '../../packages/api-core';

const BOOTSTRAP = (process.env.EXPO_PUBLIC_API_URL?.trim() || '') ||
  (__DEV__ ? 'http://localhost:8180' : (process.env.EXPO_PUBLIC_RELEASE_API_URL?.trim() || 'https://api.pegasusx.app'));

let cachedPack: MarketPack | null = null;

export function payloadApiBaseUrl(token?: string): string {
  return pinApiBaseUrl({
    bootstrap: BOOTSTRAP,
    homeCell: token ? homeCellFromJwt(token) : '',
  });
}

export function cachedPayloadPack(): MarketPack | null {
  return cachedPack;
}

export function payloadPackLabel(pack: MarketPack | null): string {
  if (!pack?.currency_code) return '';
  return `${pack.currency_code} · receipts: ${fiscalReceiptLabel(pack.fiscal_adapter)}`;
}

export async function bindPayloadPack(token: string): Promise<MarketPack | null> {
  const session = await fetchAuthSession(payloadApiBaseUrl(token), token);
  cachedPack = session.pack;
  return cachedPack;
}
