import { warehouseApiBaseUrl } from '@/lib/auth';

export async function getClientPolicy(
  platform: string,
  version: string,
  channel = 'production',
): Promise<Response> {
  const params = new URLSearchParams({
    role: 'WAREHOUSE',
    platform,
    version,
    channel,
  });
  return fetch(`${warehouseApiBaseUrl}/v1/platform/client-policy?${params.toString()}`, {
    method: 'GET',
    headers: { 'X-Trace-Id': crypto.randomUUID() },
  });
}
