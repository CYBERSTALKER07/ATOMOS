"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import {
  supplierRetailerPriceOverrideCreateKey,
  supplierRetailerPriceOverrideDeleteKey,
} from "@pegasusx/api-client";
import { supplierFetch } from "@/lib/auth";
import type { CreateRetailerPriceOverrideRequest, RetailerOverridePreview, RetailerPriceOverride } from "@pegasusx/types";
import { PageChrome } from '@/components/PageChrome';

const api = createSupplierApi();

type CatalogProductOption = {
  product_id: string;
  name: string;
  price_minor: number;
};

type OverrideForm = {
  retailer_id: string;
  product_id: string;
  price: string;
  notes: string;
};

const EMPTY_FORM: OverrideForm = {
  retailer_id: "",
  product_id: "",
  price: "",
  notes: "",
};

function formatPrice(value: number): string {
  return new Intl.NumberFormat("en-US").format(value);
}

export default function RetailerOverridesPage() {
  const [overrides, setOverrides] = useState<RetailerPriceOverride[]>([]);
  const [products, setProducts] = useState<CatalogProductOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshTick, setRefreshTick] = useState(0);
  useSupplierSessionReconcile(() => setRefreshTick(t => t + 1));
  const [error, setError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState<OverrideForm>(EMPTY_FORM);
  const [preview, setPreview] = useState<RetailerOverridePreview | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [filterRetailer, setFilterRetailer] = useState("");
  const [filterProduct, setFilterProduct] = useState("");

  const loadOverrides = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params: { retailer_id?: string; product_id?: string } = {};
      if (filterRetailer.trim()) params.retailer_id = filterRetailer.trim();
      if (filterProduct.trim()) params.product_id = filterProduct.trim();
      const res = await api.listRetailerPriceOverrides(params);
      setOverrides(res.overrides ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load overrides");
    } finally {
      setLoading(false);
    }
  }, [filterRetailer, filterProduct]);

  const loadProducts = useCallback(async () => {
    try {
      const res = await supplierFetch("/v1/catalog/products");
      if (!res.ok) return;
      const rows = (await res.json()) as CatalogProductOption[];
      setProducts(Array.isArray(rows) ? rows : []);
    } catch {
      /* product list is optional for the form */
    }
  }, []);

  useEffect(() => {
    void loadOverrides();
  }, [loadOverrides]);

  useEffect(() => {
    void loadProducts();
  }, [loadProducts]);

  useEffect(() => {
    const retailerId = form.retailer_id.trim();
    const productId = form.product_id.trim();
    const price = Number.parseInt(form.price, 10);
    if (!productId || !Number.isFinite(price) || price <= 0) {
      setPreview(null);
      return;
    }
    const timer = window.setTimeout(() => {
      setPreviewLoading(true);
      void api
        .previewRetailerPriceOverride({
          retailer_id: retailerId || undefined,
          product_id: productId,
          proposed_price: price,
        })
        .then(setPreview)
        .catch(() => setPreview(null))
        .finally(() => setPreviewLoading(false));
    }, 400);
    return () => window.clearTimeout(timer);
  }, [form.retailer_id, form.product_id, form.price]);

  const productNameById = useMemo(() => {
    const map = new Map<string, string>();
    for (const product of products) {
      map.set(product.product_id, product.name);
    }
    return map;
  }, [products]);

  const uniqueRetailers = useMemo(
    () => new Set(overrides.map((row) => row.retailer_id)).size,
    [overrides],
  );

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    const retailerId = form.retailer_id.trim();
    const productId = form.product_id.trim();
    const price = Number.parseInt(form.price, 10);
    if (!retailerId || !productId || !form.price.trim()) {
      setFormError("Retailer ID, product ID, and price are required.");
      return;
    }
    if (!Number.isFinite(price) || price <= 0) {
      setFormError("Price must be a positive integer.");
      return;
    }

    const payload: CreateRetailerPriceOverrideRequest = {
      retailer_id: retailerId,
      product_id: productId,
      price,
      notes: form.notes.trim() || undefined,
    };

    setSaving(true);
    try {
      await api.createRetailerPriceOverride(
        payload,
        supplierRetailerPriceOverrideCreateKey(supplierScopeId(), retailerId, productId, price),
      );
      setForm(EMPTY_FORM);
      setShowCreate(false);
      await loadOverrides();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to create override");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(overrideId: string) {
    if (!confirm("Remove this retailer price override?")) return;
    setDeletingId(overrideId);
    setError(null);
    try {
      await api.deleteRetailerPriceOverride(
        overrideId,
        supplierRetailerPriceOverrideDeleteKey(supplierScopeId(), overrideId),
      );
      await loadOverrides();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete override");
    } finally {
      setDeletingId(null);
    }
  }

  return (
    <PageChrome
      icon="pricing"
      title="Retailer overrides"
      description="Set custom prices per retailer and product SKU."
      loading={loading}
      error={error}
      empty={!showCreate && overrides.length === 0}
      emptyMessage="No retailer price overrides yet."
      emptyIcon="pricing"
      actions={
        <button
          type="button"
          className="md-btn md-btn-filled md-typescale-label-large px-4 py-2"
          onClick={() => {
            setShowCreate((value) => !value);
            setFormError(null);
          }}
        >
          {showCreate ? "Close" : "New override"}
        </button>
      }
    >
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <Metric label="Active overrides" value={String(overrides.length)} />
        <Metric label="Retailers" value={String(uniqueRetailers)} />
        <Metric label="Products" value={String(new Set(overrides.map((row) => row.product_id)).size)} />
        <Metric
          label="Catalog SKUs"
          value={products.length > 0 ? String(products.length) : "—"}
        />
      </div>

      <div className="md-card p-4 mb-6 flex flex-wrap gap-3 items-end">
        <Field label="Filter retailer ID">
          <input
            className="md-input-outlined w-full min-w-[12rem] px-3 py-2"
            value={filterRetailer}
            onChange={(event) => setFilterRetailer(event.target.value)}
            placeholder="Retailer UUID"
          />
        </Field>
        <Field label="Filter product ID">
          <select
            className="md-input-outlined w-full min-w-[12rem] px-3 py-2"
            value={filterProduct}
            onChange={(event) => setFilterProduct(event.target.value)}
          >
            <option value="">All products</option>
            {products.map((product) => (
              <option key={product.product_id} value={product.product_id}>
                {product.name}
              </option>
            ))}
          </select>
        </Field>
        <button
          type="button"
          className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2"
          onClick={() => void loadOverrides()}
        >
          Apply filters
        </button>
      </div>

      {showCreate ? (
        <form className="md-card p-6 mb-6 space-y-4" onSubmit={handleCreate}>
          <h2 className="md-typescale-title-medium font-semibold">New price override</h2>
          {formError ? (
            <p className="md-typescale-body-small" style={{ color: "var(--color-md-error)" }}>
              {formError}
            </p>
          ) : null}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Field label="Retailer ID">
              <input
                className="md-input-outlined w-full px-3 py-2"
                value={form.retailer_id}
                onChange={(event) => setForm((prev) => ({ ...prev, retailer_id: event.target.value }))}
                placeholder="Retailer UUID"
                required
              />
            </Field>
            <Field label="Product ID (SKU)">
              {products.length > 0 ? (
                <select
                  className="md-input-outlined w-full px-3 py-2"
                  value={form.product_id}
                  onChange={(event) => setForm((prev) => ({ ...prev, product_id: event.target.value }))}
                  required
                >
                  <option value="">Select a product</option>
                  {products.map((product) => (
                    <option key={product.product_id} value={product.product_id}>
                      {product.name}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  className="md-input-outlined w-full px-3 py-2 font-mono"
                  value={form.product_id}
                  onChange={(event) => setForm((prev) => ({ ...prev, product_id: event.target.value }))}
                  placeholder="Product UUID"
                  required
                />
              )}
            </Field>
            <Field label="Override price (minor units)">
              <input
                type="number"
                min="1"
                className="md-input-outlined w-full px-3 py-2 font-mono"
                value={form.price}
                onChange={(event) => setForm((prev) => ({ ...prev, price: event.target.value }))}
                required
              />
            </Field>
            <Field label="Notes">
              <input
                className="md-input-outlined w-full px-3 py-2"
                value={form.notes}
                onChange={(event) => setForm((prev) => ({ ...prev, notes: event.target.value }))}
                placeholder="Reason for override"
              />
            </Field>
          </div>
          {previewLoading ? (
            <p className="text-sm opacity-60">Calculating impact preview…</p>
          ) : preview ? (
            <div className="rounded-lg border p-4 space-y-2" style={{ borderColor: "var(--color-md-outline-variant)" }}>
              <p className="md-typescale-label-large font-semibold">Impact preview</p>
              <p className="text-sm">Retailers on SKU: {preview.retailers_on_sku_count}</p>
              <p className="text-sm">Active overrides: {preview.active_override_count}</p>
              <p className="text-sm">Catalog list price: {formatPrice(preview.catalog_list_price)}</p>
              <p className="text-sm">
                Margin delta / unit: {formatPrice(preview.margin_delta_per_unit)} ({preview.margin_estimate_label})
              </p>
            </div>
          ) : null}
          <div className="flex flex-wrap gap-3">
            <button
              type="submit"
              className="md-btn md-btn-filled md-typescale-label-large px-6 py-2"
              disabled={saving}
            >
              {saving ? "Creating…" : "Create override"}
            </button>
            <button
              type="button"
              className="md-btn md-btn-outlined md-typescale-label-large px-6 py-2"
              onClick={() => {
                setShowCreate(false);
                setForm(EMPTY_FORM);
                setFormError(null);
              }}
            >
              Cancel
            </button>
          </div>
        </form>
      ) : null}

      {overrides.length > 0 ? (
        <div className="md-card overflow-x-auto">
          <table className="min-w-full text-left">
            <thead className="border-b border-[var(--color-md-outline-variant)]">
              <tr>
                <th className="px-4 py-3 md-typescale-label-large">Product</th>
                <th className="px-4 py-3 md-typescale-label-large">Retailer</th>
                <th className="px-4 py-3 md-typescale-label-large text-right">Price</th>
                <th className="px-4 py-3 md-typescale-label-large">Set by</th>
                <th className="px-4 py-3 md-typescale-label-large">Notes</th>
                <th className="px-4 py-3 md-typescale-label-large text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {overrides.map((row) => (
                <tr key={row.override_id} className="border-b border-[var(--color-md-outline-variant)]">
                  <td className="px-4 py-3">
                    <div className="font-medium">
                      {productNameById.get(row.product_id) ?? row.product_id.slice(0, 12)}
                    </div>
                    <div className="font-mono text-xs text-[var(--color-md-outline)]">{row.product_id}</div>
                  </td>
                  <td className="px-4 py-3 font-mono text-sm">{row.retailer_id}</td>
                  <td className="px-4 py-3 text-right font-mono text-sm">{formatPrice(row.price)}</td>
                  <td className="px-4 py-3">
                    <span className="md-chip">{row.set_by_role || row.set_by || "—"}</span>
                  </td>
                  <td className="px-4 py-3 text-sm text-[var(--color-md-outline)]">{row.notes || "—"}</td>
                  <td className="px-4 py-3 text-right">
                    <button
                      type="button"
                      className="md-btn md-btn-outlined md-typescale-label-medium px-3 py-1 disabled:opacity-50"
                      disabled={deletingId === row.override_id}
                      onClick={() => void handleDelete(row.override_id)}
                    >
                      {deletingId === row.override_id ? "Removing…" : "Remove"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
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
    <label className="block space-y-1">
      <span className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</span>
      {children}
    </label>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="md-card p-4">
      <div className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</div>
      <div className="md-typescale-headline-small font-semibold mt-1">{value}</div>
    </div>
  );
}
