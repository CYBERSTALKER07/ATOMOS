"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { supplierFetch } from "@/lib/auth";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { BulkImportWizard } from "@/components/BulkImportWizard";
import type { CatalogProduct, CatalogCategory, CreateProductFormState } from "./components/types";
import { ALLOWED_IMAGE_TYPES, MAX_IMAGE_SIZE } from "./components/types";
import { CreateProductForm } from "./components/CreateProductForm";
import { sessionPackCurrency } from "@pegasusx/api-core";
import { CatalogTable } from "./components/CatalogTable";

export default function CatalogPage() {
  const t = usePortalT();
  const [products, setProducts] = useState<CatalogProduct[]>([]);
  const [categories, setCategories] = useState<CatalogCategory[]>([]);
  const [currency, setCurrency] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  const [showCreate, setShowCreate] = useState(false);
  const [showBulkImport, setShowBulkImport] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      const api = createSupplierApi();
      const profile = await api.getSupplierProfile();
      setCurrency(profile.currency || sessionPackCurrency());

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
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.failed_to_load_catalog"));
    } finally {
      setLoading(false);
    }
  }, []);

  useSupplierSessionReconcile(load);

  useEffect(() => {
    void load();
  }, [load]);

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

  const handleSaveProductImage = async (product: CatalogProduct, file: File) => {
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
  };

  const handleSaveProductEdits = async (product: CatalogProduct, updates: Partial<CatalogProduct>) => {
    const res = await supplierFetch(`/v1/catalog/products/${product.product_id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: product.name,
        price_minor: product.price_minor,
        currency: product.currency,
        unit: product.unit,
        unit_volume_vu: product.unit_volume_vu,
        barcode: product.barcode ?? "",
        sale_unit: product.sale_unit ?? "UNIT",
        is_active: product.is_active,
        version: product.version,
        ...updates,
      }),
    });
    if (!res.ok) throw new Error(`update ${res.status}`);
    const updated = (await res.json()) as CatalogProduct;
    setProducts(prev => prev.map(row => (row.product_id === updated.product_id ? updated : row)));
  };

  const handleCreateProduct = async (
    form: CreateProductFormState,
    normalizedBarcode: string | undefined,
    unitsPerCase: number | undefined,
    imageFile: File | null
  ) => {
    let imageUrl: string | undefined;
    if (imageFile) {
      imageUrl = await uploadCatalogImage(imageFile);
    }
    const res = await supplierFetch("/v1/catalog/products", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        category_id: form.category_id,
        name: form.name.trim(),
        description: form.description.trim(),
        price_minor: Number.parseInt(form.price_minor, 10),
        currency,
        unit_volume_vu: Number.parseFloat(form.unit_volume_vu),
        stock_quantity: 0,
        unit: "UNIT",
        sale_unit: form.sale_unit,
        ...(unitsPerCase != null ? { units_per_case: unitsPerCase } : {}),
        ...(normalizedBarcode ? { barcode: normalizedBarcode } : {}),
        ...(imageUrl ? { image_url: imageUrl } : {}),
      }),
    });
    if (!res.ok) throw new Error(`create ${res.status}`);
    setShowCreate(false);
    await load();
  };

  return (
    <PageChrome
      icon="catalog"
      title={t("portal.nav.catalog")}
      description={t("supplier_portal.residual.text.create_products_and_set_volumetric_units_vu_for_warehouse_dispat")}
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
            onClick={() => setShowCreate(value => !value)}
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
        <CreateProductForm
          categories={categories}
          currency={currency}
          onCancel={() => setShowCreate(false)}
          onSave={handleCreateProduct}
        />
      )}

      <CatalogTable
        products={products}
        onSaveEdits={handleSaveProductEdits}
        onImageChange={handleSaveProductImage}
      />
    </PageChrome>
  );
}
