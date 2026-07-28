import React from 'react';
import Link from 'next/link';
import type { SupplierManifestRow } from '@pegasusx/types';
import StatusBadge from '@/components/StatusBadge';

interface ManifestsTableProps {
  items: SupplierManifestRow[];
}

export function ManifestsTable({ items }: ManifestsTableProps) {
  return (
    <div className="md-card overflow-hidden">
      <table className="desk-table w-full">
        <thead>
          <tr className="border-b border-[var(--color-md-outline-variant)] text-[var(--color-md-outline)]">
            <th className="md-typescale-label-medium p-4 font-medium text-left">Manifest</th>
            <th className="md-typescale-label-medium p-4 font-medium text-left">Status</th>
            <th className="md-typescale-label-medium p-4 font-medium text-right">Orders</th>
            <th className="md-typescale-label-medium p-4 font-medium text-left">Driver</th>
          </tr>
        </thead>
        <tbody>
          {items.map((manifest) => (
            <tr
              key={manifest.manifest_id}
              className="border-b border-[var(--color-md-outline-variant)] last:border-0"
            >
              <td className="p-4 md-typescale-body-medium font-mono">
                <Link href={`/manifests/${manifest.manifest_id}`} className="text-[var(--color-md-primary)] underline">
                  {manifest.manifest_id}
                </Link>
              </td>
              <td className="p-4 md-typescale-body-medium"><StatusBadge state={manifest.status} /></td>
              <td className="p-4 md-typescale-body-medium text-right">{manifest.orders_count}</td>
              <td className="p-4 md-typescale-body-medium">{manifest.driver_name}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
