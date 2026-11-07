'use client';

import { SessionPackChip as GenericSessionPackChip } from '@pegasusx/ui-kit/portal';
import { readTokenFromCookie, supplierApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={supplierApiBaseUrl()} token={readTokenFromCookie()} />;
}
