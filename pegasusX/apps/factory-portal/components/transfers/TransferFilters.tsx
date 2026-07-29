import React from 'react';

interface TransferFiltersProps {
  stateFilter: string;
  setStateFilter: (filter: string) => void;
  stateFilters: string[];
}

export function TransferFilters({ stateFilter, setStateFilter, stateFilters }: TransferFiltersProps) {
  return (
    <div className="flex flex-wrap gap-2">
      {stateFilters.map((filter) => (
        <button
          key={filter}
          type="button"
          onClick={() => setStateFilter(filter)}
          className={`rounded-full border px-4 py-2 text-xs font-semibold uppercase tracking-[0.14em] transition-colors ${
            stateFilter === filter
              ? 'border-transparent bg-[var(--accent)] text-[var(--accent-foreground)]'
              : 'border-[var(--border)] bg-transparent text-[var(--muted)] hover:border-[var(--accent)] hover:text-[var(--foreground)]'
          }`}
        >
          {filter}
        </button>
      ))}
    </div>
  );
}
