"use client";

import { useEffect } from "react";
import { X, Package, ShoppingCart, Building2, Layers, Info } from "lucide-react";
import { Button, Chip } from "@heroui/react";
import { useCart } from "../lib/cart";
import type { Product, Variant } from "../lib/types";

interface ProductDetailDrawerProps {
  product: Product | null;
  isOpen: boolean;
  onClose: () => void;
}

export default function ProductDetailDrawer({ product, isOpen, onClose }: ProductDetailDrawerProps) {
  const { addToCart } = useCart();

  useEffect(() => {
    if (!isOpen) return;
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") onClose(); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [isOpen, onClose]);

  if (!isOpen || !product) return null;

  const variants = product.variants ?? [];
  const hasVariants = variants.length > 0;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/40 z-40 animate-in fade-in duration-200"
        onClick={onClose}
      />

      {/* Drawer */}
      <div className="fixed inset-y-0 right-0 w-full max-w-lg z-50 bg-background border-l border-[var(--border)] shadow-2xl flex flex-col animate-in slide-in-from-right duration-200">
        {/* Header */}
        <div className="flex items-center justify-between px-6 h-14 border-b border-[var(--border)] shrink-0">
          <span className="md-typescale-title-medium font-semibold text-foreground">Product Details</span>
          <Button variant="ghost" isIconOnly onPress={onClose} className="w-8 h-8 min-w-0 text-muted">
            <X size={20} />
          </Button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto">
          {/* Image */}
          <div className="h-56 w-full flex items-center justify-center" style={{ background: "var(--surface)" }}>
            {product.image_url ? (
              <img src={product.image_url} alt={product.name} className="w-full h-full object-cover" />
            ) : (
              <Package size={48} style={{ color: "var(--muted)", opacity: 0.3 }} />
            )}
          </div>

          <div className="p-6 flex flex-col gap-6">
            {/* Title & Price */}
            <div>
              <h2 className="md-typescale-headline-small font-semibold text-foreground">{product.name}</h2>
              <p className="md-typescale-headline-medium font-bold text-foreground mt-1 tabular-nums">
                {product.price.toLocaleString()}
              </p>
            </div>

            {/* Meta chips */}
            <div className="flex flex-wrap gap-2">
              {product.category_name && (
                <Chip size="sm" variant="soft" color="default">
                  <Layers size={12} className="inline-block mr-1" />{product.category_name}
                </Chip>
              )}
              <Chip size="sm" variant="soft" color="default">
                <Building2 size={12} className="inline-block mr-1" />{product.supplier_name}
              </Chip>
              {product.sell_by_block && (
                <Chip size="sm" variant="soft" color="warning">
                  Sold by block ({product.units_per_block} units)
                </Chip>
              )}
            </div>

            {/* Description */}
            {product.description && (
              <div>
                <span className="md-typescale-label-small uppercase tracking-widest font-semibold flex items-center gap-1" style={{ color: "var(--muted)" }}>
                  <Info size={12} /> Description
                </span>
                <p className="md-typescale-body-medium text-foreground mt-2 leading-relaxed">{product.description}</p>
              </div>
            )}

            {/* Nutrition */}
            {product.nutrition && (
              <div>
                <span className="md-typescale-label-small uppercase tracking-widest font-semibold" style={{ color: "var(--muted)" }}>
                  Nutrition
                </span>
                <p className="md-typescale-body-small text-muted mt-1">{product.nutrition}</p>
              </div>
            )}

            {/* Variants */}
            {hasVariants && (
              <div>
                <span className="md-typescale-label-small uppercase tracking-widest font-semibold mb-3 block" style={{ color: "var(--muted)" }}>
                  Variants ({variants.length})
                </span>
                <div className="flex flex-col gap-2">
                  {variants.map((v) => (
                    <VariantRow key={v.id} variant={v} product={product} />
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Footer Action */}
        <div className="px-6 py-4 border-t border-[var(--border)] shrink-0">
          <Button
            variant="primary"
            onPress={() => {
              addToCart(product);
              onClose();
            }}
            className="md-btn md-btn-filled md-typescale-label-large w-full h-11 flex items-center gap-2 justify-center"
          >
            <ShoppingCart size={18} /> Add to Cart — {product.price.toLocaleString()}
          </Button>
        </div>
      </div>
    </>
  );
}

function VariantRow({ variant, product }: { variant: Variant; product: Product }) {
  const { addToCart } = useCart();

  return (
    <div className="flex items-center justify-between p-3 rounded-xl border border-[var(--border)] hover:bg-surface transition-colors">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="md-typescale-body-medium font-semibold text-foreground">{variant.size}</span>
          {variant.pack && (
            <Chip size="sm" variant="soft" color="default">{variant.pack}</Chip>
          )}
        </div>
        <div className="flex items-center gap-3 mt-1">
          {variant.weight_per_unit && (
            <span className="md-typescale-label-small text-muted">{variant.weight_per_unit}</span>
          )}
          {variant.pack_count > 0 && (
            <span className="md-typescale-label-small text-muted">{variant.pack_count} per pack</span>
          )}
        </div>
      </div>
      <div className="flex items-center gap-3 shrink-0">
        <span className="md-typescale-label-large font-bold tabular-nums text-foreground">
          {variant.price.toLocaleString()}
        </span>
        <Button
          isIconOnly
          variant="secondary"
          size="sm"
          className="bg-accent text-accent-foreground rounded-full w-8 h-8 min-w-0"
          onPress={() => addToCart({ ...product, price: variant.price, id: variant.id })}
        >
          +
        </Button>
      </div>
    </div>
  );
}
