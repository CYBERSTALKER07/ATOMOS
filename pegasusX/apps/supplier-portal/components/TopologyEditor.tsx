"use client";

import { usePortalT } from "@/lib/i18n";
import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import type {
  OutOfStockPolicy,
  SupplierTopologyCoverageCity,
  SupplierTopologyFactory,
  SupplierTopologyInventorySeed,
  SupplierTopologyResponse,
  SupplierTopologyUpdateRequest,
  SupplierTopologyWarehouse,
} from "@pegasusx/types";
import { AUTH_COUNTRIES } from "@pegasusx/ui-kit/auth";
import { sessionMapCenter } from "@pegasusx/api-client";
import { CoverageCityChips } from "@/components/CoverageCityChips";
import { createSupplierApi } from "@/lib/api";
import { LocationPicker } from "@/components/LocationPicker";
import { useToast } from "@/components/Toast";

const api = createSupplierApi();

type InventorySeedDraft = {
  key: string;
  product_id: string;
  quantity: string;
};

type WarehouseDraft = {
  key: string;
  warehouse_id?: string;
  name: string;
  address: string;
  place_id?: string;
  lat: string;
  lng: string;
  coverage_radius_km: string;
  is_active: boolean;
  is_on_shift: boolean;
  transfer_mode: "TRUCK" | "INTERNAL";
  co_locate_with_factory_id: string;
  primary_factory_id: string;
  secondary_factory_id: string;
  assigned_factory_ids: string[];
  country_code: string;
  coverage_cities: SupplierTopologyCoverageCity[];
  default_out_of_stock_policy: OutOfStockPolicy;
  initial_inventory: InventorySeedDraft[];
};

type FactoryDraft = {
  key: string;
  factory_id?: string;
  name: string;
  address: string;
  place_id?: string;
  lat: string;
  lng: string;
  country_code: string;
  is_active: boolean;
};

function warehouseDraftFromNode(node: SupplierTopologyWarehouse, key: string): WarehouseDraft {
  const policy: OutOfStockPolicy =
    node.default_out_of_stock_policy === "ACCEPT_BACKORDER" ? "ACCEPT_BACKORDER" : "REJECT";
  return {
    key,
    warehouse_id: node.warehouse_id,
    name: node.name,
    address: node.address ?? "",
    place_id: node.place_id,
    lat: String(node.lat),
    lng: String(node.lng),
    coverage_radius_km: String(node.coverage_radius_km || 50),
    is_active: node.is_active,
    is_on_shift: node.is_on_shift,
    transfer_mode: node.transfer_mode === "INTERNAL" ? "INTERNAL" : "TRUCK",
    co_locate_with_factory_id: node.co_locate_with_factory_id ?? "",
    primary_factory_id: node.primary_factory_id ?? "",
    secondary_factory_id: node.secondary_factory_id ?? "",
    assigned_factory_ids: node.assigned_factory_ids ?? [],
    country_code: node.country_code ?? "",
    coverage_cities: node.coverage_cities ?? [],
    default_out_of_stock_policy: policy,
    initial_inventory: (node.initial_inventory ?? []).map((seed, index) => ({
      key: `${key}-seed-${index}`,
      product_id: seed.product_id,
      quantity: String(seed.quantity),
    })),
  };
}

function inventorySeedsFromDrafts(rows: InventorySeedDraft[]): SupplierTopologyInventorySeed[] {
  const out: SupplierTopologyInventorySeed[] = [];
  for (const row of rows) {
    const productId = row.product_id.trim();
    const quantity = Number.parseInt(row.quantity.trim(), 10);
    if (!productId || !Number.isFinite(quantity) || quantity <= 0) {
      continue;
    }
    out.push({ product_id: productId, quantity });
  }
  return out;
}

function factoryDraftFromNode(node: SupplierTopologyFactory, key: string): FactoryDraft {
  return {
    key,
    factory_id: node.factory_id,
    name: node.name,
    address: node.address ?? "",
    place_id: node.place_id,
    lat: String(node.lat),
    lng: String(node.lng),
    country_code: node.country_code ?? "",
    is_active: node.is_active,
  };
}

