// ─── Types ────────────────────────────────────────────────────────────────────

export type LiveOrder = {
  order_id: string;
  retailer_id: string;
  amount: number;
  payment_gateway: string;
  state: string;
  route_id?: string | null;
  warehouse_id?: string;
  items?: {
    line_item_id: string;
    sku_id: string;
    sku_name: string;
    quantity: number;
    unit_price: number;
    status: string;
  }[];
};

export type ManifestItem = {
  id: string;
  orderId: string;
  brand: string;
  label: string;
  scanned: boolean;
};

// ─── Helpers ──────────────────────────────────────────────────────────────────

export function buildManifest(orders: LiveOrder[]): ManifestItem[] {
  return orders.flatMap(order =>
    (order.items || []).map(item => ({
      id: item.line_item_id,
      orderId: order.order_id,
      brand: item.sku_id,
      label: `${item.sku_name} × ${item.quantity}`,
      scanned: false,
    }))
  );
}
