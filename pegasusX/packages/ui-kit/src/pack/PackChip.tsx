type PackChipProps = {
  currency?: string | null;
  receipts?: string | null;
  title?: string;
};

/** GS-R splash: currency + receipts label. Callers bind GET /v1/auth/session. */
export function PackChip({ currency, receipts, title }: PackChipProps) {
  if (!currency) return null;
  return (
    <span
      title={title}
      className="inline-flex items-center rounded-full border px-2 py-0.5 text-xs"
      data-testid="gs-r-pack-chip"
    >
      {currency}
      {receipts ? ` · receipts: ${receipts}` : ""}
    </span>
  );
}
