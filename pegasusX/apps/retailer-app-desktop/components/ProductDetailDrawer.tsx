"use client";

import { useEffect } from "react";
import {
  X,
  Package,
  ShoppingCart,
  Building2,
  Layers,
  Info,
  ChevronRight,
} from "lucide-react";
import { Button, Chip } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
import { useCart } from "../lib/cart";
import type { Product, Variant } from "../lib/types";

interface ProductDetailDrawerProps {
  product: Product | null;
  isOpen: boolean;
  onClose: () => void;
}

export default function ProductDetailDrawer({
  product,
  isOpen,
  onClose,
}: ProductDetailDrawerProps) {
  const { addToCart } = useCart();

  useEffect(() => {
    if (!isOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [isOpen, onClose]);

  const variants = product?.variants ?? [];
  const hasVariants = variants.length > 0;

  return (
    <AnimatePresence>
      {isOpen && product && (
        <div className="fixed inset-0 z-[100] overflow-hidden">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-[#0a0a0a]/60 backdrop-blur-sm"
            onClick={onClose}
          />

          <motion.div
            initial={{ x: "100%" }}
            animate={{ x: 0 }}
            exit={{ x: "100%" }}
            transition={{ type: "spring", stiffness: 400, damping: 40 }}
            className="absolute top-0 right-0 w-full max-w-lg h-full shadow-2xl flex flex-col border-l border-[var(--desk-border)] bg-[var(--desk-surface)]"
          >
            {/* Header */}
            <div className="p-6 flex justify-between items-center border-b border-[var(--desk-border)]">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] flex items-center justify-center">
                  <Package size={22} />
                </div>
                <div>
                  <h2 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)]">
                    Asset Specs
                  </h2>
                  <p className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                    Network SKU: #{product.id.slice(-8)}
                  </p>
                </div>
              </div>
              <button
                onClick={onClose}
                className="w-10 h-10 rounded-full hover:bg-[var(--desk-surface-subtle)] flex items-center justify-center transition-colors"
              >
                <X size={20} />
              </button>
            </div>

            {/* Content */}
            <div className="flex-1 overflow-y-auto">
              <div className="relative h-64 w-full flex items-center justify-center bg-[var(--desk-surface-subtle)] border-b border-[var(--desk-border)] overflow-hidden">
                {product.image_url ? (
                  <img
                    src={product.image_url}
                    alt={product.name}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <Package
                    size={64}
                    className="text-[var(--desk-text-tertiary)] opacity-10"
                  />
                )}
                {product.sell_by_block && (
                  <div className="absolute top-4 left-4">
                    <span className="px-3 py-1 rounded-lg bg-[var(--desk-accent)] text-white text-[10px] font-black uppercase tracking-widest shadow-lg">
                      Batch Required
                    </span>
                  </div>
                )}
              </div>

              <div className="p-8 space-y-8">
                <div>
                  <h2 className="md-typescale-display-small font-bold text-[var(--desk-text-primary)] tracking-tight">
                    {product.name}
                  </h2>
                  <div className="flex items-center gap-2 mt-4">
                    <Chip
                      size="sm"
                      variant="secondary"
                      className="font-bold text-[10px] tracking-widest"
                    >
                      {product.category_name?.toUpperCase()}
                    </Chip>
                    <span className="w-1 h-1 rounded-full bg-[var(--desk-border-strong)]" />
                    <span className="md-typescale-body-small font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                      {product.supplier_name}
                    </span>
                  </div>
                </div>

                {product.description && (
                  <div className="p-5 bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)] rounded-2xl">
                    <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-3 block">
                      Logical Context
                    </span>
                    <p className="md-typescale-body-medium text-[var(--desk-text-secondary)] leading-relaxed">
                      {product.description}
                    </p>
                  </div>
                )}

                {hasVariants && (
                  <div className="space-y-4">
                    <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] block">
                      Protocol Variants
                    </span>
                    <div className="space-y-2">
                      {variants.map((v) => (
                        <VariantRow key={v.id} variant={v} product={product} />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Footer */}
            <div className="p-8 border-t border-[var(--desk-border)] bg-[var(--desk-surface-subtle)]">
              <div className="flex justify-between items-center mb-6">
                <div>
                  <p className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                    Base Node Price
                  </p>
                  <p className="md-typescale-title-large font-bold text-[var(--desk-text-primary)] tabular-nums">
                    {product.price.toLocaleString()}{" "}
                    <small className="text-xs opacity-40 uppercase">UZS</small>
                  </p>
                </div>
                {product.units_per_block && (
                  <div className="text-right">
                    <p className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                      Batch Density
                    </p>
                    <p className="md-typescale-body-medium font-bold text-[var(--desk-text-primary)]">
                      {product.units_per_block} UNITS
                    </p>
                  </div>
                )}
              </div>

              <Button
                onPress={() => {
                  addToCart(product);
                  onClose();
                }}
                className="w-full h-14 bg-[var(--desk-text-primary)] text-white font-bold rounded-2xl shadow-xl flex items-center justify-center gap-3 transition-all hover:scale-[1.02] active:scale-95"
              >
                <ShoppingCart size={20} />
                Stage Node for Procurement
                <ChevronRight size={20} />
              </Button>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}

function VariantRow({
  variant,
  product,
}: {
  variant: Variant;
  product: Product;
}) {
  const { addToCart } = useCart();

  return (
    <div className="flex items-center justify-between p-4 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] hover:border-[var(--desk-border-strong)] transition-all">
      <div className="flex-1 min-w-0">
        <span className="md-typescale-body-medium font-bold text-[var(--desk-text-primary)] block">
          {variant.size}
        </span>
        <div className="flex items-center gap-2 mt-0.5">
          <span className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-tighter">
            {variant.pack || "Standard Pack"}
          </span>
          {variant.weight_per_unit && (
            <>
              <span className="w-1 h-1 rounded-full bg-[var(--desk-border)]" />
              <span className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-tighter">
                {variant.weight_per_unit}
              </span>
            </>
          )}
        </div>
      </div>
      <div className="flex items-center gap-4">
        <span className="md-typescale-title-small font-bold text-[var(--desk-text-primary)] tabular-nums">
          {variant.price.toLocaleString()}
        </span>
        <button
          onClick={() =>
            addToCart({ ...product, price: variant.price, id: variant.id })
          }
          className="w-8 h-8 rounded-full bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] flex items-center justify-center active:scale-90 transition-all"
        >
          <Plus size={16} />
        </button>
      </div>
    </div>
  );
}

function Plus({ size, className }: { size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="3"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <line x1="12" y1="5" x2="12" y2="19" />
      <line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}
