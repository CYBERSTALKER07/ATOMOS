import React from "react";
import { motion } from "framer-motion";
import { Package, ArrowUpRight } from "lucide-react";
import { PageSection } from "../PageSection";
import EmptyState from "../EmptyState";
import type { Product } from "../../lib/types";

interface QuickReorderSectionProps {
  reorderProducts: Product[];
  onRefresh: () => void;
  onAddToCart: (product: Product) => void;
}

export function QuickReorderSection({
  reorderProducts,
  onRefresh,
  onAddToCart,
}: QuickReorderSectionProps) {
  return (
    <PageSection
      title="Quick Reorder"
      description="Stage repeat purchases from your approved catalog."
    >
      {reorderProducts.length === 0 ? (
        <EmptyState
          headline="No products ready for reorder"
          body="Catalog feeds are still populating reorder candidates."
          variant="no-products"
          action="Sync"
          onAction={onRefresh}
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 !mt-0">
          {reorderProducts.map((p) => (
            <motion.button
              key={p.id}
              layout
              whileHover={{ y: -2 }}
              whileTap={{ scale: 0.98 }}
              onClick={() => onAddToCart(p)}
              className="flex items-center gap-4 p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl text-left hover:shadow-md transition-shadow group"
            >
              <div className="w-12 h-12 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-text-tertiary)] group-hover:bg-[var(--desk-accent-soft)] group-hover:text-[var(--desk-accent)] transition-colors">
                <Package size={20} />
              </div>
              <div className="flex-1 min-w-0">
                <p className="md-typescale-title-small font-light truncate text-[var(--desk-text-primary)]">
                  {p.name}
                </p>
                <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] truncate uppercase tracking-widest">
                  {p.supplier_name}
                </p>
              </div>
              <div className="text-right">
                <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                  {p.price.toLocaleString()}
                </p>
                <ArrowUpRight
                  size={14}
                  className="ml-auto opacity-20 group-hover:opacity-100 transition-opacity"
                />
              </div>
            </motion.button>
          ))}
        </div>
      )}
    </PageSection>
  );
}
