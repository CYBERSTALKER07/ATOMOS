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
  return (
    <PortalSection title="Pricing impact preview (read-only)">
      <p className="mb-3 text-sm text-muted">
        Preview how a proposed retailer price would compare to catalog list price for SKUs touching this depot. Does not create overrides.
      </p>
      <div className="grid gap-4 md:grid-cols-3">
        <PortalField id="pricingProductId" label="Product / SKU ID">
          <PortalInput value={productId} onChange={(e) => setProductId(e.target.value)} />
        </PortalField>
        <PortalField id="pricingRetailerId" label="Retailer ID (optional)">
          <PortalInput value={retailerId} onChange={(e) => setRetailerId(e.target.value)} />
        </PortalField>
        <PortalField id="pricingProposedPrice" label="Proposed price (minor units)">
          <PortalInput value={proposedPrice} onChange={(e) => setProposedPrice(e.target.value)} inputMode="numeric" />
        </PortalField>
      </div>
      {previewLoading ? <p className="text-sm text-muted">Loading preview…</p> : null}
      {preview ? (
        <div className="mt-3 grid gap-2 rounded-md border p-4 text-sm md:grid-cols-2">
          <div>Retailers on SKU: {preview.retailers_on_sku_count}</div>
          <div>Active overrides: {preview.active_override_count}</div>
          <div>Catalog list price: {preview.catalog_list_price}</div>
          <div>Margin delta / unit: {preview.margin_delta_per_unit}</div>
          <div className="md:col-span-2 text-muted">{preview.margin_estimate_label}</div>
          {preview.read_only ? (
            <div className="md:col-span-2 font-medium">Read-only — contact supplier to apply overrides.</div>
          ) : null}
        </div>
      ) : null}
    </PortalSection>
  );
}
