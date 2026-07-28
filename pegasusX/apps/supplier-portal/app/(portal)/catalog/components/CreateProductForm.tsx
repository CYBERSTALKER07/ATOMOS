import { useState, useRef } from "react";
import type { CatalogCategory, CreateProductFormState, SaleUnit } from "./types";
import { ALLOWED_IMAGE_TYPES } from "./types";
import { normalizeEanBarcode } from "@pegasusx/validation";

export const EMPTY_CREATE_FORM: CreateProductFormState = {
  name: "",
  category_id: "",
  description: "",
  price_minor: "",
  unit_volume_vu: "1",
  units_per_case: "",
  sale_unit: "UNIT",
  barcode: "",
};

interface CreateProductFormProps {
  categories: CatalogCategory[];
  currency: string;
  initialCategory?: string;
  onCancel: () => void;
  onSave: (
    form: CreateProductFormState,
    normalizedBarcode: string | undefined,
    unitsPerCase: number | undefined,
    imageFile: File | null
  ) => Promise<void>;
}

export function CreateProductForm({ categories, currency, initialCategory, onCancel, onSave }: CreateProductFormProps) {
  const [form, setForm] = useState<CreateProductFormState>({
    ...EMPTY_CREATE_FORM,
    category_id: initialCategory || categories[0]?.category_id || "",
  });
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string | null>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);

  const handleSubmit = async () => {
    const name = form.name.trim();
    const categoryId = form.category_id.trim();
    const priceMinor = Number.parseInt(form.price_minor, 10);
    const unitVolume = Number.parseFloat(form.unit_volume_vu);
    
    if (!name || !categoryId) {
      setError("Name and category are required.");
      return;
    }
    if (!Number.isFinite(priceMinor) || priceMinor < 0) {
      setError("Price must be a non-negative integer (minor units).");
      return;
    }
    if (!Number.isFinite(unitVolume) || unitVolume <= 0) {
      setError("Unit VU must be a positive number.");
      return;
    }
    
    const trimmedBarcode = form.barcode.trim();
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
    
    let unitsPerCase: number | undefined;
    if (form.sale_unit === "CASE") {
      const parsed = Number.parseInt(form.units_per_case, 10);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        setError("Units per case must be a positive integer when selling by case.");
        return;
      }
      unitsPerCase = parsed;
    }
    
    setCreating(true);
    setError(null);
    try {
      await onSave(form, normalizedBarcode, unitsPerCase, imageFile);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create product");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="md-card p-4 mb-4 grid grid-cols-1 md:grid-cols-2 gap-4">
      <label className="flex flex-col gap-1 md-typescale-body-medium">
        Product name
        <input
          value={form.name}
          onChange={e => setForm(prev => ({ ...prev, name: e.target.value }))}
          className="px-3 py-2 rounded border"
          style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
        />
      </label>
      <label className="flex flex-col gap-1 md-typescale-body-medium">
        Category
        <select
          value={form.category_id}
          onChange={e => setForm(prev => ({ ...prev, category_id: e.target.value }))}
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
          value={form.price_minor}
          onChange={e => setForm(prev => ({ ...prev, price_minor: e.target.value }))}
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
          value={form.unit_volume_vu}
          onChange={e => setForm(prev => ({ ...prev, unit_volume_vu: e.target.value }))}
          className="px-3 py-2 rounded border font-mono"
          style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
        />
      </label>
      <label className="flex flex-col gap-1 md-typescale-body-medium">
        Sale unit
        <select
          value={form.sale_unit}
          onChange={e => setForm(prev => ({ ...prev, sale_unit: e.target.value as SaleUnit }))}
          className="px-3 py-2 rounded border"
          style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
        >
          <option value="UNIT">Unit</option>
          <option value="CASE">Case</option>
        </select>
      </label>
      {form.sale_unit === "CASE" && (
        <label className="flex flex-col gap-1 md-typescale-body-medium">
          Units per case
          <input
            type="number"
            min="1"
            step="1"
            value={form.units_per_case}
            onChange={e => setForm(prev => ({ ...prev, units_per_case: e.target.value }))}
            className="px-3 py-2 rounded border font-mono"
            style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
          />
        </label>
      )}
      <label className="flex flex-col gap-1 md-typescale-body-medium md:col-span-2">
        Product image (optional)
        <input
          ref={imageInputRef}
          type="file"
          accept={ALLOWED_IMAGE_TYPES.join(",")}
          className="hidden"
          onChange={e => {
            const file = e.target.files?.[0] ?? null;
            setImageFile(file);
            setImagePreview(file ? URL.createObjectURL(file) : null);
          }}
        />
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => imageInputRef.current?.click()}
            className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
          >
            Choose image
          </button>
          {imagePreview && (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={imagePreview}
              alt="Preview"
              className="h-16 w-16 object-cover border border-[var(--color-md-outline-variant)]"
            />
          )}
        </div>
      </label>
      <label className="flex flex-col gap-1 md-typescale-body-medium">
        EAN / GTIN barcode (optional)
        <input
          value={form.barcode}
          onChange={e => setForm(prev => ({ ...prev, barcode: e.target.value }))}
          placeholder="8–14 digit retail barcode"
          className="px-3 py-2 rounded border font-mono"
          style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
        />
      </label>
      <label className="flex flex-col gap-1 md-typescale-body-medium md:col-span-2">
        Description (optional)
        <textarea
          value={form.description}
          onChange={e => setForm(prev => ({ ...prev, description: e.target.value }))}
          rows={2}
          className="px-3 py-2 rounded border"
          style={{ background: "var(--field-background)", borderColor: "var(--field-border)" }}
        />
      </label>
      {error && (
        <p className="md:col-span-2 text-sm" style={{ color: "var(--color-md-error)" }}>
          {error}
        </p>
      )}
      <div className="md:col-span-2 flex justify-end gap-2">
        <button
          type="button"
          onClick={onCancel}
          className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
        >
          Cancel
        </button>
        <button
          type="button"
          disabled={creating || categories.length === 0}
          onClick={() => void handleSubmit()}
          className="md-btn md-btn-filled md-typescale-label-large px-4 py-2 disabled:opacity-50"
        >
          {creating ? "Creating…" : "Create product"}
        </button>
      </div>
    </div>
  );
}
