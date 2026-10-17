"use client";

import { usePortalT } from "@/lib/i18n";
import type { RetailerOverridePreview } from '@pegasusx/types';
import { PortalField, PortalInput, PortalSection } from '@/components/portal';

interface OperationsPricingPreviewProps {
  productId: string;
  setProductId: (val: string) => void;
  retailerId: string;
  setRetailerId: (val: string) => void;
  proposedPrice: string;
  setProposedPrice: (val: string) => void;
  previewLoading: boolean;
  preview: RetailerOverridePreview | null;
}

export function OperationsPricingPreview({
  productId,
  setProductId,
  retailerId,
  setRetailerId,
  proposedPrice,
  setProposedPrice,
  previewLoading,
  preview,
}: OperationsPricingPreviewProps) {
  const t = usePortalT();
  return (
    <PortalSection title={t("warehouse_portal.operations.operations_pricing_preview.text.pricing_impact_preview_read_only")}>
      <p className="mb-3 text-sm text-muted">
        Preview how a proposed retailer price would compare to catalog list price for SKUs touching this depot. Does not create overrides.
      </p>
      <div className="grid gap-4 md:grid-cols-3">
        <PortalField id="pricingProductId" label={t("warehouse_portal.residual.text.product_sku_id")}>
          <PortalInput value={productId} onChange={(e) => setProductId(e.target.value)} />
        </PortalField>
        <PortalField id="pricingRetailerId" label={t("warehouse_portal.residual.text.retailer_id_optional")}>
          <PortalInput value={retailerId} onChange={(e) => setRetailerId(e.target.value)} />
        </PortalField>
        <PortalField id="pricingProposedPrice" label={t("warehouse_portal.residual.text.proposed_price_minor_units")}>
          <PortalInput value={proposedPrice} onChange={(e) => setProposedPrice(e.target.value)} inputMode="numeric" />
        </PortalField>
      </div>
      {previewLoading ? <p className="text-sm text-muted">{t("warehouse_portal.operations.operations_pricing_preview.text.loading_preview")}</p> : null}
      {preview ? (
        <div className="mt-3 grid gap-2 rounded-md border p-4 text-sm md:grid-cols-2">
          <div>Retailers on SKU: {preview.retailers_on_sku_count}</div>
          <div>Active overrides: {preview.active_override_count}</div>
          <div>Catalog list price: {preview.catalog_list_price}</div>
          <div>Margin delta / unit: {preview.margin_delta_per_unit}</div>
          <div className="md:col-span-2 text-muted">{preview.margin_estimate_label}</div>
          {preview.read_only ? (
            <div className="md:col-span-2 font-medium">{t("warehouse_portal.operations.operations_pricing_preview.text.read_only_contact_supplier_to_apply_overrides")}</div>
          ) : null}
        </div>
      ) : null}
    </PortalSection>
  );
}
