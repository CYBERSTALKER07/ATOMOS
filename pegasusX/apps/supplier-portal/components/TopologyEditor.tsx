"use client";

import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import type {
  SupplierTopologyFactory,
  SupplierTopologyResponse,
  SupplierTopologyUpdateRequest,
  SupplierTopologyWarehouse,
} from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";

const api = createSupplierApi();

type WarehouseDraft = {
  key: string;
  warehouse_id?: string;
  name: string;
  lat: string;
  lng: string;
  coverage_radius_km: string;
  is_active: boolean;
  is_on_shift: boolean;
  transfer_mode: "TRUCK" | "INTERNAL";
  co_locate_with_factory_id: string;
};

type FactoryDraft = {
  key: string;
  factory_id?: string;
  name: string;
  lat: string;
  lng: string;
  is_active: boolean;
};

function warehouseDraftFromNode(node: SupplierTopologyWarehouse, key: string): WarehouseDraft {
  return {
    key,
    warehouse_id: node.warehouse_id,
    name: node.name,
    lat: String(node.lat),
    lng: String(node.lng),
    coverage_radius_km: String(node.coverage_radius_km || 50),
    is_active: node.is_active,
    is_on_shift: node.is_on_shift,
    transfer_mode: node.transfer_mode === "INTERNAL" ? "INTERNAL" : "TRUCK",
    co_locate_with_factory_id: node.co_locate_with_factory_id ?? "",
  };
}

function factoryDraftFromNode(node: SupplierTopologyFactory, key: string): FactoryDraft {
  return {
    key,
    factory_id: node.factory_id,
    name: node.name,
    lat: String(node.lat),
    lng: String(node.lng),
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
      const lat = parseCoordinate(draft.lat, `Warehouse ${index + 1} latitude`);
      const lng = parseCoordinate(draft.lng, `Warehouse ${index + 1} longitude`);
      if (lat < -90 || lat > 90) {
        throw new Error(`Warehouse ${index + 1}: latitude out of range`);
      }
      if (lng < -180 || lng > 180) {
        throw new Error(`Warehouse ${index + 1}: longitude out of range`);
      }
      const coverage = Number.parseFloat(draft.coverage_radius_km.trim());
      const body: SupplierTopologyUpdateRequest["warehouses"][number] = {
        name,
        lat,
        lng,
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
      return body;
    }),
    factories: factories.map((draft, index) => {
      const name = draft.name.trim();
      if (!name) {
        throw new Error(`Factory ${index + 1}: name is required`);
      }
      const lat = parseCoordinate(draft.lat, `Factory ${index + 1} latitude`);
      const lng = parseCoordinate(draft.lng, `Factory ${index + 1} longitude`);
      if (lat < -90 || lat > 90) {
        throw new Error(`Factory ${index + 1}: latitude out of range`);
      }
      if (lng < -180 || lng > 180) {
        throw new Error(`Factory ${index + 1}: longitude out of range`);
      }
      const body: SupplierTopologyUpdateRequest["factories"][number] = {
        name,
        lat,
        lng,
        is_active: draft.is_active,
      };
      if (draft.factory_id) {
        body.factory_id = draft.factory_id;
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
  const baseline = useMemo(() => draftsFromTopology(initial), [initial]);
  const [warehouses, setWarehouses] = useState(baseline.warehouses);
  const [factories, setFactories] = useState(baseline.factories);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
      setError(err instanceof Error ? err.message : "save_topology_failed");
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
          <h2 className="md-typescale-title-medium">Warehouses</h2>
          <button
            type="button"
            className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
            onClick={() =>
              setWarehouses((prev) => [
                ...prev,
                {
                  key: `new-wh-${Date.now()}`,
                  name: `Warehouse ${prev.length + 1}`,
                  lat: "41.2995",
                  lng: "69.2401",
                  coverage_radius_km: "50",
                  is_active: true,
                  is_on_shift: true,
                  transfer_mode: "TRUCK",
                  co_locate_with_factory_id: "",
                },
              ])
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
                  ? () => setWarehouses((prev) => prev.filter((row) => row.key !== warehouse.key))
                  : undefined
              }
            >
              <Field
                label="Name"
                value={warehouse.name}
                onChange={(value) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, name: value } : row)),
                  )
                }
              />
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <Field
                  label="Latitude"
                  value={warehouse.lat}
                  onChange={(value) =>
                    setWarehouses((prev) =>
                      prev.map((row) => (row.key === warehouse.key ? { ...row, lat: value } : row)),
                    )
                  }
                />
                <Field
                  label="Longitude"
                  value={warehouse.lng}
                  onChange={(value) =>
                    setWarehouses((prev) =>
                      prev.map((row) => (row.key === warehouse.key ? { ...row, lng: value } : row)),
                    )
                  }
                />
              </div>
              <Field
                label="Coverage radius (km)"
                value={warehouse.coverage_radius_km}
                onChange={(value) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, coverage_radius_km: value } : row)),
                  )
                }
              />
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <SelectField
                  label="Transfer mode"
                  value={warehouse.transfer_mode}
                  options={[
                    { value: "TRUCK", label: "Truck replenishment" },
                    { value: "INTERNAL", label: "Internal / co-located" },
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
                  label="Co-locate with factory"
                  value={warehouse.co_locate_with_factory_id}
                  options={[
                    { value: "", label: "None" },
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
              <ToggleRow
                label="Active"
                checked={warehouse.is_active}
                onChange={(checked) =>
                  setWarehouses((prev) =>
                    prev.map((row) => (row.key === warehouse.key ? { ...row, is_active: checked } : row)),
                  )
                }
              />
              <ToggleRow
                label="On shift"
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
          <h2 className="md-typescale-title-medium">Factories</h2>
          <button
            type="button"
            className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
            onClick={() =>
              setFactories((prev) => [
                ...prev,
                {
                  key: `new-fc-${Date.now()}`,
                  name: `Factory ${prev.length + 1}`,
                  lat: "41.3111",
                  lng: "69.2797",
                  is_active: true,
                },
              ])
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
              onRemove={() => setFactories((prev) => prev.filter((row) => row.key !== factory.key))}
            >
              <Field
                label="Name"
                value={factory.name}
                onChange={(value) =>
                  setFactories((prev) =>
                    prev.map((row) => (row.key === factory.key ? { ...row, name: value } : row)),
                  )
                }
              />
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <Field
                  label="Latitude"
                  value={factory.lat}
                  onChange={(value) =>
                    setFactories((prev) =>
                      prev.map((row) => (row.key === factory.key ? { ...row, lat: value } : row)),
                    )
                  }
                />
                <Field
                  label="Longitude"
                  value={factory.lng}
                  onChange={(value) =>
                    setFactories((prev) =>
                      prev.map((row) => (row.key === factory.key ? { ...row, lng: value } : row)),
                    )
                  }
                />
              </div>
              <ToggleRow
                label="Active"
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
