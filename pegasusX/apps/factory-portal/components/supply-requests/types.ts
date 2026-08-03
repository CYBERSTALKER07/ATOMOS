export interface SupplyRequestItem {
  item_id: string;
  product_id: string;
  requested_quantity: number;
  shipped_quantity?: number;
  received_quantity?: number;
  recommended_qty?: number;
  unit_volume_vu?: number;
}

export interface SupplyRequest {
  request_id: string;
  warehouse_id: string;
  warehouse_name?: string;
  supplier_id: string;
  state: string;
  priority: string;
  requested_delivery_date: string;
  total_volume_vu: number;
  notes: string;
  item_count?: number;
  items?: SupplyRequestItem[];
  created_at: string;
}
