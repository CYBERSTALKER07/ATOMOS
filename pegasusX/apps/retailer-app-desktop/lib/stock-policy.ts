import type { Product } from './types';

export interface StockAwareProduct {
  is_out_of_stock?: boolean;
  accepts_backorder?: boolean;
  available_stock?: number;
  show_stock_counts?: boolean;
  max_quantity?: number | null;
}

export function isCatalogBlocked(product: StockAwareProduct): boolean {
  if (product.is_out_of_stock) {
    return !product.accepts_backorder;
  }
  if (
    product.available_stock != null &&
    product.available_stock <= 0 &&
    !product.accepts_backorder
  ) {
    return true;
  }
  return false;
}

export function productMaxQuantity(product: StockAwareProduct): number | null {
  if (product.max_quantity != null && product.max_quantity > 0) {
    return product.max_quantity;
  }
  if (
    product.show_stock_counts &&
    product.available_stock != null &&
    product.available_stock > 0 &&
    !product.accepts_backorder
  ) {
    return product.available_stock;
  }
  if (
    product.available_stock != null &&
    product.available_stock > 0 &&
    !product.accepts_backorder &&
    product.is_out_of_stock !== true
  ) {
    return product.available_stock;
  }
  return null;
}

export function clampCartQuantity(
  product: StockAwareProduct,
  quantity: number,
): number {
  const max = productMaxQuantity(product);
  if (max != null && quantity > max) {
    return max;
  }
  return Math.max(1, quantity);
}

export function stockMetaFromProduct(product: Product): StockAwareProduct {
  return {
    is_out_of_stock: product.is_out_of_stock,
    accepts_backorder: product.accepts_backorder,
    available_stock: product.available_stock,
    show_stock_counts: product.show_stock_counts,
    max_quantity: product.max_quantity,
  };
}
