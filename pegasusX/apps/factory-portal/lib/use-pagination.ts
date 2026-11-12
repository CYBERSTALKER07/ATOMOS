import { useMemo, useState } from "react";

export function usePagination<T>(items: T[], pageSize: number) {
  const [page, setPage] = useState(0);
  const safePageSize = Math.max(1, Math.floor(pageSize) || 1);
  const safeItems = Array.isArray(items) ? items : [];
  const pageCount = Math.max(1, Math.ceil(safeItems.length / safePageSize));
  const currentPage = Math.max(0, Math.min(page, pageCount - 1));

  const pageItems = useMemo(() => {
    const start = currentPage * safePageSize;
    return safeItems.slice(start, start + safePageSize);
  }, [safeItems, currentPage, safePageSize]);

  return {
    page: currentPage,
    pageCount,
    pageItems,
    pageSize: safePageSize,
    setPage: (p: number) => setPage(Math.max(0, Math.min(p, pageCount - 1))),
    next: () => setPage((value) => Math.min(value + 1, pageCount - 1)),
    prev: () => setPage((value) => Math.max(value - 1, 0)),
    reset: () => setPage(0),
  };
}
