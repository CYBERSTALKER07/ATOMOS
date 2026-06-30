type SpecTableProps = {
  rows: { key: string; value: string }[];
  className?: string;
};

export function SpecTable({ rows, className = "" }: SpecTableProps) {
  return (
    <div className={`overflow-x-auto rounded-lg border border-[var(--mkt-border)] ${className}`.trim()}>
      <table className="mkt-spec-table">
        <thead>
          <tr>
            <th>Property</th>
            <th>Detail</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.key}>
              <td>{row.key}</td>
              <td>{row.value}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function BulletList({
  items,
  className = "",
}: {
  items: string[];
  className?: string;
}) {
  return (
    <ul className={`space-y-2 text-sm text-[var(--mkt-muted)] ${className}`.trim()}>
      {items.map((item) => (
        <li key={item} className="flex gap-3">
          <span className="mt-2 h-px w-3 shrink-0 bg-[var(--mkt-text)]" aria-hidden />
          {item}
        </li>
      ))}
    </ul>
  );
}

export function SectionHeader({
  label,
  title,
  description,
  titleId,
  platformFrame: _platformFrame = false,
}: {
  label: string;
  title: string;
  description?: string;
  titleId?: string;
  platformFrame?: boolean;
}) {
  return (
    <div className="max-w-3xl">
      <p className="void-tag mb-4">{label}</p>
      <h2 id={titleId} className="void-section-title">
        {title}
      </h2>
      {description ? (
        <p className="mt-4 text-[var(--mkt-muted)] md:text-lg leading-relaxed">{description}</p>
      ) : null}
    </div>
  );
}
