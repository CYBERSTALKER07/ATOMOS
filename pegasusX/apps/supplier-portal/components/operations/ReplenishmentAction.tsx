import React from 'react';
import { PageSection } from '@/components/PageSection';

interface ReplenishmentActionProps {
  replenishing: boolean;
  onReplenishment: () => void;
}

export function ReplenishmentAction({ replenishing, onReplenishment }: ReplenishmentActionProps) {
  return (
    <PageSection
      title="Replenishment"
      description="Opens a warehouse supply request against your primary active warehouse."
    >
      <button type="button" className="md-btn md-btn-tonal" onClick={onReplenishment} disabled={replenishing}>
        {replenishing ? "Triggering…" : "Trigger replenishment"}
      </button>
    </PageSection>
  );
}
