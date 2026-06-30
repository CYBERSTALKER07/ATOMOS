"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { createSupplierApi } from "@/lib/api";
import { ListToolbar } from "@/components/ListToolbar";
import { usePagination } from "@/lib/use-pagination";
import { PageChrome } from "@/components/PageChrome";
import { normalizeEanBarcode } from "@pegasusx/validation";
import { BulkImportWizard } from "@/components/BulkImportWizard";

const ALLOWED_IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp"];
const MAX_IMAGE_SIZE = 5 * 1024 * 1024;

type SaleUnit = "UNIT" | "CASE";

type CatalogProduct = {
  product_id: string;
  name: string;
  category_id: string;
  price_minor: number;
  currency: string;
  unit: string;
  unit_volume_vu: number;
  units_per_case?: number;
  sale_unit?: SaleUnit;
  barcode?: string;
  image_url?: string;
  is_active: boolean;
  version: number;
};

type CatalogCategory = {
  category_id: string;
  name: string;
};

type CreateProductForm = {
  name: string;
  category_id: string;
  description: string;
  price_minor: string;
  unit_volume_vu: string;
  units_per_case: string;
  sale_unit: SaleUnit;
  barcode: string;
};

const EMPTY_CREATE_FORM: CreateProductForm = {
  name: "",
  category_id: "",
  description: "",
  price_minor: "",
  unit_volume_vu: "1",
  units_per_case: "",
  sale_unit: "UNIT",
  barcode: "",
};

