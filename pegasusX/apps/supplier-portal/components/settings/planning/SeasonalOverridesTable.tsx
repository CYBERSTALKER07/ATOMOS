import React from 'react';
import type { SeasonalOverrideRow } from '@pegasusx/types';

interface SeasonalOverridesTableProps {
  overrides: SeasonalOverrideRow[];
}

export function SeasonalOverridesTable({ overrides }: SeasonalOverridesTableProps) {
  if (overrides.length === 0) {
    return (
      <p className="md-typescale-body-small mt-4" style={{ color: "var(--desk-text-secondary)" }}>
        No custom seasonal overrides yet.
      </p>
    );
  }

  return (
    <table className="desk-table w-full mt-4">
      <thead>
        <tr style={{ color: "var(--desk-text-secondary)" }}>
          <th className="md-typescale-label-medium p-3 text-left font-medium">Name</th>
          <th className="md-typescale-label-medium p-3 text-left font-medium">Template</th>
          <th className="md-typescale-label-medium p-3 text-left font-medium">Window</th>
          <th className="md-typescale-label-medium p-3 text-left font-medium">Status</th>
        </tr>
      </thead>
      <tbody>
        {overrides.map((row) => (
          <tr key={row.override_id} style={{ borderTop: "1px solid var(--desk-border)" }}>
            <td className="p-3 md-typescale-body-medium">{row.name || "—"}</td>
            <td className="p-3 md-typescale-body-medium font-mono text-sm">{row.template_id}</td>
            <td className="p-3 md-typescale-body-medium">
              {row.start_date} → {row.end_date}
            </td>
            <td className="p-3 md-typescale-body-medium">{row.is_active ? "Active" : "Inactive"}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
