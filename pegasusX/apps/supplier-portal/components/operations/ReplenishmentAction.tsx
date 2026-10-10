"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';
import { PageSection } from '@/components/PageSection';

interface ReplenishmentActionProps {
  replenishing: boolean;
  onReplenishment: () => void;
}

export function ReplenishmentAction({ replenishing, onReplenishment }: ReplenishmentActionProps) {
  const t = usePortalT();
  return (
    <PageSection
      title={t("portal.nav.replenishment")}
      description={t("supplier_portal.residual.text.opens_a_warehouse_supply_request_against_your_primary_active_war")}
    >
      <button type="button" className="md-btn md-btn-tonal" onClick={onReplenishment} disabled={replenishing}>
        {replenishing ? "Triggering…" : "Trigger replenishment"}
      </button>
    </PageSection>
  );
}