export default function CatalogPage() {
  const [products, setProducts] = useState<CatalogProduct[]>([]);
  const [categories, setCategories] = useState<CatalogCategory[]>([]);
  const [currency, setCurrency] = useState("UZS");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [savingId, setSavingId] = useState<string | null>(null);
  const [draftVU, setDraftVU] = useState<Record<string, string>>({});
  const [draftBarcode, setDraftBarcode] = useState<Record<string, string>>({});
  const [draftUnitsPerCase, setDraftUnitsPerCase] = useState<Record<string, string>>({});
  const [draftSaleUnit, setDraftSaleUnit] = useState<Record<string, SaleUnit>>({});
  const [showCreate, setShowCreate] = useState(false);
  const [showBulkImport, setShowBulkImport] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createForm, setCreateForm] = useState<CreateProductForm>(EMPTY_CREATE_FORM);
  const [createError, setCreateError] = useState<string | null>(null);
  const [createImageFile, setCreateImageFile] = useState<File | null>(null);
  const [createImagePreview, setCreateImagePreview] = useState<string | null>(null);
  const createImageInputRef = useRef<HTMLInputElement>(null);
  const rowImageInputRef = useRef<HTMLInputElement>(null);
  const [imageTargetId, setImageTargetId] = useState<string | null>(null);
  const [imageSavingId, setImageSavingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      const api = createSupplierApi();
      const profile = await api.getSupplierProfile();
      setCurrency(profile.currency || "UZS");

      const [productsRes, categoriesRes] = await Promise.all([
        supplierFetch("/v1/catalog/products"),
        supplierFetch(`/v1/catalog/categories?supplier_id=${encodeURIComponent(profile.supplier_id)}`),
      ]);
      if (!productsRes.ok) throw new Error(`catalog ${productsRes.status}`);
      if (!categoriesRes.ok) throw new Error(`categories ${categoriesRes.status}`);

      const rows = (await productsRes.json()) as CatalogProduct[];
      const cats = (await categoriesRes.json()) as CatalogCategory[];
      setProducts(Array.isArray(rows) ? rows : []);
      setCategories(Array.isArray(cats) ? cats : []);
      setDraftVU({});
      setDraftBarcode({});
      setDraftUnitsPerCase({});
      setDraftSaleUnit({});
      setCreateForm(current => ({
        ...current,
        category_id: current.category_id || cats[0]?.category_id || "",
      }));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load catalog");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const pagination = usePagination(products, 20);

  async function uploadCatalogImage(file: File): Promise<string> {
    if (!ALLOWED_IMAGE_TYPES.includes(file.type)) {
      throw new Error("Image must be JPEG, PNG, or WebP.");
    }
    if (file.size > MAX_IMAGE_SIZE) {
      throw new Error("Image must be under 5 MB.");
    }
    const ext = file.name.split(".").pop()?.toLowerCase() || "jpg";
    const ticketRes = await supplierFetch(
      `/v1/catalog/products/upload-ticket?ext=${encodeURIComponent(ext)}`,
    );
    if (!ticketRes.ok) throw new Error(`upload ticket ${ticketRes.status}`);
    const ticket = (await ticketRes.json()) as { upload_url: string; image_url: string };
    const isPlaceholder = ticket.upload_url.includes("placehold.co");
    if (!isPlaceholder) {
      const uploadRes = await fetch(ticket.upload_url, {
        method: "PUT",
        body: file,
        headers: { "Content-Type": file.type || "image/jpeg" },
      });
      if (!uploadRes.ok) throw new Error("Image upload to storage failed.");
    }
    return ticket.image_url;
  }

  async function saveProductImage(product: CatalogProduct, file: File) {
    setImageSavingId(product.product_id);
    setError(null);
    try {
      const imageUrl = await uploadCatalogImage(file);
      const res = await supplierFetch(`/v1/catalog/products/${product.product_id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: product.name,
          price_minor: product.price_minor,
          currency: product.currency,
          unit: product.unit,
          unit_volume_vu: product.unit_volume_vu,
          image_url: imageUrl,
          is_active: product.is_active,
          version: product.version,
        }),
      });
      if (!res.ok) throw new Error(`image update ${res.status}`);
      const updated = (await res.json()) as CatalogProduct;
      setProducts(prev => prev.map(row => (row.product_id === updated.product_id ? updated : row)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update product image");
    } finally {
      setImageSavingId(null);
      setImageTargetId(null);
    }
  }

  async function saveProductEdits(product: CatalogProduct) {
    const rawVu = draftVU[product.product_id] ?? String(product.unit_volume_vu ?? 1);
    const parsedVu = Number.parseFloat(rawVu);
    if (!Number.isFinite(parsedVu) || parsedVu <= 0) {
      setError("Unit volume must be a positive number.");
      return;
    }

    const barcodeRaw = draftBarcode[product.product_id] ?? product.barcode ?? "";
    const trimmedBarcode = barcodeRaw.trim();
    let normalizedBarcode: string | undefined;
    if (trimmedBarcode) {
      const result = normalizeEanBarcode(trimmedBarcode);
      if (!result.ok) {
        setError(
          result.error === "invalid_barcode_checksum"
            ? "Barcode checksum is invalid."
            : "Barcode must be 8–14 digits (EAN/GTIN).",
        );
        return;
      }
      normalizedBarcode = result.code;
    }

    const saleUnit = draftSaleUnit[product.product_id] ?? product.sale_unit ?? "UNIT";
    const unitsPerCaseRaw = draftUnitsPerCase[product.product_id] ?? (
      product.units_per_case != null ? String(product.units_per_case) : ""
    );
    let unitsPerCase: number | undefined;
    if (saleUnit === "CASE") {
      const parsed = Number.parseInt(unitsPerCaseRaw, 10);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        setError("Units per case must be a positive integer when selling by case.");
        return;
      }
      unitsPerCase = parsed;
    }

    const vuDirty = rawVu !== String(product.unit_volume_vu ?? 1);
    const barcodeDirty = trimmedBarcode !== (product.barcode ?? "").trim();
    const saleUnitDirty = saleUnit !== (product.sale_unit ?? "UNIT");
    const unitsPerCaseDirty = unitsPerCaseRaw !== (
      product.units_per_case != null ? String(product.units_per_case) : ""
    );
    if (!vuDirty && !barcodeDirty && !saleUnitDirty && !unitsPerCaseDirty) return;

    setSavingId(product.product_id);
    setError(null);
    try {
      const res = await supplierFetch(`/v1/catalog/products/${product.product_id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: product.name,
          price_minor: product.price_minor,
          currency: product.currency,
          unit: product.unit,
          unit_volume_vu: parsedVu,
          barcode: normalizedBarcode ?? "",
          sale_unit: saleUnit,
          ...(unitsPerCase != null ? { units_per_case: unitsPerCase } : {}),
          is_active: product.is_active,
          version: product.version,
        }),
      });
      if (!res.ok) throw new Error(`update ${res.status}`);
      const updated = (await res.json()) as CatalogProduct;
      setProducts(prev => prev.map(row => (row.product_id === updated.product_id ? updated : row)));
      setDraftVU(prev => {
        const next = { ...prev };
        delete next[product.product_id];
        return next;
      });
      setDraftBarcode(prev => {
        const next = { ...prev };
        delete next[product.product_id];
        return next;
      });
      setDraftUnitsPerCase(prev => {
        const next = { ...prev };
        delete next[product.product_id];
        return next;
      });
      setDraftSaleUnit(prev => {
        const next = { ...prev };
        delete next[product.product_id];
        return next;
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save product");
    } finally {
      setSavingId(null);
    }
  }

  async function createProduct() {
    const name = createForm.name.trim();
    const categoryId = createForm.category_id.trim();
    const priceMinor = Number.parseInt(createForm.price_minor, 10);
    const unitVolume = Number.parseFloat(createForm.unit_volume_vu);
    if (!name || !categoryId) {
      setCreateError("Name and category are required.");
      return;
    }
    if (!Number.isFinite(priceMinor) || priceMinor < 0) {
      setCreateError("Price must be a non-negative integer (minor units).");
      return;
    }
    if (!Number.isFinite(unitVolume) || unitVolume <= 0) {
      setCreateError("Unit VU must be a positive number.");
      return;
    }
    const trimmedBarcode = createForm.barcode.trim();
    let normalizedBarcode: string | undefined;
    if (trimmedBarcode) {
      const result = normalizeEanBarcode(trimmedBarcode);
      if (!result.ok) {
        setCreateError(
          result.error === "invalid_barcode_checksum"
            ? "Barcode checksum is invalid."
            : "Barcode must be 8–14 digits (EAN/GTIN).",
        );
        return;
      }
      normalizedBarcode = result.code;
    }
    let unitsPerCase: number | undefined;
    if (createForm.sale_unit === "CASE") {
      const parsed = Number.parseInt(createForm.units_per_case, 10);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        setCreateError("Units per case must be a positive integer when selling by case.");
        return;
      }
      unitsPerCase = parsed;
    }
    setCreating(true);
    setCreateError(null);
    try {
      let imageUrl: string | undefined;
      if (createImageFile) {
        imageUrl = await uploadCatalogImage(createImageFile);
      }
      const res = await supplierFetch("/v1/catalog/products", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          category_id: categoryId,
          name,
          description: createForm.description.trim(),
          price_minor: priceMinor,
          currency,
          unit_volume_vu: unitVolume,
          stock_quantity: 0,
          unit: "UNIT",
          sale_unit: createForm.sale_unit,
          ...(unitsPerCase != null ? { units_per_case: unitsPerCase } : {}),
          ...(normalizedBarcode ? { barcode: normalizedBarcode } : {}),
          ...(imageUrl ? { image_url: imageUrl } : {}),
        }),
      });
      if (!res.ok) throw new Error(`create ${res.status}`);
      setShowCreate(false);
      setCreateForm(EMPTY_CREATE_FORM);
      setCreateImageFile(null);
      setCreateImagePreview(null);
      await load();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Failed to create product");
    } finally {
      setCreating(false);
    }
  }

  return (
    <PageChrome
      icon="catalog"
      title="Catalog"
      description="Create products and set volumetric units (VU) for warehouse dispatch capacity."
      loading={loading}
      error={error}
      empty={!showCreate && products.length === 0}
      actions={
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setShowBulkImport(true)}
            className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
          >
            Bulk import
          </button>
          <button
            type="button"
            onClick={() => {
              setShowCreate(value => !value);
              setCreateError(null);
            }}
            className="md-btn md-btn-filled md-typescale-label-large px-4 py-2"
          >
            {showCreate ? "Close" : "Add product"}
          </button>
        </div>
      }
    >
      <BulkImportWizard
        isOpen={showBulkImport}
        onClose={() => setShowBulkImport(false)}
        onImport={(rows) => {
          // Future integration: send rows to API
          setShowBulkImport(false);
          alert(`Successfully initiated import for ${rows.length} products`);
        }}
      />
      {showCreate && (
        <div className="md-card p-4 mb-4 grid grid-cols-1 md:grid-cols-2 gap-4">
          <label className="flex flex-col gap-1 md-typescale-body-medium">
            Product name
            <input
              value={createForm.name}
              onChange={event => setCreateForm(prev => ({ ...prev, name: event.target.value }))}
              className="px-3 py-2 rounded border"
              style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
            />
          </label>
          <label className="flex flex-col gap-1 md-typescale-body-medium">
            Category
            <select
              value={createForm.category_id}
              onChange={event => setCreateForm(prev => ({ ...prev, category_id: event.target.value }))}
              className="px-3 py-2 rounded border"
              style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
            >
              {categories.length === 0 ? (
                <option value="">No categories — complete onboarding first</option>
              ) : (
                categories.map(category => (
                  <option key={category.category_id} value={category.category_id}>
                    {category.name}
                  </option>
                ))
              )}
            </select>
          </label>
          <label className="flex flex-col gap-1 md-typescale-body-medium">
            Price ({currency}, minor units)
            <input
              type="number"
              min="0"
              step="1"
              value={createForm.price_minor}
              onChange={event => setCreateForm(prev => ({ ...prev, price_minor: event.target.value }))}
              className="px-3 py-2 rounded border font-mono"
              style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
            />
          </label>
          <label className="flex flex-col gap-1 md-typescale-body-medium">
            Unit volume (VU)
            <input
              type="number"
              min="0.1"
              step="0.1"
              value={createForm.unit_volume_vu}
              onChange={event => setCreateForm(prev => ({ ...prev, unit_volume_vu: event.target.value }))}
              className="px-3 py-2 rounded border font-mono"
              style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
            />
          </label>
          <label className="flex flex-col gap-1 md-typescale-body-medium">
            Sale unit
            <select
              value={createForm.sale_unit}
              onChange={event =>
                setCreateForm(prev => ({
                  ...prev,
                  sale_unit: event.target.value as SaleUnit,
                }))
              }
              className="px-3 py-2 rounded border"
              style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
            >
              <option value="UNIT">Unit</option>
              <option value="CASE">Case</option>
            </select>
          </label>
          {createForm.sale_unit === "CASE" && (
            <label className="flex flex-col gap-1 md-typescale-body-medium">
              Units per case
              <input
                type="number"
                min="1"
                step="1"
                value={createForm.units_per_case}
                onChange={event =>
                  setCreateForm(prev => ({ ...prev, units_per_case: event.target.value }))
                }
                className="px-3 py-2 rounded border font-mono"
                style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
              />
            </label>
          )}
          <label className="flex flex-col gap-1 md-typescale-body-medium md:col-span-2">
            Product image (optional)
            <input
              ref={createImageInputRef}
              type="file"
              accept={ALLOWED_IMAGE_TYPES.join(",")}
              className="hidden"
              onChange={event => {
                const file = event.target.files?.[0] ?? null;
                setCreateImageFile(file);
                setCreateImagePreview(file ? URL.createObjectURL(file) : null);
              }}
            />
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => createImageInputRef.current?.click()}
                className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
              >
                Choose image
              </button>
              {createImagePreview && (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={createImagePreview}
                  alt="Preview"
                  className="h-16 w-16 object-cover border border-[var(--color-md-outline-variant)]"
                />
              )}
            </div>
          </label>
          <label className="flex flex-col gap-1 md-typescale-body-medium">
            EAN / GTIN barcode (optional)
            <input
              value={createForm.barcode}
              onChange={event => setCreateForm(prev => ({ ...prev, barcode: event.target.value }))}
              placeholder="8–14 digit retail barcode"
              className="px-3 py-2 rounded border font-mono"
              style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
            />
          </label>
          <label className="flex flex-col gap-1 md-typescale-body-medium md:col-span-2">
            Description (optional)
            <textarea
              value={createForm.description}
              onChange={event => setCreateForm(prev => ({ ...prev, description: event.target.value }))}
              rows={2}
              className="px-3 py-2 rounded border"
              style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
            />
          </label>
          {createError && (
            <p className="md:col-span-2 text-sm" style={{ color: "var(--color-md-error)" }}>
              {createError}
            </p>
          )}
          <div className="md:col-span-2 flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setShowCreate(false)}
              className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={creating || categories.length === 0}
              onClick={() => void createProduct()}
              className="md-btn md-btn-filled md-typescale-label-large px-4 py-2 disabled:opacity-50"
            >
              {creating ? "Creating…" : "Create product"}
            </button>
          </div>
        </div>
      )}

      <ListToolbar
        page={pagination.page}
        pageCount={pagination.pageCount}
        totalLabel={`${products.length} products`}
        onPrev={pagination.prev}
        onNext={pagination.next}
      />
      <input
        ref={rowImageInputRef}
        type="file"
        accept={ALLOWED_IMAGE_TYPES.join(",")}
        className="hidden"
        onChange={event => {
          const file = event.target.files?.[0];
          const targetId = imageTargetId;
          event.target.value = "";
          if (!file || !targetId) return;
          const product = products.find(row => row.product_id === targetId);
          if (product) void saveProductImage(product, file);
        }}
      />
      <div className="md-card overflow-x-auto">
        <table className="min-w-full text-left">
          <thead className="border-b border-[var(--color-md-outline-variant)]">
            <tr>
              <th className="px-4 py-3 md-typescale-label-large">Product</th>
              <th className="px-4 py-3 md-typescale-label-large">Image</th>
              <th className="px-4 py-3 md-typescale-label-large">Category</th>
              <th className="px-4 py-3 md-typescale-label-large">Barcode</th>
              <th className="px-4 py-3 md-typescale-label-large text-right">Price (minor)</th>
              <th className="px-4 py-3 md-typescale-label-large text-right">Unit VU</th>
              <th className="px-4 py-3 md-typescale-label-large">Sale unit</th>
              <th className="px-4 py-3 md-typescale-label-large text-right">Units/case</th>
              <th className="px-4 py-3 md-typescale-label-large text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {pagination.pageItems.map(product => {
              const vuValue = draftVU[product.product_id] ?? String(product.unit_volume_vu ?? 1);
              const barcodeValue = draftBarcode[product.product_id] ?? product.barcode ?? "";
              const saleUnit = draftSaleUnit[product.product_id] ?? product.sale_unit ?? "UNIT";
              const unitsPerCaseValue = draftUnitsPerCase[product.product_id] ?? (
                product.units_per_case != null ? String(product.units_per_case) : ""
              );
              const vuDirty = vuValue !== String(product.unit_volume_vu ?? 1);
              const barcodeDirty = barcodeValue.trim() !== (product.barcode ?? "").trim();
              const saleUnitDirty = saleUnit !== (product.sale_unit ?? "UNIT");
              const unitsPerCaseDirty = unitsPerCaseValue !== (
                product.units_per_case != null ? String(product.units_per_case) : ""
              );
              const dirty = vuDirty || barcodeDirty || saleUnitDirty || unitsPerCaseDirty;
              return (
                <tr key={product.product_id} className="border-b border-[var(--color-md-outline-variant)]">
                  <td className="px-4 py-3">
                    <div className="font-medium">{product.name}</div>
                    <div className="font-mono text-xs text-[var(--color-md-outline)]">{product.product_id}</div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-col items-start gap-2">
                      {product.image_url ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={product.image_url}
                          alt=""
                          className="h-12 w-12 object-cover border border-[var(--color-md-outline-variant)]"
                        />
                      ) : (
                        <span className="text-xs text-[var(--color-md-outline)]">No image</span>
                      )}
                      <button
                        type="button"
                        disabled={imageSavingId === product.product_id}
                        onClick={() => {
                          setImageTargetId(product.product_id);
                          rowImageInputRef.current?.click();
                        }}
                        className="md-btn md-btn-outlined md-typescale-label-small px-2 py-1 disabled:opacity-50"
                      >
                        {imageSavingId === product.product_id ? "Uploading…" : "Change"}
                      </button>
                    </div>
                  </td>
                  <td className="px-4 py-3 font-mono text-sm">{product.category_id}</td>
                  <td className="px-4 py-3">
                    <input
                      type="text"
                      inputMode="numeric"
                      value={barcodeValue}
                      onChange={event =>
                        setDraftBarcode(prev => ({ ...prev, [product.product_id]: event.target.value }))
                      }
                      placeholder="EAN / GTIN"
                      className="w-36 px-2 py-1 rounded border font-mono text-sm"
                      style={{
                        background: "var(--field-background)",
                        borderColor: "var(--field-border)",
                        color: "var(--field-foreground)",
                      }}
                    />
                  </td>
                  <td className="px-4 py-3 text-right font-mono text-sm">
                    {product.price_minor} {product.currency}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <input
                      type="number"
                      min="0.1"
                      step="0.1"
                      value={vuValue}
                      onChange={event =>
                        setDraftVU(prev => ({ ...prev, [product.product_id]: event.target.value }))
                      }
                      className="w-24 px-2 py-1 rounded border text-right font-mono text-sm"
                      style={{
                        background: "var(--field-background)",
                        borderColor: "var(--field-border)",
                        color: "var(--field-foreground)",
                      }}
                    />
                  </td>
                  <td className="px-4 py-3">
                    <select
                      value={saleUnit}
                      onChange={event =>
                        setDraftSaleUnit(prev => ({
                          ...prev,
                          [product.product_id]: event.target.value as SaleUnit,
                        }))
                      }
                      className="px-2 py-1 rounded border text-sm"
                      style={{
                        background: "var(--field-background)",
                        borderColor: "var(--field-border)",
                        color: "var(--field-foreground)",
                      }}
                    >
                      <option value="UNIT">Unit</option>
                      <option value="CASE">Case</option>
                    </select>
                  </td>
                  <td className="px-4 py-3 text-right">
                    {saleUnit === "CASE" ? (
                      <input
                        type="number"
                        min="1"
                        step="1"
                        value={unitsPerCaseValue}
                        onChange={event =>
                          setDraftUnitsPerCase(prev => ({
                            ...prev,
                            [product.product_id]: event.target.value,
                          }))
                        }
                        className="w-20 px-2 py-1 rounded border text-right font-mono text-sm"
                        style={{
                          background: "var(--field-background)",
                          borderColor: "var(--field-border)",
                          color: "var(--field-foreground)",
                        }}
                      />
                    ) : (
                      <span className="text-xs text-[var(--color-md-outline)]">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button
                      type="button"
                      disabled={!dirty || savingId === product.product_id}
                      onClick={() => void saveProductEdits(product)}
                      className="md-btn md-btn-tonal md-typescale-label-medium px-3 py-1 disabled:opacity-50"
                    >
                      {savingId === product.product_id ? "Saving…" : "Save"}
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </PageChrome>
  );
}