function draftsFromTopology(topology: SupplierTopologyResponse): {
  warehouses: WarehouseDraft[];
  factories: FactoryDraft[];
} {
  return {
    warehouses: topology.warehouses.map((node, index) =>
      warehouseDraftFromNode(node, node.warehouse_id || `wh-${index}`),
    ),
    factories: topology.factories.map((node, index) =>
      factoryDraftFromNode(node, node.factory_id || `fc-${index}`),
    ),
  };
}

function parseCoordinate(value: string, field: string): number {
  const parsed = Number.parseFloat(value.trim());
  if (!Number.isFinite(parsed)) {
    throw new Error(`${field} must be a number`);
  }
  return parsed;
}

function buildUpdateRequest(warehouses: WarehouseDraft[], factories: FactoryDraft[]): SupplierTopologyUpdateRequest {
  if (warehouses.length === 0) {
    throw new Error("At least one warehouse is required");
  }

  return {
    warehouses: warehouses.map((draft, index) => {
      const name = draft.name.trim();
      if (!name) {
        throw new Error(`Warehouse ${index + 1}: name is required`);
      }
      const lat = parseCoordinate(draft.lat, `Warehouse ${index + 1} location`);
      const lng = parseCoordinate(draft.lng, `Warehouse ${index + 1} location`);
      if (lat < -90 || lat > 90) {
        throw new Error(`Warehouse ${index + 1}: location out of range`);
      }
      if (lng < -180 || lng > 180) {
        throw new Error(`Warehouse ${index + 1}: location out of range`);
      }
      if (!draft.address.trim()) {
        throw new Error(`Warehouse ${index + 1}: address is required`);
      }
      const coverage = Number.parseFloat(draft.coverage_radius_km.trim());
      const body: SupplierTopologyUpdateRequest["warehouses"][number] = {
        name,
        lat,
        lng,
        address: draft.address.trim(),
        place_id: draft.place_id,
        coverage_radius_km: Number.isFinite(coverage) && coverage > 0 ? coverage : 50,
        is_active: draft.is_active,
        is_on_shift: draft.is_on_shift,
        transfer_mode: draft.transfer_mode,
      };
      if (draft.warehouse_id) {
        body.warehouse_id = draft.warehouse_id;
      }
      const coLocate = draft.co_locate_with_factory_id.trim();
      if (coLocate) {
        body.co_locate_with_factory_id = coLocate;
      }
      const primary = draft.primary_factory_id.trim();
      if (primary) {
        body.primary_factory_id = primary;
      }
      const secondary = draft.secondary_factory_id.trim();
      if (secondary) {
        body.secondary_factory_id = secondary;
      }
      if (draft.assigned_factory_ids.length > 0) {
        body.assigned_factory_ids = draft.assigned_factory_ids;
      }
      if (draft.country_code.trim()) {
        body.country_code = draft.country_code.trim().toUpperCase();
      }
      if (draft.coverage_cities.length > 0) {
        body.coverage_cities = draft.coverage_cities;
      }
      body.default_out_of_stock_policy = draft.default_out_of_stock_policy;
      const seeds = inventorySeedsFromDrafts(draft.initial_inventory);
      if (seeds.length > 0) {
        body.initial_inventory = seeds;
      }
      return body;
    }),
    factories: factories.map((draft, index) => {
      const name = draft.name.trim();
      if (!name) {
        throw new Error(`Factory ${index + 1}: name is required`);
      }
      const lat = parseCoordinate(draft.lat, `Factory ${index + 1} location`);
      const lng = parseCoordinate(draft.lng, `Factory ${index + 1} location`);
      if (lat < -90 || lat > 90) {
        throw new Error(`Factory ${index + 1}: location out of range`);
      }
      if (lng < -180 || lng > 180) {
        throw new Error(`Factory ${index + 1}: location out of range`);
      }
      if (!draft.address.trim()) {
        throw new Error(`Factory ${index + 1}: address is required`);
      }
      const body: SupplierTopologyUpdateRequest["factories"][number] = {
        name,
        lat,
        lng,
        address: draft.address.trim(),
        place_id: draft.place_id,
        is_active: draft.is_active,
      };
      if (draft.factory_id) {
        body.factory_id = draft.factory_id;
      }
      if (draft.country_code.trim()) {
        body.country_code = draft.country_code.trim().toUpperCase();
      }
      return body;
    }),
  };
}

