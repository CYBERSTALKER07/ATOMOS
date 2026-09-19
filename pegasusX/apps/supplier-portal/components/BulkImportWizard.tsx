"use client";

import { usePortalT } from "@/lib/i18n";
import { useState, useRef } from 'react';

export function BulkImportWizard({
  isOpen,
  onClose,
  onImport,
}: {
  isOpen: boolean;
  onClose: () => void;
  onImport: (rows: any[]) => void;
}) {
  const t = usePortalT();
  const [step, setStep] = useState<'upload' | 'map' | 'review'>('upload');
  const [file, setFile] = useState<File | null>(null);
  const [columns, setColumns] = useState<string[]>([]);
  const [rows, setRows] = useState<any[][]>([]);
  const [mapping, setMapping] = useState<Record<string, string>>({});
  
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (f) {
      setFile(f);
      const reader = new FileReader();
      reader.onload = (evt) => {
        const text = evt.target?.result as string;
        const lines = text.split('\n').filter(l => l.trim().length > 0);
        if (lines.length > 0) {
          const cols = lines[0].split(',').map(s => s.trim());
          const data = lines.slice(1).map(l => l.split(',').map(s => s.trim()));
          setColumns(cols);
          setRows(data);
          
          // Auto-map based on common names
          const initialMap: Record<string, string> = {};
          cols.forEach((col, idx) => {
            const low = col.toLowerCase();
            if (low.includes('name')) initialMap['name'] = String(idx);
            if (low.includes('price')) initialMap['price'] = String(idx);
            if (low.includes('barcode')) initialMap['barcode'] = String(idx);
            if (low.includes('category')) initialMap['category'] = String(idx);
            if (low.includes('volume') || low.includes('vu')) initialMap['vu'] = String(idx);
          });
          setMapping(initialMap);
          setStep('map');
        }
      };
      reader.readAsText(f);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="bg-[var(--surface)] w-full max-w-2xl rounded-xl shadow-xl overflow-hidden flex flex-col max-h-[90vh]">
        <div className="px-6 py-4 border-b border-[var(--border)] flex items-center justify-between">
          <h2 className="text-xl font-semibold">{t("supplier_portal.bulk_import_wizard.text.bulk_import_wizard")}</h2>
          <button onClick={onClose} className="text-[var(--muted)] hover:text-current">&times;</button>
        </div>
        
        <div className="p-6 flex-1 overflow-y-auto">
          {step === 'upload' && (
            <div className="flex flex-col items-center justify-center py-12 border-2 border-dashed border-[var(--border)] rounded-lg">
              <p className="mb-4 text-sm text-[var(--muted)]">{t("supplier_portal.bulk_import_wizard.text.upload_a_csv_file_to_import_products")}</p>
              <input type="file" accept=".csv" className="hidden" ref={fileInputRef} onChange={handleFileChange} />
              <button 
                type="button" 
                onClick={() => fileInputRef.current?.click()}
                className="md-btn md-btn-filled px-4 py-2"
              >
                Choose CSV File
              </button>
            </div>
          )}

          {step === 'map' && (
            <div className="space-y-4">
              <p className="text-sm">{t("supplier_portal.bulk_import_wizard.text.map_your_csv_columns_to_product_fields")}</p>
              <div className="grid grid-cols-2 gap-4">
                {['name', 'price', 'barcode', 'category', 'vu'].map(field => (
                  <label key={field} className="flex flex-col gap-1 text-sm">
                    <span className="capitalize">{field}</span>
                    <select 
                      className="px-3 py-2 rounded border bg-[var(--field-background)] border-[var(--field-border)]"
                      value={mapping[field] || ''}
                      onChange={e => setMapping(prev => ({...prev, [field]: e.target.value}))}
                    >
                      <option value="">-- Ignore --</option>
                      {columns.map((col, idx) => (
                        <option key={idx} value={String(idx)}>{col}</option>
                      ))}
                    </select>
                  </label>
                ))}
              </div>
            </div>
          )}

          {step === 'review' && (
            <div className="space-y-4">
              <p className="text-sm">Review imported products ({rows.length} total):</p>
              <div className="overflow-x-auto border border-[var(--border)] rounded">
                <table className="min-w-full text-sm text-left">
                  <thead className="bg-[var(--field-background)] border-b border-[var(--border)]">
                    <tr>
                      <th className="px-3 py-2">{t("supplier_portal.analytics.knowledge_graph.text.name")}</th>
                      <th className="px-3 py-2">{t("supplier_portal.pricing.retailer_overrides.text.price")}</th>
                      <th className="px-3 py-2">{t("supplier_portal.catalog.components.catalog_table.text.barcode")}</th>
                      <th className="px-3 py-2">{t("supplier_portal.catalog.components.catalog_table.text.category")}</th>
                      <th className="px-3 py-2">VU</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.slice(0, 5).map((row, idx) => (
                      <tr key={idx} className="border-b border-[var(--border)]">
                        <td className="px-3 py-2">{mapping['name'] ? row[Number(mapping['name'])] : '—'}</td>
                        <td className="px-3 py-2">{mapping['price'] ? row[Number(mapping['price'])] : '—'}</td>
                        <td className="px-3 py-2">{mapping['barcode'] ? row[Number(mapping['barcode'])] : '—'}</td>
                        <td className="px-3 py-2">{mapping['category'] ? row[Number(mapping['category'])] : '—'}</td>
                        <td className="px-3 py-2">{mapping['vu'] ? row[Number(mapping['vu'])] : '—'}</td>
                      </tr>
                    ))}
                    {rows.length > 5 && (
                      <tr>
                        <td colSpan={5} className="px-3 py-2 text-center text-[var(--muted)]">...and {rows.length - 5} more</td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>

        <div className="px-6 py-4 border-t border-[var(--border)] flex justify-end gap-3 bg-[var(--field-background)]">
          <button type="button" onClick={onClose} className="md-btn md-btn-outlined px-4 py-2">{t("common.action.cancel")}</button>
          
          {step === 'map' && (
            <button type="button" onClick={() => setStep('review')} className="md-btn md-btn-filled px-4 py-2">{t("supplier_portal.bulk_import_wizard.text.continue")}</button>
          )}
          {step === 'review' && (
            <button 
              type="button" 
              onClick={() => {
                const mappedData = rows.map(row => ({
                  name: mapping['name'] ? row[Number(mapping['name'])] : '',
                  price: mapping['price'] ? row[Number(mapping['price'])] : '',
                  barcode: mapping['barcode'] ? row[Number(mapping['barcode'])] : '',
                  category: mapping['category'] ? row[Number(mapping['category'])] : '',
                  vu: mapping['vu'] ? row[Number(mapping['vu'])] : '',
                }));
                onImport(mappedData);
              }} 
              className="md-btn md-btn-filled px-4 py-2"
            >
              Import {rows.length} Products
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
