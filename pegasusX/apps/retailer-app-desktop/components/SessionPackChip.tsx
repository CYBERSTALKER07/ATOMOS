'use client';

import { SessionPackChip as GenericSessionPackChip } from '@/components/ui/SessionPackChip';
import { readToken, retailerApiBaseUrl } from '@/lib/auth';

export function SessionPackChip() {
  return <GenericSessionPackChip baseUrl={retailerApiBaseUrl()} token={readToken()} />;
}
