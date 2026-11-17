import type {
  SupplierTopologyFactory,
  SupplierTopologyFactoryInput,
  SupplierTopologyWarehouse,
  SupplierTopologyWarehouseInput,
} from "@pegasusx/types";

export function warehouseToTopologyInput(w: SupplierTopologyWarehouse): SupplierTopologyWarehouseInput {
  return {
    warehouse_id: w.warehouse_id,
    name: w.name,
    address: w.address,
    place_id: w.place_id,
    lat: w.lat,
    lng: w.lng,
    coverage_radius_km: w.coverage_radius_km,
    is_active: w.is_active,
    is_on_shift: w.is_on_shift,
    transfer_mode: w.transfer_mode,
    co_locate_with_factory_id: w.co_locate_with_factory_id,
    primary_factory_id: w.primary_factory_id,
    secondary_factory_id: w.secondary_factory_id,
    assigned_factory_ids: w.assigned_factory_ids,
    country_code: w.country_code,
    coverage_cities: w.coverage_cities,
    default_out_of_stock_policy: w.default_out_of_stock_policy,
  };
}

export function factoryToTopologyInput(f: SupplierTopologyFactory): SupplierTopologyFactoryInput {
  return {
    factory_id: f.factory_id,
    name: f.name,
    address: f.address,
    place_id: f.place_id,
    lat: f.lat,
    lng: f.lng,
    country_code: f.country_code,
    is_active: f.is_active,
  };
}
