import { factoryApiBaseUrl } from '@/lib/auth';

export async function getClientPolicy(
  platform: string,
  version: string,
  channel = 'production',
): Promise<Response> {
  const params = new URLSearchParams({
    role: 'FACTORY',
    platform,
    version,
    channel,
  });
  return fetch(`${factoryApiBaseUrl}/v1/platform/client-policy?${params.toString()}`, {
    method: 'GET',
    headers: { 'X-Trace-Id': crypto.randomUUID() },
  });
}
