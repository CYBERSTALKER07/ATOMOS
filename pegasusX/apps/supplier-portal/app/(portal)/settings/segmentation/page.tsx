"use client";

import { usePortalT } from "@/lib/i18n";
import { PageChrome } from "@/components/PageChrome";
import { SegmentationPanel } from "@/components/settings/SegmentationPanel";

export default function SegmentationSettingsPage() {
  const t = usePortalT();
  return (
    <PageChrome
      title={t("portal.nav.segmentation")}
      description={t("supplier_portal.residual.text.retailer_segments_and_sku_velocity_classes_for_constrained_alloc")}
    >
      <SegmentationPanel />
    </PageChrome>
  );
}
