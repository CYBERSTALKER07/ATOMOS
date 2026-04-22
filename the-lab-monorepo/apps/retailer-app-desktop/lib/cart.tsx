'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';

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
  total: number;
};

const CartContext = createContext<CartContextType | undefined>(undefined);

export function CartProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<CartItem[]>([]);

  // Load from local storage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem('retailer_cart');
      if (stored) {
        try {
          setItems(JSON.parse(stored));
        } catch { }
      }
    }
  }, []);

  // Save to local storage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('retailer_cart', JSON.stringify(items));
    }
  }, [items]);

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

  const clearCart = () => setItems([]);

  const total = items.reduce((sum, item) => sum + item.price * item.quantity, 0);

  return (
    <CartContext.Provider
      value={{
        items,
        addToCart,
        removeFromCart,
        updateQuantity,
        clearCart,
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