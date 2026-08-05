import { Building2, Layers, Package, Box, CheckCircle2 } from "lucide-react";
import { PageSection } from "@/components/PageSection";
import type { AutoOrderSettings } from "@/lib/types";

export function AutoOrderRules({
  settings,
  handleToggle,
}: {
  settings: AutoOrderSettings | undefined;
  handleToggle: (
    type: "global" | "supplier" | "category" | "product" | "variant",
    enabled: boolean,
    hasHistory: boolean,
    id?: string
  ) => void;
}) {
  return (
    <>
      <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">Global Auto-Order</h3>
            <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] mt-1">Auto-order everything from all suppliers</p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              className="sr-only peer"
              checked={settings?.global_enabled || false}
              onChange={(e) => handleToggle("global", e.target.checked, settings?.has_any_history || false)}
            />
            <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
          </label>
        </div>
        {settings?.global_enabled && (
          <div className="mt-4 flex items-center gap-2 text-[var(--desk-success)]">
            <CheckCircle2 size={16} />
            <span className="md-typescale-body-small">Global on. Scoped Off still blocks matching suppliers/categories/products/size-variants.</span>
          </div>
        )}
      </div>

      {(settings?.supplier_overrides?.length ?? 0) > 0 && (
        <PageSection title="Supplier Overrides">
          <div className="space-y-2">
            {settings?.supplier_overrides.map((item) => (
              <div key={item.supplier_id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-xl">
                <div className="flex items-center gap-3">
                  <Building2 size={18} className="text-[var(--desk-text-tertiary)]" />
                  <div>
                    <div className="md-typescale-body-medium">{item.supplier_id}</div>
                    <div className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Supplier-level override</div>
                  </div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    className="sr-only peer"
                    checked={item.enabled}
                    onChange={(e) => handleToggle("supplier", e.target.checked, item.has_history, item.supplier_id)}
                  />
                  <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
                </label>
              </div>
            ))}
          </div>
        </PageSection>
      )}

      {(settings?.category_overrides?.length ?? 0) > 0 && (
        <PageSection title="Category Overrides">
          <div className="space-y-2">
            {settings?.category_overrides.map((item) => (
              <div key={item.category_id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-xl">
                <div className="flex items-center gap-3">
                  <Layers size={18} className="text-[var(--desk-text-tertiary)]" />
                  <div>
                    <div className="md-typescale-body-medium">{item.category_id}</div>
                    <div className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Category-level override</div>
                  </div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    className="sr-only peer"
                    checked={item.enabled}
                    onChange={(e) => handleToggle("category", e.target.checked, item.has_history, item.category_id)}
                  />
                  <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
                </label>
              </div>
            ))}
          </div>
        </PageSection>
      )}

      {(settings?.product_overrides?.length ?? 0) > 0 && (
        <PageSection title="Product Overrides">
          <div className="space-y-2">
            {settings?.product_overrides.map((item) => (
              <div key={item.product_id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-xl">
                <div className="flex items-center gap-3">
                  <Package size={18} className="text-[var(--desk-text-tertiary)]" />
                  <div>
                    <div className="md-typescale-body-medium">{item.product_id}</div>
                    <div className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Product-level override</div>
                  </div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    className="sr-only peer"
                    checked={item.enabled}
                    onChange={(e) => handleToggle("product", e.target.checked, false, item.product_id)}
                  />
                  <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
                </label>
              </div>
            ))}
          </div>
        </PageSection>
      )}

      {(settings?.variant_overrides?.length ?? 0) > 0 && (
        <PageSection title="Size / variant Overrides">
          <div className="space-y-2">
            {settings?.variant_overrides.map((item) => (
              <div key={item.variant_id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-xl">
                <div className="flex items-center gap-3">
                  <Box size={18} className="text-[var(--desk-text-tertiary)]" />
                  <div>
                    <div className="md-typescale-body-medium">{item.variant_id}</div>
                    <div className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Size / variant override</div>
                  </div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    className="sr-only peer"
                    checked={item.enabled}
                    onChange={(e) => handleToggle("variant", e.target.checked, false, item.variant_id)}
                  />
                  <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
                </label>
              </div>
            ))}
          </div>
        </PageSection>
      )}
    </>
  );
}
