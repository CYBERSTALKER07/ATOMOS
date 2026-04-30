import { useMemo, useState } from "react";

export interface PaginationState<T> {
  page: number;
  pageSize: number;
  totalPages: number;
  totalItems: number;
  pageItems: T[];
  setPage: (p: number) => void;
  nextPage: () => void;
  prevPage: () => void;
  setPageSize: (size: number) => void;
  canNext: boolean;
  canPrev: boolean;
}

export function usePagination<T>(items: T[], defaultPageSize = 25): PaginationState<T> {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSizeRaw] = useState(defaultPageSize);

  const totalItems = items.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));

  // Clamp page when data shrinks
  const safePage = Math.min(page, totalPages);

  const pageItems = useMemo(() => {
    const start = (safePage - 1) * pageSize;
    return items.slice(start, start + pageSize);
  }, [items, safePage, pageSize]);

  const setPageSize = (size: number) => {
    setPageSizeRaw(size);
    setPage(1);
  };

  return {
    page: safePage,
    pageSize,
    totalPages,
    totalItems,
    pageItems,
    setPage,
    nextPage: () => setPage(Math.min(safePage + 1, totalPages)),
    prevPage: () => setPage(Math.max(safePage - 1, 1)),
    setPageSize,
    canNext: safePage < totalPages,
    canPrev: safePage > 1,
  };
}
