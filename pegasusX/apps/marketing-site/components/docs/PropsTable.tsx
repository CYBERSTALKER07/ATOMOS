import type { PropRow } from "@/content/components/types";

type PropsTableProps = {
  props: PropRow[];
};

export function PropsTable({ props }: PropsTableProps) {
  return (
    <div className="overflow-x-auto rounded-xl border border-[var(--mkt-border)]">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-[var(--mkt-border)] text-[var(--mkt-text-tertiary)]">
            <th className="px-4 py-3 font-semibold">Prop</th>
            <th className="px-4 py-3 font-semibold">Type</th>
            <th className="px-4 py-3 font-semibold">Default</th>
            <th className="px-4 py-3 font-semibold">Description</th>
          </tr>
        </thead>
        <tbody>
          {props.map((row) => (
            <tr key={row.name} className="border-b border-[var(--mkt-border)] last:border-0">
              <td className="px-4 py-3 font-mono text-[var(--mkt-text)]">{row.name}</td>
              <td className="px-4 py-3 font-mono text-xs">{row.type}</td>
              <td className="px-4 py-3 font-mono text-xs text-[var(--mkt-text-tertiary)]">
                {row.default ?? "—"}
              </td>
              <td className="px-4 py-3 text-[var(--mkt-text-secondary)]">{row.description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
