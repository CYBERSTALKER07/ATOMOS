import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import React from 'react';
import { CartProvider, useCart } from '../../lib/cart';

function wrapper({ children }: { children: React.ReactNode }) {
  return React.createElement(CartProvider, null, children);
}

describe('useCart', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('starts with empty cart', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    expect(result.current.items).toEqual([]);
    expect(result.current.total).toBe(0);
  });

  it('addToCart adds an item', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    act(() => {
      result.current.addToCart({
        id: 'p1',
        supplier_id: 's1',
        name: 'Milk',
        price: 10000,
      });
    });
    expect(result.current.items.length).toBe(1);
    expect(result.current.items[0].product_id).toBe('p1');
    expect(result.current.items[0].quantity).toBe(1);
    expect(result.current.items[0].price).toBe(10000);
  });

  it('addToCart increments existing product', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    const product = { id: 'p1', supplier_id: 's1', name: 'Milk', price: 10000 };
    act(() => result.current.addToCart(product));
    act(() => result.current.addToCart(product));
    expect(result.current.items.length).toBe(1);
    expect(result.current.items[0].quantity).toBe(2);
  });

  it('addToCart with custom quantity', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    act(() => {
      result.current.addToCart(
        { id: 'p1', supplier_id: 's1', name: 'Milk', price: 10000 },
        5
      );
    });
    expect(result.current.items[0].quantity).toBe(5);
  });

  it('removeFromCart removes item', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    act(() => {
      result.current.addToCart({ id: 'p1', supplier_id: 's1', name: 'A', price: 100 });
      result.current.addToCart({ id: 'p2', supplier_id: 's1', name: 'B', price: 200 });
    });
    act(() => result.current.removeFromCart('p1'));
    expect(result.current.items.length).toBe(1);
    expect(result.current.items[0].product_id).toBe('p2');
  });

  it('updateQuantity changes quantity', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    act(() => {
      result.current.addToCart({ id: 'p1', supplier_id: 's1', name: 'A', price: 100 });
    });
    act(() => result.current.updateQuantity('p1', 10));
    expect(result.current.items[0].quantity).toBe(10);
  });

  it('updateQuantity to 0 removes item', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    act(() => {
      result.current.addToCart({ id: 'p1', supplier_id: 's1', name: 'A', price: 100 });
    });
    act(() => result.current.updateQuantity('p1', 0));
    expect(result.current.items.length).toBe(0);
  });

  it('updateQuantity negative removes item', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    act(() => {
      result.current.addToCart({ id: 'p1', supplier_id: 's1', name: 'A', price: 100 });
    });
    act(() => result.current.updateQuantity('p1', -1));
    expect(result.current.items.length).toBe(0);
  });

  it('clearCart empties everything', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    act(() => {
      result.current.addToCart({ id: 'p1', supplier_id: 's1', name: 'A', price: 100 });
      result.current.addToCart({ id: 'p2', supplier_id: 's1', name: 'B', price: 200 });
    });
    act(() => result.current.clearCart());
    expect(result.current.items).toEqual([]);
    expect(result.current.total).toBe(0);
  });

  it('total sums price * quantity', () => {
    const { result } = renderHook(() => useCart(), { wrapper });
    act(() => {
      result.current.addToCart({ id: 'p1', supplier_id: 's1', name: 'A', price: 10000 }, 3);
      result.current.addToCart({ id: 'p2', supplier_id: 's1', name: 'B', price: 25000 }, 2);
    });
    // 30_000 + 50_000 = 80_000
    expect(result.current.total).toBe(80000);
  });

  it('useCart throws outside CartProvider', () => {
    expect(() => {
      renderHook(() => useCart());
    }).toThrow('useCart must be used within CartProvider');
  });
});
