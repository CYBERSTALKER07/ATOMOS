"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { normalizeEanBarcode } from "@pegasusx/validation";
import { PageChrome } from '@/components/PageChrome';

type CatalogProduct = {
  product_id: string;
  name: string;
  category_id: string;
  price_minor: number;
  currency: string;
  unit: string;
  unit_volume_vu: number;
  barcode?: string;
  is_active: boolean;
  version: number;
};

type ProductDraft = {
  name: string;
  barcode: string;
  unit_volume_vu: string;
  price_minor: string;
};

function draftFromProduct(product: CatalogProduct): ProductDraft {
  return {
    name: product.name,
    barcode: product.barcode ?? "",
    unit_volume_vu: String(product.unit_volume_vu ?? 1),
    price_minor: String(product.price_minor),
  };
}

export default function CatalogProductDetailPage() {
  const params = useParams<{ productId: string }>();
  const productId = decodeURIComponent(params.productId);
  const [product, setProduct] = useState<CatalogProduct | null>(null);
  const [draft, setDraft] = useState<ProductDraft | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await supplierFetch(`/v1/catalog/products/${encodeURIComponent(productId)}`);
      if (!res.ok) throw new Error(`product ${res.status}`);
      const row = (await res.json()) as CatalogProduct;
      setProduct(row);
      setDraft(draftFromProduct(row));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load product");
      setProduct(null);
      setDraft(null);
    } finally {
      setLoading(false);
    }
  }, [productId]);

  useEffect(() => {
    void load();
  }, [load]);

  const dirty =
    product &&
    draft &&
    (draft.name.trim() !== product.name ||
      draft.barcode.trim() !== (product.barcode ?? "").trim() ||
      draft.unit_volume_vu !== String(product.unit_volume_vu ?? 1) ||
      draft.price_minor !== String(product.price_minor));

  async function saveProduct() {
    if (!product || !draft) return;

    const name = draft.name.trim();
    if (!name) {
      setSaveError("Product name is required.");
      return;
    }

    const priceMinor = Number.parseInt(draft.price_minor, 10);
    if (!Number.isFinite(priceMinor) || priceMinor < 0) {
      setSaveError("Price must be a non-negative integer (minor units).");
      return;
    }

    const parsedVu = Number.parseFloat(draft.unit_volume_vu);
    if (!Number.isFinite(parsedVu) || parsedVu <= 0) {
      setSaveError("Unit volume must be a positive number.");
      return;
    }

    const trimmedBarcode = draft.barcode.trim();
    let normalizedBarcode: string | undefined;
    if (trimmedBarcode) {
      const result = normalizeEanBarcode(trimmedBarcode);
      if (!result.ok) {
        setSaveError(
          result.error === "invalid_barcode_checksum"
            ? "Barcode checksum is invalid."
            : "Barcode must be 8–14 digits (EAN/GTIN).",
        );
        return;
      }
      normalizedBarcode = result.code;
    }

    setSaving(true);
    setSaveError(null);
    try {
      const res = await supplierFetch(`/v1/catalog/products/${encodeURIComponent(product.product_id)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name,
          price_minor: priceMinor,
          currency: product.currency,
          unit: product.unit,
          unit_volume_vu: parsedVu,
          barcode: normalizedBarcode ?? "",
          is_active: product.is_active,
          version: product.version,
        }),
      });
      if (!res.ok) throw new Error(`update ${res.status}`);
      const updated = (await res.json()) as CatalogProduct;
      setProduct(updated);
      setDraft(draftFromProduct(updated));
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Failed to save product");
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageChrome
      icon="catalog"
      title="Product detail"
      description={productId}
      loading={loading}
      error={error}
      empty={!product || !draft}
      actions={
        <Link href="/catalog" className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2">
          Back to catalog
        </Link>
      }
    >
      {product && draft ? (
        <div className="space-y-6">
          <div className="md-card p-6 grid grid-cols-1 md:grid-cols-2 gap-4">
            <Field label="Product ID">
              <div className="font-mono text-sm mt-1">{product.product_id}</div>
            </Field>
            <Field label="Category">
              <div className="font-mono text-sm mt-1">{product.category_id}</div>
            </Field>
            <Field label="Name">
              <input
                className="md-input-outlined mt-1 w-full px-3 py-2"
                value={draft.name}
                onChange={(event) => setDraft((prev) => (prev ? { ...prev, name: event.target.value } : prev))}
              />
            </Field>
            <Field label="Barcode">
              <input
                className="md-input-outlined mt-1 w-full px-3 py-2 font-mono"
                value={draft.barcode}
                onChange={(event) => setDraft((prev) => (prev ? { ...prev, barcode: event.target.value } : prev))}
                placeholder="EAN / GTIN"
              />
            </Field>
            <Field label={`Price (${product.currency}, minor units)`}>
              <input
                type="number"
                min="0"
                step="1"
                className="md-input-outlined mt-1 w-full px-3 py-2 font-mono"
                value={draft.price_minor}
                onChange={(event) =>
                  setDraft((prev) => (prev ? { ...prev, price_minor: event.target.value } : prev))
                }
              />
            </Field>
            <Field label="Unit volume (VU)">
              <input
                type="number"
                min="0.1"
                step="0.1"
                className="md-input-outlined mt-1 w-full px-3 py-2 font-mono"
                value={draft.unit_volume_vu}
                onChange={(event) =>
                  setDraft((prev) => (prev ? { ...prev, unit_volume_vu: event.target.value } : prev))
                }
              />
            </Field>
            <Field label="Status">
              <div className="mt-1">{product.is_active ? "Active" : "Inactive"}</div>
            </Field>
            <Field label="Version">
              <div className="mt-1 font-mono text-sm">{product.version}</div>
            </Field>
          </div>

          {saveError ? (
            <p className="md-typescale-body-small" style={{ color: "var(--color-md-error)" }}>
              {saveError}
            </p>
          ) : null}

          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              className="md-btn md-btn-filled md-typescale-label-large px-6 py-2 disabled:opacity-50"
              disabled={!dirty || saving}
              onClick={() => void saveProduct()}
            >
              {saving ? "Saving…" : "Save changes"}
            </button>
            <button
              type="button"
              className="md-btn md-btn-outlined md-typescale-label-large px-6 py-2 disabled:opacity-50"
              disabled={!dirty || saving}
              onClick={() => setDraft(draftFromProduct(product))}
            >
              Reset
            </button>
          </div>
        </div>
      ) : null}
    </PageChrome>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <div className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</div>
      {children}
    </label>
  );
}
