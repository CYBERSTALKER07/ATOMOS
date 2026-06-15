import { supplierApiBaseUrl } from '@/lib/auth';

export async function getClientPolicy(
  platform: string,
  version: string,
  channel = 'production',
): Promise<Response> {
  const params = new URLSearchParams({
    role: 'ADMIN',
    platform,
    version,
    channel,
  });
  return fetch(`${supplierApiBaseUrl()}/v1/platform/client-policy?${params.toString()}`, {
    method: 'GET',
    headers: { 'X-Trace-Id': crypto.randomUUID() },
  });
}
