'use client';

import type { FlowConfig } from '@/app/data/topicTypes';
import { FlowShell } from './FlowShell';

const ROLES = ['Supplier', 'Warehouse', 'Factory', 'Driver', 'Retailer', 'Payload'];
const SURFACES = ['Portal', 'Mobile', 'Desktop'];

type Props = { config?: FlowConfig };

export default function AppsMatrixFlow({ config }: Props) {
  const highlightRole = config?.highlightStep ?? 0;

  return (
    <FlowShell title="Role × surface parity">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[480px] border-collapse text-left text-xs font-mono">
          <thead>
            <tr className="border-b border-white/20 text-white/50">
              <th className="py-2 pr-4">Role</th>
              {SURFACES.map((s) => (
                <th key={s} className="px-2 py-2 text-center">
                  {s}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {ROLES.map((role, ri) => (
              <tr key={role} className={ri === highlightRole ? 'bg-white/10' : ''}>
                <td className="py-2 pr-4 uppercase">{role}</td>
                {SURFACES.map((_, si) => (
                  <td key={si} className="px-2 py-2 text-center">
                    <span
                      className={`inline-block h-2 w-2 ${
                        ri === highlightRole || si < 2 ? 'bg-white' : 'bg-white/30'
                      }`}
                    />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </FlowShell>
  );
}
