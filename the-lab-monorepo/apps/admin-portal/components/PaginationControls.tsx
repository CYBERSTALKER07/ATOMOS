"use client";

import React from "react";
import { Button } from '@heroui/react';
import type { PaginationState } from "@/lib/usePagination";

interface Props {
  pagination: PaginationState<unknown>;
  pageSizeOptions?: number[];
}

export default function PaginationControls({ pagination, pageSizeOptions = [10, 25, 50, 100] }: Props) {
  const { page, totalPages, totalItems, pageSize, setPage, nextPage, prevPage, canNext, canPrev, setPageSize } = pagination;

  if (totalItems === 0) return null;

  return (
    <div className="flex items-center justify-between px-4 py-3" style={{ borderTop: '1px solid var(--border)' }}>
      <div className="flex items-center gap-2">
        <label className="md-typescale-label-small text-muted">Rows</label>
        <select
          value={pageSize}
          onChange={(e) => setPageSize(Number(e.target.value))}
          className="md-typescale-label-small px-2 py-1 rounded-md bg-surface text-foreground"
          style={{ border: '1px solid var(--border)' }}
        >
          {pageSizeOptions.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
      </div>

      <div className="flex items-center gap-3">
        <span className="md-typescale-label-small text-muted">
          {totalItems === 0 ? '0 items' : `${(page - 1) * pageSize + 1}–${Math.min(page * pageSize, totalItems)} of ${totalItems}`}
        </span>
        <div className="flex gap-1">
          <Button variant="ghost" isIconOnly onPress={() => setPage(1)} isDisabled={!canPrev} aria-label="First page" className="w-8 h-8 min-w-0 text-sm">⟨⟨</Button>
          <Button variant="ghost" isIconOnly onPress={prevPage} isDisabled={!canPrev} aria-label="Previous page" className="w-8 h-8 min-w-0 text-sm">⟨</Button>
          <span className="md-typescale-label-small px-2 py-1 text-foreground">
            {page} / {totalPages}
          </span>
          <Button variant="ghost" isIconOnly onPress={nextPage} isDisabled={!canNext} aria-label="Next page" className="w-8 h-8 min-w-0 text-sm">⟩</Button>
          <Button variant="ghost" isIconOnly onPress={() => setPage(totalPages)} isDisabled={!canNext} aria-label="Last page" className="w-8 h-8 min-w-0 text-sm">⟩⟩</Button>
        </div>
      </div>
    </div>
  );
}
