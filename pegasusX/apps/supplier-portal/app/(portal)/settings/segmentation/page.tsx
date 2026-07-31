"use client";

import { PageChrome } from "@/components/PageChrome";
import { SegmentationPanel } from "@/components/settings/SegmentationPanel";

export default function SegmentationSettingsPage() {
  return (
    <PageChrome
      title="Segmentation"
      subtitle="Retailer segments and SKU velocity classes for constrained allocation (O9-1)."
    >
      <SegmentationPanel />
    </PageChrome>
  );
}
