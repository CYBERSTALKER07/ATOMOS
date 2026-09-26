'use client';

import { SessionPackChip as GenericSessionPackChip } from '@pegasusx/ui-kit/portal';
import { readToken, retailerApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={retailerApiBaseUrl()} token={readToken()} />;
}
