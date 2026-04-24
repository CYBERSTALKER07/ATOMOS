import { describe, it, expect } from 'vitest';
import { buildManifest, type LiveOrder } from '../utils/manifest';

// ── buildManifest ──────────────────────────────────────────────────────────

describe('buildManifest', () => {
  it('returns empty array for empty orders', () => {
    expect(buildManifest([])).toEqual([]);
  });

  it('returns empty array when orders have no items', () => {
    const orders: LiveOrder[] = [
      { order_id: 'ORD-1', retailer_id: 'R1', amount: 100000, payment_gateway: 'GLOBAL_PAY', state: 'LOADED' },
    ];
    expect(buildManifest(orders)).toEqual([]);
  });

  it('returns empty array when items is undefined', () => {
    const orders: LiveOrder[] = [
      { order_id: 'ORD-1', retailer_id: 'R1', amount: 100000, payment_gateway: 'CASH', state: 'LOADED', items: undefined },
    ];
    expect(buildManifest(orders)).toEqual([]);
  });

  it('flattens single order with single item', () => {
    const orders: LiveOrder[] = [{
      order_id: 'ORD-1',
      retailer_id: 'R1',
      amount: 50000,
      payment_gateway: 'CASH',
      state: 'LOADED',
      items: [{
        line_item_id: 'LI-1',
        sku_id: 'SKU-WATER',
        sku_name: 'Water 1.5L',
        quantity: 10,
        unit_price: 5000,
        status: 'PENDING',
      }],
    }];
    const result = buildManifest(orders);
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('LI-1');
    expect(result[0].orderId).toBe('ORD-1');
    expect(result[0].brand).toBe('SKU-WATER');
    expect(result[0].label).toBe('Water 1.5L × 10');
    expect(result[0].scanned).toBe(false);
  });

  it('flattens multiple items from single order', () => {
    const orders: LiveOrder[] = [{
      order_id: 'ORD-1',
      retailer_id: 'R1',
      amount: 100000,
      payment_gateway: 'GLOBAL_PAY',
      state: 'LOADED',
      items: [
        { line_item_id: 'LI-1', sku_id: 'SKU-A', sku_name: 'Product A', quantity: 5, unit_price: 10000, status: 'PENDING' },
        { line_item_id: 'LI-2', sku_id: 'SKU-B', sku_name: 'Product B', quantity: 3, unit_price: 20000, status: 'PENDING' },
      ],
    }];
    const result = buildManifest(orders);
    expect(result).toHaveLength(2);
    expect(result[0].id).toBe('LI-1');
    expect(result[1].id).toBe('LI-2');
  });

  it('flattens items across multiple orders', () => {
    const orders: LiveOrder[] = [
      {
        order_id: 'ORD-1', retailer_id: 'R1', amount: 50000, payment_gateway: 'CASH', state: 'LOADED',
        items: [{ line_item_id: 'LI-1', sku_id: 'SKU-A', sku_name: 'A', quantity: 1, unit_price: 50000, status: 'PENDING' }],
      },
      {
        order_id: 'ORD-2', retailer_id: 'R2', amount: 75000, payment_gateway: 'GLOBAL_PAY', state: 'LOADED',
        items: [
          { line_item_id: 'LI-2', sku_id: 'SKU-B', sku_name: 'B', quantity: 2, unit_price: 25000, status: 'PENDING' },
          { line_item_id: 'LI-3', sku_id: 'SKU-C', sku_name: 'C', quantity: 1, unit_price: 25000, status: 'PENDING' },
        ],
      },
    ];
    const result = buildManifest(orders);
    expect(result).toHaveLength(3);
    expect(result[0].orderId).toBe('ORD-1');
    expect(result[1].orderId).toBe('ORD-2');
    expect(result[2].orderId).toBe('ORD-2');
  });

  it('sets scanned to false for all items', () => {
    const orders: LiveOrder[] = [{
      order_id: 'ORD-1', retailer_id: 'R1', amount: 50000, payment_gateway: 'CASH', state: 'LOADED',
      items: [
        { line_item_id: 'LI-1', sku_id: 'SKU-A', sku_name: 'A', quantity: 1, unit_price: 50000, status: 'PENDING' },
        { line_item_id: 'LI-2', sku_id: 'SKU-B', sku_name: 'B', quantity: 2, unit_price: 25000, status: 'PENDING' },
      ],
    }];
    const result = buildManifest(orders);
    result.forEach(item => expect(item.scanned).toBe(false));
  });

  it('formats label as name × quantity', () => {
    const orders: LiveOrder[] = [{
      order_id: 'ORD-1', retailer_id: 'R1', amount: 50000, payment_gateway: 'CASH', state: 'LOADED',
      items: [{ line_item_id: 'LI-1', sku_id: 'SKU-COLA', sku_name: 'Coca-Cola 2L', quantity: 24, unit_price: 9000, status: 'PENDING' }],
    }];
    const result = buildManifest(orders);
    expect(result[0].label).toBe('Coca-Cola 2L × 24');
  });

  it('uses sku_id as brand field', () => {
    const orders: LiveOrder[] = [{
      order_id: 'ORD-1', retailer_id: 'R1', amount: 10000, payment_gateway: 'CASH', state: 'LOADED',
      items: [{ line_item_id: 'LI-1', sku_id: 'SKU-TEA-GREEN', sku_name: 'Green Tea', quantity: 1, unit_price: 10000, status: 'PENDING' }],
    }];
    expect(buildManifest(orders)[0].brand).toBe('SKU-TEA-GREEN');
  });

  it('uses line_item_id as manifest item id', () => {
    const orders: LiveOrder[] = [{
      order_id: 'ORD-1', retailer_id: 'R1', amount: 10000, payment_gateway: 'CASH', state: 'LOADED',
      items: [{ line_item_id: 'LI-UNIQUE-999', sku_id: 'SKU-X', sku_name: 'X', quantity: 1, unit_price: 10000, status: 'PENDING' }],
    }];
    expect(buildManifest(orders)[0].id).toBe('LI-UNIQUE-999');
  });

  it('preserves order across items', () => {
    const orders: LiveOrder[] = [{
      order_id: 'ORD-1', retailer_id: 'R1', amount: 50000, payment_gateway: 'CASH', state: 'LOADED',
      items: [
        { line_item_id: 'LI-A', sku_id: 'A', sku_name: 'First', quantity: 1, unit_price: 10000, status: 'PENDING' },
        { line_item_id: 'LI-B', sku_id: 'B', sku_name: 'Second', quantity: 2, unit_price: 20000, status: 'PENDING' },
      ],
    }];
    const result = buildManifest(orders);
    expect(result[0].brand).toBe('A');
    expect(result[1].brand).toBe('B');
  });

  it('skips orders with empty items array', () => {
    const orders: LiveOrder[] = [
      { order_id: 'ORD-EMPTY', retailer_id: 'R1', amount: 0, payment_gateway: 'CASH', state: 'LOADED', items: [] },
      {
        order_id: 'ORD-HAS', retailer_id: 'R2', amount: 10000, payment_gateway: 'CASH', state: 'LOADED',
        items: [{ line_item_id: 'LI-1', sku_id: 'S', sku_name: 'S', quantity: 1, unit_price: 10000, status: 'PENDING' }],
      },
    ];
    const result = buildManifest(orders);
    expect(result).toHaveLength(1);
    expect(result[0].orderId).toBe('ORD-HAS');
  });
});
