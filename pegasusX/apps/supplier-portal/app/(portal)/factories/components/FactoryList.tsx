import type { SupplierTopologyFactory } from "@pegasusx/types";

interface FactoryListProps {
  factories: SupplierTopologyFactory[];
  onAddFirst: () => void;
}

export function FactoryList({ factories, onAddFirst }: FactoryListProps) {
  if (factories.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 gap-4">
        <p className="md-typescale-body-medium text-[var(--color-md-outline)]">Add your first factory to link production to warehouses.</p>
        <button type="button" className="md-btn md-btn-filled px-6 py-3" onClick={onAddFirst}>
          Add first factory
        </button>
      </div>
    );
  }

  return (
    <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
      {factories.map((f) => (
        <li key={f.factory_id || f.name} className="p-4 md-typescale-body-medium">
          <div className="font-medium">{f.name}</div>
          <div className="text-[var(--color-md-outline)] text-sm mt-1">
            {(f.address || "Coordinates on file").toString()} · {f.is_active ? "Active" : "Inactive"}
          </div>
        </li>
      ))}
    </ul>
  );
}
