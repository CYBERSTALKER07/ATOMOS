'use client';

import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { apiFetch, readToken } from './auth';
import { useOptionalWebSocket } from './ws';

export interface CartItem {
  product_id: string;
  supplier_id: string;
  name: string;
  price: number;
  quantity: number;
  image_url?: string;
}

interface AddToCartProduct {
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
      currency: 'UZS',
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
          currency: 'UZS',
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

export function CartProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<CartItem[]>([]);
  const [subtotal, setSubtotal] = useState(0);
  const [discount, setDiscount] = useState(0);
  const [total, setTotal] = useState(0);
  const quoteTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
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

  const addToCart = (product: AddToCartProduct, quantity = 1) => {
    setItems((prev) => {
      const existing = prev.find((i) => i.product_id === product.id);
      if (existing) {
        return prev.map((i) =>
          i.product_id === product.id ? { ...i, quantity: i.quantity + quantity } : i
        );
      }
      return [
        ...prev,
        {
          product_id: product.id,
          supplier_id: product.supplier_id,
          name: product.name,
          price: product.price,
          quantity,
          image_url: product.image_url,
        },
      ];
    });
  };

  const removeFromCart = (product_id: string) => {
    setItems((prev) => prev.filter((i) => i.product_id !== product_id));
  };

  const updateQuantity = (product_id: string, quantity: number) => {
    if (quantity <= 0) {
      removeFromCart(product_id);
      return;
    }
    setItems((prev) =>
      prev.map((i) => (i.product_id === product_id ? { ...i, quantity } : i))
    );
  };

  const clearCart = () => {
    setItems([]);
    setSubtotal(0);
    setDiscount(0);
    setTotal(0);
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