"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import type { CatalogProduct, SupplierPromotion } from "@pegasusx/types";
import { supplierFetch } from "@/lib/auth";
import { supplierScopeId } from "@/lib/supplier-scope";
import {
  supplierPromotionCreateKey,
  supplierPromotionDeactivateKey,
  supplierPromotionUpdateKey,
} from "@pegasusx/api-client";
import { PageChrome } from '@/components/PageChrome';

export default function ProductPricingPage() {
  const params = useParams<{ productId: string }>();
  const productId = params.productId;
  const [product, setProduct] = useState<CatalogProduct | null>(null);
  const [activeSale, setActiveSale] = useState<SupplierPromotion | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [priceMajor, setPriceMajor] = useState("");
  const [saleEnabled, setSaleEnabled] = useState(false);
  const [saleDiscountBps, setSaleDiscountBps] = useState("");
  const [showPreview, setShowPreview] = useState(false);

  useEffect(() => {
    if (!productId) return;
    setLoading(true);
    setError(null);
    Promise.all([
      supplierFetch(`/v1/catalog/products/${encodeURIComponent(productId)}`),
      supplierFetch("/v1/supplier/promotions"),
    ])
      .then(async ([productRes, promoRes]) => {
        if (!productRes.ok) throw new Error(`product_failed_${productRes.status}`);
        const loaded = (await productRes.json()) as CatalogProduct;
        setProduct(loaded);
        setPriceMajor((loaded.price_minor / 100).toFixed(2));

        if (promoRes.ok) {
          const body = (await promoRes.json()) as { promotions?: SupplierPromotion[] };
          const promo = (body.promotions ?? []).find(
            (row) => row.is_active && row.scope_type === "PRODUCT" && row.scope_product_id === productId,
          );
          setActiveSale(promo ?? null);
          if (promo) {
            setSaleEnabled(true);
            setSaleDiscountBps(String(promo.discount_bps));
          }
        }
      })
      .catch((err) => setError(err instanceof Error ? err.message : "load_failed"))
      .finally(() => setLoading(false));
  }, [productId]);

  const save = async () => {
    if (!product) return;
    const parsed = Number.parseFloat(priceMajor.replace(",", "."));
    if (!Number.isFinite(parsed) || parsed < 0) {
      setError("Enter a valid list price.");
      return;
    }
    if (saleEnabled) {
      const bps = Number.parseInt(saleDiscountBps, 10);
      if (!Number.isFinite(bps) || bps <= 0) {
        setError("Sale discount must be greater than zero.");
        return;
      }
    }
    setShowPreview(true);
  };

  const commitSave = async () => {
    if (!product) return;
    const parsed = Number.parseFloat(priceMajor.replace(",", "."));
    
    setSaving(true);
    setShowPreview(false);
    setError(null);
    try {
      const priceMinor = Math.round(parsed * 100);
      const updateRes = await supplierFetch(`/v1/catalog/products/${encodeURIComponent(product.product_id)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: product.name,
          price_minor: priceMinor,
          currency: product.currency,
          unit: product.unit,
          unit_volume_vu: product.unit_volume_vu ?? 1,
          image_url: product.image_url,
          barcode: product.barcode,
          is_active: product.is_active,
          version: product.version,
        }),
      });
      if (!updateRes.ok) throw new Error(`update_failed_${updateRes.status}`);

      if (saleEnabled) {
        const bps = Number.parseInt(saleDiscountBps, 10);
        if (!Number.isFinite(bps) || bps <= 0) throw new Error("Sale discount must be greater than zero.");
        const promoBody = {
          name: `Sale · ${product.name}`,
          description: "Product sale pricing",
          discount_bps: bps,
          scope_type: "PRODUCT",
          scope_product_id: product.product_id,
          retailer_scope: "ALL",
        };
        const promoFingerprint = JSON.stringify(promoBody);
        const scope = supplierScopeId();
        const promoRes = activeSale
          ? await supplierFetch(`/v1/supplier/promotions/${encodeURIComponent(activeSale.promotion_id)}`, {
              method: "PATCH",
              headers: {
                "Content-Type": "application/json",
                "Idempotency-Key": supplierPromotionUpdateKey(
                  scope,
                  activeSale.promotion_id,
                  promoFingerprint,
                ),
              },
              body: promoFingerprint,
            })
          : await supplierFetch("/v1/supplier/promotions", {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
                "Idempotency-Key": supplierPromotionCreateKey(scope, promoFingerprint),
              },
              body: promoFingerprint,
            });
        if (!promoRes.ok) throw new Error(`promotion_failed_${promoRes.status}`);
      } else if (activeSale) {
        await supplierFetch(`/v1/supplier/promotions/${encodeURIComponent(activeSale.promotion_id)}`, {
          method: "DELETE",
          headers: {
            "Idempotency-Key": supplierPromotionDeactivateKey(supplierScopeId(), activeSale.promotion_id),
          },
        });
      }
      setProduct({ ...product, price_minor: priceMinor, version: product.version + 1 });
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_failed");
    } finally {
      setSaving(false);
    }
  };

  return (
    <PageChrome
      icon="pricing"
      title={product?.name ?? "Product pricing"}
      description="List price and optional product-scoped sale discount."
      loading={loading}
      error={error}
      empty={!product && !loading}
      emptyMessage="Product not found."
    >
      {product ? (
        <div className="md-card p-6 space-y-6 max-w-xl">
          <label className="block space-y-1">
            <span className="md-typescale-label-medium">List price ({product.currency})</span>
            <input className="md-input w-full" value={priceMajor} onChange={(e) => setPriceMajor(e.target.value)} />
          </label>
          <label className="flex items-center gap-3">
            <input type="checkbox" checked={saleEnabled} onChange={(e) => setSaleEnabled(e.target.checked)} />
            <span className="md-typescale-body-medium">On sale</span>
          </label>
          {saleEnabled ? (
            <label className="block space-y-1">
              <span className="md-typescale-label-medium">Discount (bps)</span>
              <input className="md-input w-full" value={saleDiscountBps} onChange={(e) => setSaleDiscountBps(e.target.value)} />
              <span className="text-sm text-[var(--color-md-outline)]">100 bps = 1% off list price.</span>
            </label>
          ) : null}
          <button type="button" className="md-btn md-btn-filled px-4 py-2" disabled={saving} onClick={() => void save()}>
            {saving ? "Saving…" : "Preview impact"}
          </button>
        </div>
      ) : null}
      
      {showPreview && product && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-[var(--surface)] w-full max-w-md rounded-xl shadow-xl overflow-hidden flex flex-col">
            <div className="px-6 py-4 border-b border-[var(--border)]">
              <h2 className="text-xl font-semibold">Preview Pricing Impact</h2>
            </div>
            
            <div className="p-6 space-y-4">
              {(() => {
                const oldList = product.price_minor;
                const oldSale = activeSale ? oldList * (1 - activeSale.discount_bps / 10000) : oldList;
                
                const parsedList = Number.parseFloat(priceMajor.replace(",", "."));
                const newList = Math.round(parsedList * 100);
                const parsedBps = Number.parseInt(saleDiscountBps, 10);
                const newSale = saleEnabled && Number.isFinite(parsedBps) ? newList * (1 - parsedBps / 10000) : newList;
                
                const formatMinor = (m: number) => (m / 100).toFixed(2) + " " + product.currency;
                
                return (
                  <div className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div className="p-3 bg-[var(--field-background)] rounded-lg">
                        <div className="text-sm text-[var(--muted)]">Old List Price</div>
                        <div className="font-mono mt-1">{formatMinor(oldList)}</div>
                      </div>
                      <div className="p-3 bg-[var(--field-background)] rounded-lg">
                        <div className="text-sm text-[var(--muted)]">New List Price</div>
                        <div className="font-mono mt-1">{formatMinor(newList)}</div>
                      </div>
                    </div>
                    
                    <div className="grid grid-cols-2 gap-4">
                      <div className="p-3 bg-[var(--field-background)] rounded-lg">
                        <div className="text-sm text-[var(--muted)]">Old Sale Price</div>
                        <div className="font-mono mt-1">{formatMinor(oldSale)}</div>
                      </div>
                      <div className="p-3 bg-[var(--field-background)] rounded-lg">
                        <div className="text-sm text-[var(--muted)]">New Sale Price</div>
                        <div className="font-mono mt-1 font-bold text-[var(--primary)]">{formatMinor(newSale)}</div>
                      </div>
                    </div>
                    
                    <div className="pt-2 border-t border-[var(--border)] text-sm">
                      Effective immediately upon commit.
                    </div>
                  </div>
                );
              })()}
            </div>
            
            <div className="px-6 py-4 border-t border-[var(--border)] flex justify-end gap-3 bg-[var(--field-background)]">
              <button type="button" onClick={() => setShowPreview(false)} className="md-btn md-btn-outlined px-4 py-2">
                Cancel
              </button>
              <button type="button" onClick={() => void commitSave()} className="md-btn md-btn-filled px-4 py-2">
                Commit rule
              </button>
            </div>
          </div>
        </div>
      )}
    </PageChrome>
  );
}
