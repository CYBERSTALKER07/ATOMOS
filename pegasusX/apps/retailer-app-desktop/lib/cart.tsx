'use client';

import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { apiFetch, readToken } from './auth';
import { useOptionalWebSocket } from './ws';
import {
  clampCartQuantity,
  orderableCapsFromPreview,
  stockMetaFromProduct,
  type StockAwareProduct,
} from './stock-policy';
import { getRetailerId } from './retailer-profile';
import { packCurrency, readCachedAuthSession } from '@pegasusx/api-core';
import type { CheckoutPreviewResponse, Product } from './types';

function cartCurrency(): string {
  return packCurrency(readCachedAuthSession()?.pack);
}

export interface CartItem {
  product_id: string;
  supplier_id: string;
  name: string;
  price: number;
  quantity: number;
  image_url?: string;
  is_out_of_stock?: boolean;
  accepts_backorder?: boolean;
  available_stock?: number;
  show_stock_counts?: boolean;
  max_quantity?: number | null;
}

interface AddToCartProduct extends StockAwareProduct {
  id: string;
  supplier_id: string;
  name: string;
  price: number;
  image_url?: string;
}

type CartContextType = {
  items: CartItem[];
  addToCart: (product: AddToCartProduct, quantity?: number) => void;
  removeFromCart: (product_id: string) => void;
  updateQuantity: (product_id: string, quantity: number) => void;
  clearCart: () => void;
  subtotal: number;
  discount: number;
  total: number;
  previewOrderableQuantities: Record<string, number>;
  previewShowStockCounts: boolean;
  previewStockPolicyReject: boolean;
  checkoutPolicyToken: string | null;
  applyPreviewOrderableCaps: (preview: CheckoutPreviewResponse) => void;
};

const CartContext = createContext<CartContextType | undefined>(undefined);
const LOCAL_CART_KEY = 'retailer_cart';

interface ServerCartItem {
  cart_id?: string;
  sku_id: string;
  supplier_id: string;
  quantity: number;
  unit_price: number;
  currency: string;
}

interface ServerCartResponse {
  items: ServerCartItem[];
  total: number;
}

function buildCartSignature(items: CartItem[]): string {
  return items
    .slice()
    .sort((a, b) => a.product_id.localeCompare(b.product_id))
    .map((item) => `${item.product_id}:${item.supplier_id}:${item.quantity}:${item.price}`)
    .join('|');
}

function toServerCartItems(items: CartItem[]): ServerCartItem[] {
  return items
    .filter((item) => item.product_id && item.supplier_id && item.quantity > 0)
    .map((item) => ({
      sku_id: item.product_id,
      supplier_id: item.supplier_id,
      quantity: item.quantity,
      unit_price: Math.round(item.price),
      currency: cartCurrency(),
    }));
}

interface CheckoutQuoteResponse {
  supplier_id: string;
  subtotal_minor: number;
  discount_minor: number;
  total_minor: number;
  currency: string;
}

async function fetchPromotionTotals(cartItems: CartItem[]): Promise<{
  subtotal: number;
  discount: number;
  total: number;
}> {
  const lineSubtotal = cartItems.reduce((sum, item) => sum + item.price * item.quantity, 0);
  if (!readToken() || cartItems.length === 0) {
    return { subtotal: lineSubtotal, discount: 0, total: lineSubtotal };
  }

  const grouped = new Map<string, CartItem[]>();
  for (const item of cartItems) {
    if (!item.supplier_id) continue;
    const bucket = grouped.get(item.supplier_id) ?? [];
    bucket.push(item);
    grouped.set(item.supplier_id, bucket);
  }

  if (grouped.size === 0) {
    return { subtotal: lineSubtotal, discount: 0, total: lineSubtotal };
  }

  let subtotalMinor = 0;
  let discountMinor = 0;
  for (const [supplierId, supplierItems] of grouped.entries()) {
    const res = await apiFetch('/v1/retailer/checkout/quote', {
      method: 'POST',
      body: JSON.stringify({
        supplier_id: supplierId,
        lines: supplierItems.map((item) => ({
          product_id: item.product_id,
          quantity: item.quantity,
          unit_price_minor: Math.round(item.price),
          currency: cartCurrency(),
        })),
      }),
    });
    if (!res.ok) {
      return { subtotal: lineSubtotal, discount: 0, total: lineSubtotal };
    }
    const quote = (await res.json()) as CheckoutQuoteResponse;
    subtotalMinor += quote.subtotal_minor ?? 0;
    discountMinor += quote.discount_minor ?? 0;
  }

  const subtotal = subtotalMinor;
  const discount = discountMinor;
  return { subtotal, discount, total: Math.max(0, subtotal - discount) };
}

async function fetchCheckoutPreviewCaps(
  cartItems: CartItem[],
): Promise<CheckoutPreviewResponse | null> {
  const retailerId = getRetailerId();
  if (!retailerId || cartItems.length === 0) {
    return null;
  }
  const res = await apiFetch('/v1/checkout/preview', {
    method: 'POST',
    body: JSON.stringify({
      retailer_id: retailerId,
      latitude: 0,
      longitude: 0,
      items: cartItems.map((item) => ({
        sku_id: item.product_id,
        quantity: item.quantity,
        unit_price: Math.round(item.price),
      })),
    }),
  });
  if (!res.ok) {
    return null;
  }
  return (await res.json()) as CheckoutPreviewResponse;
}