type TopologyEditorProps = {
  initial: SupplierTopologyResponse;
  onSaved: (topology: SupplierTopologyResponse) => void;
};

export function TopologyEditor({ initial, onSaved }: TopologyEditorProps) {
  const t = usePortalT();
  const baseline = useMemo(() => draftsFromTopology(initial), [initial]);
  const [warehouses, setWarehouses] = useState(baseline.warehouses);
  const [factories, setFactories] = useState(baseline.factories);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { push } = useToast();

  const factoryOptions = useMemo(
    () =>
      factories
        .filter((factory) => factory.factory_id)
        .map((factory) => ({ id: factory.factory_id as string, name: factory.name || factory.factory_id as string })),
    [factories],
  );

  const reset = () => {
    const next = draftsFromTopology(initial);
    setWarehouses(next.warehouses);
    setFactories(next.factories);
    setError(null);
  };

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      const body = buildUpdateRequest(warehouses, factories);
      const updated = await api.updateSupplierTopology(body);
      onSaved(updated);
      const next = draftsFromTopology(updated);
      setWarehouses(next.warehouses);
      setFactories(next.factories);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.save_topology_failed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      {error ? (
        <div
          className="md-card p-3 md-typescale-body-medium"
          style={{ color: "var(--color-md-error)", borderColor: "var(--color-md-error)" }}
        >
          {error}
        </div>
      ) : null}

      <section className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="md-typescale-title-medium">{t("portal.nav.warehouses")}</h2>
          <button
            type="button"
            className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
            onClick={() =>
              setWarehouses((prev) => {
                const c = sessionMapCenter();
                return [
                  ...prev,
                  {
                    key: `new-wh-${Date.now()}`,
                    name: `Warehouse ${prev.length + 1}`,
                    address: "",
                    lat: c ? String(c.lat) : "",
                    lng: c ? String(c.lng) : "",
                    coverage_radius_km: "50",
                    is_active: true,
                    is_on_shift: true,
                    transfer_mode: "TRUCK",
                    co_locate_with_factory_id: "",
                    primary_factory_id: "",
                    secondary_factory_id: "",
                    assigned_factory_ids: [],
                    country_code: "",
                    coverage_cities: [],
                    default_out_of_stock_policy: "REJECT",
                    initial_inventory: [],
                  },
                ];
              })
            }
          >
            Add warehouse
          </button>
        </div>
        <div className="grid grid-cols-1 gap-4">
          {warehouses.map((warehouse, index) => (
            <NodeCard
              key={warehouse.key}
              title={warehouse.warehouse_id ? `Warehouse ${warehouse.name}` : `New warehouse ${index + 1}`}
              onRemove={
                warehouses.length > 1
                  ? () => {
                      const idx = warehouses.findIndex(w => w.key === warehouse.key);
                      setWarehouses((prev) => prev.filter((row) => row.key !== warehouse.key));
                      push(`Removed warehouse ${warehouse.name || "Untitled"}`, "info", {
                        label: t("supplier_portal.residual.text.undo"),
                        onClick: () => {
                          setWarehouses((prev) => {
                            const copy = [...prev];
                            copy.splice(idx, 0, warehouse);
                            return copy;
                          });
                        }
                      });
                    }
                  : undefined
              }
            >
              <Field
                label={t("supplier_portal.analytics.knowledge_graph.text.name")}
                value={warehouse.name}
                onChange={(value) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, name: value } : row)),
                  )
                }
              />
              <LocationPicker
                label={t("supplier_portal.residual.text.warehouse_address")}
                value={{
                  address: warehouse.address,
                  lat: warehouse.lat,
                  lng: warehouse.lng,
                  place_id: warehouse.place_id,
                }}
                onChange={(loc) =>
                  setWarehouses((prev) =>
                    prev.map((row) =>
                      row.key === warehouse.key
                        ? {
                            ...row,
                            address: loc.address,
                            lat: loc.lat,
                            lng: loc.lng,
                            place_id: loc.place_id,
                          }
                        : row,
                    ),
                  )
                }
              />
              <Field
                label={t("supplier_portal.residual.text.coverage_radius_km")}
                value={warehouse.coverage_radius_km}
                onChange={(value) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, coverage_radius_km: value } : row)),
                  )
                }
              />
              <SelectField
                label="Country"
                value={warehouse.country_code}
                options={[
                  { value: "", label: "Unset (covers any country until set)" },
                  ...AUTH_COUNTRIES.map((c) => ({ value: c.code, label: `${c.name} (${c.code})` })),
                ]}
                onChange={(value) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, country_code: value } : row)),
                  )
                }
              />
              <CoverageCityChips
                cities={warehouse.coverage_cities}
                onChange={(coverage_cities) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, coverage_cities } : row)),
                  )
                }
              />
              <SelectField
                label="Primary factory"
                value={warehouse.primary_factory_id}
                options={[
                  { value: "", label: t("supplier_portal.residual.text.none") },
                  ...factoryOptions.map((factory) => ({ value: factory.id, label: factory.name })),
                ]}
                onChange={(value) =>
                  setWarehouses((prev) =>
                    prev.map((row) =>
                      row.key === warehouse.key
                        ? {
                            ...row,
                            primary_factory_id: value,
                            assigned_factory_ids: value
                              ? Array.from(new Set([value, ...row.assigned_factory_ids.filter((id) => id !== value)]))
                              : row.assigned_factory_ids,
                          }
                        : row,
                    ),
                  )
                }
              />
              <SelectField
                label="Secondary factory"
                value={warehouse.secondary_factory_id}
                options={[
                  { value: "", label: t("supplier_portal.residual.text.none") },
                  ...factoryOptions.map((factory) => ({ value: factory.id, label: factory.name })),
                ]}
                onChange={(value) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, secondary_factory_id: value } : row)),
                  )
                }
              />
              <div className="space-y-1">
                <span className="md-typescale-label-medium">Assigned factories</span>
                <div className="flex flex-wrap gap-2">
                  {factoryOptions.map((factory) => {
                    const checked = warehouse.assigned_factory_ids.includes(factory.id);
                    return (
                      <label key={factory.id} className="inline-flex items-center gap-2 text-sm">
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() =>
                            setWarehouses((prev) =>
                              prev.map((row) => {
                                if (row.key !== warehouse.key) return row;
                                const next = checked
                                  ? row.assigned_factory_ids.filter((id) => id !== factory.id)
                                  : [...row.assigned_factory_ids, factory.id];
                                return { ...row, assigned_factory_ids: next };
                              }),
                            )
                          }
                        />
                        {factory.name}
                      </label>
                    );
                  })}
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <SelectField
                  label={t("supplier_portal.residual.text.transfer_mode")}
                  value={warehouse.transfer_mode}
                  options={[
                    { value: "TRUCK", label: t("supplier_portal.residual.text.truck_replenishment") },
                    { value: "INTERNAL", label: t("supplier_portal.residual.text.internal_co_located") },
                  ]}
                  onChange={(value) =>
                    setWarehouses((prev) =>
                      prev.map((row) =>
                        row.key === warehouse.key
                          ? { ...row, transfer_mode: value as "TRUCK" | "INTERNAL" }
                          : row,
                      ),
                    )
                  }
                />
                <SelectField
                  label={t("supplier_portal.residual.text.co_locate_with_factory")}
                  value={warehouse.co_locate_with_factory_id}
                  options={[
                    { value: "", label: t("supplier_portal.residual.text.none") },
                    ...factoryOptions.map((factory) => ({ value: factory.id, label: factory.name })),
                  ]}
                  onChange={(value) =>
                    setWarehouses((prev) =>
                      prev.map((row) =>
                        row.key === warehouse.key ? { ...row, co_locate_with_factory_id: value } : row,
                      ),
                    )
                  }
                />
              </div>
              <SelectField
                label={t("supplier_portal.residual.text.default_out_of_stock_policy")}
                value={warehouse.default_out_of_stock_policy}
                options={[
                  { value: "REJECT", label: t("supplier_portal.residual.text.reject_orders_when_out_of_stock") },
                  { value: "ACCEPT_BACKORDER", label: t("supplier_portal.residual.text.accept_backorders_when_out_of_stock") },
                ]}
                onChange={(value) =>
                  setWarehouses((prev) =>
                    prev.map((row) =>
                      row.key === warehouse.key
                        ? { ...row, default_out_of_stock_policy: value as OutOfStockPolicy }
                        : row,
                    ),
                  )
                }
              />
              <div className="space-y-2">
                <div className="flex items-center justify-between gap-3">
                  <div className="md-typescale-label-medium text-[var(--color-md-outline)]">
                    Starter inventory (optional)
                  </div>
                  <button
                    type="button"
                    className="md-btn md-btn-text md-typescale-label-large"
                    onClick={() =>
                      setWarehouses((prev) =>
                        prev.map((row) =>
                          row.key === warehouse.key
                            ? {
                                ...row,
                                initial_inventory: [
                                  ...row.initial_inventory,
                                  { key: `seed-${Date.now()}`, product_id: "", quantity: "0" },
                                ],
                              }
                            : row,
                        ),
                      )
                    }
                  >
                    Add SKU
                  </button>
                </div>
                {warehouse.initial_inventory.length === 0 ? (
                  <p className="md-typescale-body-small text-[var(--color-md-outline)]">
                    Seed opening stock when this warehouse is saved. Re-saving replaces seeded quantities for listed SKUs.
                  </p>
                ) : null}
                {warehouse.initial_inventory.map((seed) => (
                  <div key={seed.key} className="grid grid-cols-1 md:grid-cols-[1fr_120px_auto] gap-2 items-end">
                    <Field
                      label={t("supplier_portal.analytics.demand.signals.text.product_id")}
                      value={seed.product_id}
                      onChange={(value) =>
                        setWarehouses((prev) =>
                          prev.map((row) =>
                            row.key === warehouse.key
                              ? {
                                  ...row,
                                  initial_inventory: row.initial_inventory.map((item) =>
                                    item.key === seed.key ? { ...item, product_id: value } : item,
                                  ),
                                }
                              : row,
                          ),
                        )
                      }
                    />
                    <Field
                      label={t("warehouse_portal.inventory.inventory_stock_list.text.quantity")}
                      value={seed.quantity}
                      onChange={(value) =>
                        setWarehouses((prev) =>
                          prev.map((row) =>
                            row.key === warehouse.key
                              ? {
                                  ...row,
                                  initial_inventory: row.initial_inventory.map((item) =>
                                    item.key === seed.key ? { ...item, quantity: value } : item,
                                  ),
                                }
                              : row,
                          ),
                        )
                      }
                    />
                    <button
                      type="button"
                      className="md-btn md-btn-text md-typescale-label-large mb-1"
                      onClick={() =>
                        setWarehouses((prev) =>
                          prev.map((row) =>
                            row.key === warehouse.key
                              ? {
                                  ...row,
                                  initial_inventory: row.initial_inventory.filter((item) => item.key !== seed.key),
                                }
                              : row,
                          ),
                        )
                      }
                    >
                      Remove
                    </button>
                  </div>
                ))}
              </div>
              <ToggleRow
                label={t("retailer_desktop.stock.local_skus.text.active")}
                checked={warehouse.is_active}
                onChange={(checked) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, is_active: checked } : row)),
                  )
                }
              />
              <ToggleRow
                label={t("supplier_portal.residual.text.on_shift")}
                checked={warehouse.is_on_shift}
                onChange={(checked) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, is_on_shift: checked } : row)),
                  )
                }
              />
            </NodeCard>
          ))}
        </div>
      </section>

      <section className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="md-typescale-title-medium">{t("portal.nav.factories")}</h2>
          <button
            type="button"
            className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
            onClick={() =>
              setFactories((prev) => {
                const c = sessionMapCenter();
                return [
                ...prev,
                {
                  key: `new-fc-${Date.now()}`,
                  name: `Factory ${prev.length + 1}`,
                  address: "",
                  lat: c ? String(c.lat) : "",
                  lng: c ? String(c.lng) : "",
                  country_code: "",
                  is_active: true,
                },
              ];
              })
            }
          >
            Add factory
          </button>
        </div>
        <div className="grid grid-cols-1 gap-4">
          {factories.length === 0 ? (
            <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
              No factories yet. Add one to enable factory-home drivers and internal transfer lanes.
            </p>
          ) : null}
          {factories.map((factory, index) => (
            <NodeCard
              key={factory.key}
              title={factory.factory_id ? `Factory ${factory.name}` : `New factory ${index + 1}`}
              onRemove={() => {
                const idx = factories.findIndex(f => f.key === factory.key);
                setFactories((prev) => prev.filter((row) => row.key !== factory.key));
                push(`Removed factory ${factory.name || "Untitled"}`, "info", {
                  label: t("supplier_portal.residual.text.undo"),
                  onClick: () => {
                    setFactories((prev) => {
                      const copy = [...prev];
                      copy.splice(idx, 0, factory);
                      return copy;
                    });
                  }
                });
              }}
            >
              <Field
                label={t("supplier_portal.analytics.knowledge_graph.text.name")}
                value={factory.name}
                onChange={(value) =>
                  setFactories((prev) =>
                    prev.map((row) => (row.key === factory.key ? { ...row, name: value } : row)),
                  )
                }
              />
              <LocationPicker
                label={t("supplier_portal.residual.text.factory_address")}
                value={{
                  address: factory.address,
                  lat: factory.lat,
                  lng: factory.lng,
                  place_id: factory.place_id,
                }}
                onChange={(loc) =>
                  setFactories((prev) =>
                    prev.map((row) =>
                      row.key === factory.key
                        ? {
                            ...row,
                            address: loc.address,
                            lat: loc.lat,
                            lng: loc.lng,
                            place_id: loc.place_id,
                          }
                        : row,
                    ),
                  )
                }
              />
              <SelectField
                label="Country"
                value={factory.country_code}
                options={[
                  { value: "", label: "Unset" },
                  ...AUTH_COUNTRIES.map((c) => ({ value: c.code, label: `${c.name} (${c.code})` })),
                ]}
                onChange={(value) =>
                  setFactories((prev) =>
                    prev.map((row) => (row.key === factory.key ? { ...row, country_code: value } : row)),
                  )
                }
              />
              <ToggleRow
                label={t("retailer_desktop.stock.local_skus.text.active")}
                checked={factory.is_active}
                onChange={(checked) =>
                  setFactories((prev) =>
                    prev.map((row) => (row.key === factory.key ? { ...row, is_active: checked } : row)),
                  )
                }
              />
            </NodeCard>
          ))}
        </div>
      </section>

      <div className="flex flex-wrap gap-3">
        <button
          type="button"
          className="md-btn md-btn-filled md-typescale-label-large px-6 py-2"
          disabled={saving}
          onClick={() => void save()}
        >
          {saving ? "Saving…" : "Save topology"}
        </button>
        <button
          type="button"
          className="md-btn md-btn-outlined md-typescale-label-large px-6 py-2"
          disabled={saving}
          onClick={reset}
        >
          Reset
        </button>
      </div>
    </div>
  );
}

function NodeCard({
  title,
  children,
  onRemove,
}: {
  title: string;
  children: ReactNode;
  onRemove?: () => void;
}) {
  return (
    <div className="md-card p-4 space-y-3">
      <div className="flex items-center justify-between gap-3">
        <h3 className="md-typescale-title-small">{title}</h3>
        {onRemove ? (
          <button type="button" className="md-btn md-btn-text md-typescale-label-large" onClick={onRemove}>
            Remove
          </button>
        ) : null}
      </div>
      {children}
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label className="block">
      <div className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</div>
      <input
        className="md-input-outlined mt-1 w-full px-3 py-2"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}

function SelectField({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: Array<{ value: string; label: string }>;
  onChange: (value: string) => void;
}) {
  return (
    <label className="block">
      <div className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</div>
      <select
        className="md-input-outlined mt-1 w-full px-3 py-2"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        {options.map((option) => (
          <option key={option.value || "none"} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

function ToggleRow({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-2 md-typescale-body-medium">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      {label}
    </label>
  );
}
