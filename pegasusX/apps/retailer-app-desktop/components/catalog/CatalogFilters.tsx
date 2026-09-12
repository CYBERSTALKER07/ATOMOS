"use client";

import { usePortalT } from "@/lib/i18n";
import { Search, SlidersHorizontal } from "lucide-react";
import { Skeleton } from "../Skeleton";
import type { Supplier } from "../../lib/types";

export interface CatalogFiltersProps {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  hasActiveFilters: boolean;
  clearFilters: () => void;
  categoryTabs: string[];
  activeCategory: string;
  setActiveCategory: (category: string) => void;
  categorySuppliersLoading: boolean;
  categorySuppliersError: string | null;
  categorySuppliers: Supplier[];
  activeSupplier: string;
  setActiveSupplier: (supplierId: string) => void;
  supplierList: Supplier[];
}

export function CatalogFilters({
  searchQuery,
  setSearchQuery,
  hasActiveFilters,
  clearFilters,
  categoryTabs,
  activeCategory,
  setActiveCategory,
  categorySuppliersLoading,
  categorySuppliersError,
  categorySuppliers,
  activeSupplier,
  setActiveSupplier,
  supplierList
}: CatalogFiltersProps) {
  const t = usePortalT();
  return (
    <div className="mb-8 flex flex-col gap-6">
        <div className="flex flex-wrap items-center gap-4 p-2 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
          <div className="flex-1 min-w-[280px] relative group">
            <Search
              className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors"
              size={18}
            />
            <input
              type="text"
              placeholder={t("retailer_desktop.catalog.catalog_filters.text.search_assets_and_suppliers")}
              className="w-full h-11 pl-11 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] transition-all md-typescale-body-medium text-[var(--desk-text-primary)]"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>

          <div className="h-11 flex items-center gap-2">
            <button
              onClick={clearFilters}
              disabled={!hasActiveFilters}
              className={`px-4 h-full rounded-xl md-typescale-label-large transition-colors flex items-center gap-2 ${
                hasActiveFilters
                  ? "hover:bg-[var(--desk-surface-subtle)] text-[var(--desk-text-secondary)]"
                  : "text-[var(--desk-text-tertiary)] opacity-50 cursor-not-allowed"
              }`}
            >
              <SlidersHorizontal size={16} />
              Reset
            </button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {categoryTabs.map((category) => (
            <button
              key={category}
              onClick={() => setActiveCategory(category)}
              className={`px-5 py-2 rounded-full md-typescale-label-large font-light transition-all ${
                activeCategory === category
                  ? "bg-[var(--desk-accent)] text-white shadow-[var(--shadow-sm)] scale-105"
                  : "bg-[var(--desk-surface)] text-[var(--desk-text-secondary)] border border-[var(--desk-border)] hover:bg-[var(--desk-surface-subtle)]"
              }`}
            >
              {category}
            </button>
          ))}
        </div>

        {activeCategory !== "All" && (
          <div className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-4 py-3">
            <p className="text-[10px] font-black uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2">
              Category suppliers
            </p>
            {categorySuppliersLoading ? (
              <div className="flex flex-wrap gap-2">
                {[0, 1, 2, 3].map((i) => (
                  <Skeleton key={i} style={{ height: 32, width: 96, borderRadius: 9999 }} />
                ))}
              </div>
            ) : categorySuppliersError ? (
              <p className="text-sm text-orange-700">{categorySuppliersError}</p>
            ) : categorySuppliers.length === 0 ? (
              <p className="text-sm text-[var(--desk-text-tertiary)]">
                No suppliers mapped to this category yet.
              </p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {categorySuppliers.map((supplier) => (
                  <button
                    key={supplier.id}
                    type="button"
                    onClick={() => setActiveSupplier(supplier.id)}
                    className={`px-3 py-1.5 rounded-full text-xs font-light uppercase tracking-wide border ${
                      activeSupplier === supplier.id
                        ? "border-[var(--desk-accent)] bg-[var(--desk-accent-soft)] text-[var(--desk-accent)]"
                        : "border-[var(--desk-border)] text-[var(--desk-text-secondary)]"
                    }`}
                  >
                    {supplier.name}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {supplierList.length > 0 && (
          <div className="lg:hidden flex flex-wrap items-center gap-2">
            <button
              onClick={() => setActiveSupplier("")}
              className={`px-4 py-1.5 rounded-full text-[11px] font-light uppercase tracking-wide transition-all ${
                activeSupplier === ""
                  ? "bg-[var(--desk-accent-soft)] text-[var(--desk-accent)]"
                  : "bg-[var(--desk-surface)] border border-[var(--desk-border)] text-[var(--desk-text-secondary)]"
              }`}
            >
              All Suppliers
            </button>
            {supplierList.slice(0, 6).map((supplier) => (
              <button
                key={supplier.id}
                onClick={() => setActiveSupplier(supplier.id)}
                className={`px-4 py-1.5 rounded-full text-[11px] font-light uppercase tracking-wide transition-all ${
                  activeSupplier === supplier.id
                    ? "bg-[var(--desk-accent-soft)] text-[var(--desk-accent)]"
                    : "bg-[var(--desk-surface)] border border-[var(--desk-border)] text-[var(--desk-text-secondary)]"
                }`}
              >
                {supplier.name}
              </button>
            ))}
          </div>
        )}
      </div>
  );
}