export function CartProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<CartItem[]>([]);
  const [subtotal, setSubtotal] = useState(0);
  const [discount, setDiscount] = useState(0);
  const [total, setTotal] = useState(0);
  const [previewOrderableQuantities, setPreviewOrderableQuantities] = useState<
    Record<string, number>
  >({});
  const [previewShowStockCounts, setPreviewShowStockCounts] = useState(false);
  const [previewStockPolicyReject, setPreviewStockPolicyReject] = useState(true);
  const [checkoutPolicyToken, setCheckoutPolicyToken] = useState<string | null>(null);
  const quoteTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const ws = useOptionalWebSocket();
  const syncTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const skipNextSyncRef = useRef(false);
  const lastServerSignatureRef = useRef('');

  const hydrateFromServer = useCallback(async () => {
    if (!readToken()) {
      return;
    }
    try {
      const res = await apiFetch('/v1/retailer/cart/sync');
      if (!res.ok) {
        return;
      }

      const payload = (await res.json()) as ServerCartResponse;
      if (!Array.isArray(payload.items)) {
        return;
      }

      setItems((current) => {
        const bySku = new Map(current.map((item) => [item.product_id, item]));
        const merged = payload.items
          .filter((item) => item.sku_id && item.supplier_id && Number(item.quantity) > 0)
          .map((item) => {
            const existing = bySku.get(item.sku_id);
            return {
              product_id: item.sku_id,
              supplier_id: item.supplier_id,
              name: existing?.name ?? 'Item',
              price: Number(item.unit_price),
              quantity: Math.max(1, Number(item.quantity)),
              image_url: existing?.image_url,
            };
          });

        skipNextSyncRef.current = true;
        lastServerSignatureRef.current = buildCartSignature(merged);
        return merged;
      });
    } catch {
      // Fail open to local cart cache when server sync is unavailable.
    }
  }, []);

  const pushToServer = useCallback(async (nextItems: CartItem[], signature: string) => {
    if (!readToken()) {
      return;
    }
    try {
      const body = { items: toServerCartItems(nextItems) };
      const res = await apiFetch('/v1/retailer/cart/sync', {
        method: 'POST',
        body: JSON.stringify(body),
      });
      if (res.ok) {
        lastServerSignatureRef.current = signature;
      }
    } catch {
      // Keep local state and retry on the next mutation.
    }
  }, []);

  // Load local cache first, then hydrate from server when authenticated.
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem(LOCAL_CART_KEY);
      if (stored) {
        try {
          const parsed = JSON.parse(stored) as CartItem[];
          setItems(parsed);
          lastServerSignatureRef.current = buildCartSignature(parsed);
        } catch {
          // Ignore malformed local cache.
        }
      }
      void hydrateFromServer();
    }
  }, [hydrateFromServer]);

  // C1.3: clear cart in-memory when org is switched/selected.
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const onOrgSwitched = () => {
      skipNextSyncRef.current = true;
      setItems([]);
      lastServerSignatureRef.current = '';
      try {
        localStorage.removeItem(LOCAL_CART_KEY);
      } catch {
        /* ignore */
      }
    };
    window.addEventListener('pegasusx:org-switched', onOrgSwitched);
    return () => window.removeEventListener('pegasusx:org-switched', onOrgSwitched);
  }, []);

  // Refresh local cart when another device updates the same retailer cart.
  useEffect(() => {
    if (!ws) {
      return;
    }
    const unsubCart = ws.subscribe('CART_SYNC_UPDATED', () => {
      void hydrateFromServer();
    });
    const unsubPromo = ws.subscribe('PROMOTION_CHANGED', () => {
      void hydrateFromServer();
    });
    return () => {
      unsubCart();
      unsubPromo();
    };
  }, [hydrateFromServer, ws]);

  // Persist local cache and debounce full-cart server sync.
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(LOCAL_CART_KEY, JSON.stringify(items));
    }

    if (skipNextSyncRef.current) {
      skipNextSyncRef.current = false;
      return;
    }

    if (!readToken()) {
      return;
    }

    const signature = buildCartSignature(items);
    if (signature === lastServerSignatureRef.current) {
      return;
    }

    if (syncTimerRef.current) {
      clearTimeout(syncTimerRef.current);
    }
    syncTimerRef.current = setTimeout(() => {
      void pushToServer(items, signature);
    }, 250);

    return () => {
      if (syncTimerRef.current) {
        clearTimeout(syncTimerRef.current);
      }
    }
  }, [items, pushToServer]);

  const addToCart = (product: AddToCartProduct | Product, quantity = 1) => {
    const meta = stockMetaFromProduct(product as Product);
    const cappedQty = clampCartQuantity(meta, quantity, previewOrderableQuantities, previewStockPolicyReject);
    setItems((prev) => {
      const existing = prev.find((i) => i.product_id === product.id);
      if (existing) {
        const mergedMeta: StockAwareProduct = {
          is_out_of_stock: existing.is_out_of_stock ?? meta.is_out_of_stock,
          accepts_backorder: existing.accepts_backorder ?? meta.accepts_backorder,
          available_stock: meta.available_stock ?? existing.available_stock,
          show_stock_counts: meta.show_stock_counts ?? existing.show_stock_counts,
          max_quantity: meta.max_quantity ?? existing.max_quantity,
        };
        const nextQty = clampCartQuantity(
          { ...mergedMeta, product_id: product.id } as StockAwareProduct & { product_id: string },
          existing.quantity + cappedQty,
          previewOrderableQuantities,
          previewStockPolicyReject,
        );
        return prev.map((i) =>
          i.product_id === product.id
            ? {
                ...i,
                quantity: nextQty,
                ...mergedMeta,
              }
            : i,
        );
      }
      return [
        ...prev,
        {
          product_id: product.id,
          supplier_id: product.supplier_id,
          name: product.name,
          price: product.price,
          quantity: cappedQty,
          image_url: product.image_url,
          ...meta,
        },
      ];
    });
  };

  const removeFromCart = (product_id: string) => {
    setItems((prev) => prev.filter((i) => i.product_id !== product_id));
  };

  const applyPreviewOrderableCaps = useCallback((preview: CheckoutPreviewResponse) => {
    const caps = orderableCapsFromPreview(preview);
    if (!caps) return;
    setPreviewOrderableQuantities(caps);
    setPreviewShowStockCounts(Boolean(preview.show_stock_counts));
    setPreviewStockPolicyReject(preview.default_out_of_stock_policy !== "ACCEPT_BACKORDER");
    setCheckoutPolicyToken(preview.checkout_policy_token ?? null);
    setItems((prev) =>
      prev.map((item) => {
        if (preview.default_out_of_stock_policy === "ACCEPT_BACKORDER" || item.accepts_backorder) return item;
        const cap = caps[item.product_id];
        if (cap == null || item.quantity <= cap) return item;
        return { ...item, quantity: cap };
      }),
    );
  }, []);

  const updateQuantity = (product_id: string, quantity: number) => {
    if (quantity <= 0) {
      removeFromCart(product_id);
      return;
    }
    setItems((prev) =>
      prev.map((i) => {
        if (i.product_id !== product_id) {
          return i;
        }
        const nextQty = clampCartQuantity(i, quantity, previewOrderableQuantities, previewStockPolicyReject);
        return { ...i, quantity: nextQty };
      }),
    );
  };

  const clearCart = () => {
    setItems([]);
    setSubtotal(0);
    setDiscount(0);
    setTotal(0);
    setPreviewOrderableQuantities({});
    setPreviewShowStockCounts(false);
    setPreviewStockPolicyReject(true);
    setCheckoutPolicyToken(null);
  };

  useEffect(() => {
    const lineSubtotal = items.reduce((sum, item) => sum + item.price * item.quantity, 0);
    setSubtotal(lineSubtotal);
    setDiscount(0);
    setTotal(lineSubtotal);

    if (!readToken() || items.length === 0) {
      return;
    }

    if (quoteTimerRef.current) {
      clearTimeout(quoteTimerRef.current);
    }
    quoteTimerRef.current = setTimeout(() => {
      void fetchPromotionTotals(items)
        .then((quoted) => {
          setSubtotal(quoted.subtotal);
          setDiscount(quoted.discount);
          setTotal(quoted.total);
        })
        .catch(() => {
          setSubtotal(lineSubtotal);
          setDiscount(0);
          setTotal(lineSubtotal);
        });
    }, 300);
    return () => {
      if (quoteTimerRef.current) {
        clearTimeout(quoteTimerRef.current);
      }
    };
  }, [items]);

  useEffect(() => {
    if (!readToken() || items.length === 0) {
      setPreviewOrderableQuantities({});
      setPreviewShowStockCounts(false);
      return;
    }

    if (previewTimerRef.current) {
      clearTimeout(previewTimerRef.current);
    }
    previewTimerRef.current = setTimeout(() => {
      void fetchCheckoutPreviewCaps(items)
        .then((preview) => {
          if (preview) {
            applyPreviewOrderableCaps(preview);
          }
        })
        .catch(() => {
          // Keep last preview caps when refresh fails transiently.
        });
    }, 300);

    return () => {
      if (previewTimerRef.current) {
        clearTimeout(previewTimerRef.current);
      }
    };
  }, [items, applyPreviewOrderableCaps]);

  return (
    <CartContext.Provider
      value={{
        items,
        addToCart,
        removeFromCart,
        updateQuantity,
        clearCart,
        subtotal,
        discount,
        total,
        previewOrderableQuantities,
        previewShowStockCounts,
        previewStockPolicyReject,
        checkoutPolicyToken,
        applyPreviewOrderableCaps,
      }}
    >
      {children}
    </CartContext.Provider>
  );
}

export function useCart() {
  const ctx = useContext(CartContext);
  if (!ctx) throw new Error('useCart must be used within CartProvider');
  return ctx;
}