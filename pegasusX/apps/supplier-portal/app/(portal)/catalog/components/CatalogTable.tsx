import { useState, useRef } from "react";
import type { CatalogProduct, SaleUnit } from "./types";
import { ALLOWED_IMAGE_TYPES } from "./types";
import { normalizeEanBarcode } from "@pegasusx/validation";
import { ListToolbar } from "@/components/ListToolbar";
import { usePagination } from "@/lib/use-pagination";

interface CatalogTableProps {
  products: CatalogProduct[];
  onSaveEdits: (product: CatalogProduct, updates: Partial<CatalogProduct>) => Promise<void>;
  onImageChange: (product: CatalogProduct, file: File) => Promise<void>;
}

export function CatalogTable({ products, onSaveEdits, onImageChange }: CatalogTableProps) {
  const pagination = usePagination(products, 20);
  
  const [draftVU, setDraftVU] = useState<Record<string, string>>({});
  const [draftBarcode, setDraftBarcode] = useState<Record<string, string>>({});
  const [draftUnitsPerCase, setDraftUnitsPerCase] = useState<Record<string, string>>({});
  const [draftSaleUnit, setDraftSaleUnit] = useState<Record<string, SaleUnit>>({});
  
  const [savingId, setSavingId] = useState<string | null>(null);
  const [errorId, setErrorId] = useState<string | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  
  const [imageSavingId, setImageSavingId] = useState<string | null>(null);
  const [imageTargetId, setImageTargetId] = useState<string | null>(null);
  const rowImageInputRef = useRef<HTMLInputElement>(null);

  const handleSave = async (product: CatalogProduct) => {
    const rawVu = draftVU[product.product_id] ?? String(product.unit_volume_vu ?? 1);
    const parsedVu = Number.parseFloat(rawVu);
    if (!Number.isFinite(parsedVu) || parsedVu <= 0) {
      setErrorId(product.product_id);
      setErrorMsg("Unit volume must be a positive number.");
      return;
    }

    const barcodeRaw = draftBarcode[product.product_id] ?? product.barcode ?? "";
    const trimmedBarcode = barcodeRaw.trim();
    let normalizedBarcode: string | undefined = "";
    if (trimmedBarcode) {
      const result = normalizeEanBarcode(trimmedBarcode);
      if (!result.ok) {
        setErrorId(product.product_id);
        setErrorMsg(
          result.error === "invalid_barcode_checksum"
            ? "Barcode checksum is invalid."
            : "Barcode must be 8–14 digits (EAN/GTIN)."
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
        setErrorId(product.product_id);
        setErrorMsg("Units per case must be a positive integer when selling by case.");
        return;
      }
      unitsPerCase = parsed;
    }

    setSavingId(product.product_id);
    setErrorId(null);
    setErrorMsg(null);
    
    try {
      await onSaveEdits(product, {
        unit_volume_vu: parsedVu,
        barcode: normalizedBarcode,
        sale_unit: saleUnit,
        units_per_case: unitsPerCase,
      });
      // Clear drafts on success
      setDraftVU(prev => { const next = { ...prev }; delete next[product.product_id]; return next; });
      setDraftBarcode(prev => { const next = { ...prev }; delete next[product.product_id]; return next; });
      setDraftUnitsPerCase(prev => { const next = { ...prev }; delete next[product.product_id]; return next; });
      setDraftSaleUnit(prev => { const next = { ...prev }; delete next[product.product_id]; return next; });
    } catch (err) {
      setErrorId(product.product_id);
      setErrorMsg(err instanceof Error ? err.message : "Failed to save product");
    } finally {
      setSavingId(null);
    }
  };

  const handleImageChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    const targetId = imageTargetId;
    event.target.value = "";
    if (!file || !targetId) return;
    const product = products.find(row => row.product_id === targetId);
    if (!product) return;
    
    setImageSavingId(product.product_id);
    try {
      await onImageChange(product, file);
    } finally {
      setImageSavingId(null);
      setImageTargetId(null);
    }
  };

  return (
    <>
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
        onChange={handleImageChange}
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
                    {errorId === product.product_id && errorMsg && (
                      <div className="text-xs mt-1" style={{ color: "var(--color-md-error)" }}>{errorMsg}</div>
                    )}
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
                      onClick={() => void handleSave(product)}
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
    </>
  );
}
