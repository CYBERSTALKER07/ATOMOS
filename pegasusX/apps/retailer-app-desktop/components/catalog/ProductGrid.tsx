import { motion, AnimatePresence } from "framer-motion";
import { Package } from "lucide-react";
import { Skeleton } from "../../../components/Skeleton";
import EmptyState from "../../../components/EmptyState";
import { isCatalogBlocked } from "../../../lib/stock-policy";
import { productDisplayPrice, productListPrice, productSalePrice } from "../../../lib/types";
import type { Product } from "../../../lib/types";

export interface EmptyStateConfig {
  headline: string;
  body: string;
  variant: "restricted" | "offline" | "error" | "no-products" | "no-results";
}

export interface ProductGridProps {
  loadingProducts: boolean;
  filteredProducts: Product[];
  emptyStateConfig: EmptyStateConfig;
  loadIssue: "restricted" | "offline" | "error" | null;
  refreshAll: () => void;
  hasActiveFilters: boolean;
  clearFilters: () => void;
  setSelectedProduct: (p: Product) => void;
  addToCart: (p: Product) => void;
}

export function ProductGrid({
  loadingProducts,
  filteredProducts,
  emptyStateConfig,
  loadIssue,
  refreshAll,
  hasActiveFilters,
  clearFilters,
  setSelectedProduct,
  addToCart
}: ProductGridProps) {
  return (
    <AnimatePresence mode="popLayout">
            {loadingProducts ? (
              <motion.div
                key="loading"
                className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6 !mt-0"
              >
                {[0, 1, 2, 3, 4, 5].map((i) => (
                  <Skeleton key={i} style={{ height: 288, borderRadius: 16 }} />
                ))}
              </motion.div>
            ) : filteredProducts.length === 0 ? (
              <motion.div
                key="empty"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="py-20"
              >
                <EmptyState
                  headline={emptyStateConfig.headline}
                  body={emptyStateConfig.body}
                  variant={emptyStateConfig.variant}
                  action={
                    loadIssue ? "Retry Sync" : hasActiveFilters ? "Reset Filters" : undefined
                  }
                  onAction={
                    loadIssue
                      ? refreshAll
                      : hasActiveFilters
                        ? clearFilters
                        : undefined
                  }
                />
              </motion.div>
            ) : (
              <motion.div
                key="list"
                layout
                className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6"
              >
                {filteredProducts.map((product) => {
                  const blocked = isCatalogBlocked(product);
                  return (
                  <motion.article
                    key={product.id}
                    layout
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, scale: 0.95 }}
                    onClick={() => !blocked && setSelectedProduct(product)}
                    className={`group bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl overflow-hidden transition-all ${
                      blocked
                        ? "opacity-50 grayscale cursor-not-allowed"
                        : "cursor-pointer hover:shadow-md hover:-translate-y-1 active:scale-[0.98]"
                    }`}
                  >
                    <div className="relative h-44 bg-[var(--desk-surface-subtle)] overflow-hidden">
                      {product.image_url ? (
                        <img
                          src={product.image_url}
                          alt={product.name}
                          className="w-full h-full object-cover"
                        />
                      ) : (
                        <div className="w-full h-full flex items-center justify-center opacity-20">
                          <Package size={48} />
                        </div>
                      )}
                      <div className="absolute top-3 right-3">
                        <StockBadge
                          stock={product.available_stock}
                          acceptsBackorder={product.accepts_backorder}
                        />
                      </div>
                    </div>

                    <div className="p-5 flex flex-col gap-4">
                      <div className="space-y-1">
                        <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)] line-clamp-1 group-hover:text-[var(--desk-accent)] transition-colors">
                          {product.name}
                        </h3>
                        <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                          {product.supplier_name}
                        </p>
                      </div>

                      <div className="flex items-end justify-between">
                        <div className="flex flex-col">
                          <span className="md-typescale-label-small text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                            Price Point
                          </span>
                          {productSalePrice(product) != null ? (
                            <div className="flex items-baseline gap-2">
                              <span className="md-typescale-label-small tabular-nums text-[var(--desk-text-tertiary)] line-through">
                                {productListPrice(product).toLocaleString()}
                              </span>
                              <span className="md-typescale-title-medium font-light tabular-nums text-[var(--desk-accent)]">
                                {productDisplayPrice(product).toLocaleString()}{" "}
                                <small className="text-[var(--desk-text-tertiary)] ml-0.5">
                                  UZS
                                </small>
                              </span>
                            </div>
                          ) : (
                            <span className="md-typescale-title-medium font-light tabular-nums text-[var(--desk-text-primary)]">
                              {productDisplayPrice(product).toLocaleString()}{" "}
                              <small className="text-[var(--desk-text-tertiary)] ml-0.5">
                                UZS
                              </small>
                            </span>
                          )}
                        </div>
                        <button
                          type="button"
                          className="portal-btn portal-btn--primary rounded-lg h-9 px-4 font-light shadow-[var(--shadow-sm)] transition-all active:scale-95"
                          onClick={(e) => {
                            e.stopPropagation();
                            if (!blocked) addToCart(product);
                          }}
                          disabled={blocked}
                        >
                          Add to Cart
                        </button>
                      </div>
                    </div>
                  </motion.article>
                  );
                })}
              </motion.div>
            )}
          </AnimatePresence>
  );
}

export function StockBadge({
  stock,
  acceptsBackorder,
}: {
  stock?: number;
  acceptsBackorder?: boolean;
}) {
  if (stock !== undefined && stock <= 0) {
    if (acceptsBackorder) {
      return (
        <span className="px-2.5 py-1 rounded-lg bg-[var(--desk-warning)] text-white md-typescale-label-small font-light uppercase tracking-widest shadow-[var(--shadow-sm)]">
          Backorder
        </span>
      );
    }
    return (
      <span className="px-2.5 py-1 rounded-lg bg-[var(--desk-danger)] text-white md-typescale-label-small font-light uppercase tracking-widest shadow-[var(--shadow-sm)]">
        Empty
      </span>
    );
  }

  if (stock !== undefined && stock <= 5) {
    return (
      <span className="px-2.5 py-1 rounded-lg bg-[var(--desk-warning)] text-white md-typescale-label-small font-light uppercase tracking-widest shadow-[var(--shadow-sm)]">
        Critical
      </span>
    );
  }

  return (
    <span className="px-2.5 py-1 rounded-lg bg-[var(--desk-success)] text-white md-typescale-label-small font-light uppercase tracking-widest shadow-[var(--shadow-sm)]">
      Stable
    </span>
  );
}
