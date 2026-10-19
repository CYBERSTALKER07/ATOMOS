'use client';

import { useMarketPack } from '@pegasusx/api-react';
import { PackChip } from '../pack';
import { fiscalReceiptLabel } from '@pegasusx/api-core';

export function SessionPackChip({ baseUrl, token }: { baseUrl: string; token: string }) {
  const { pack, session } = useMarketPack({ baseUrl, token });
  return (
    <PackChip
      currency={pack?.currency_code}
      receipts={fiscalReceiptLabel(pack?.fiscal_adapter)}
      title={[
        pack?.timezone,
        pack?.maps_adapter,
        session?.checkout_reads_this === false ? 'checkout_reads_this=false' : ''
      ]
        .filter(Boolean)
        .join(' · ')}
    />
  );
}
