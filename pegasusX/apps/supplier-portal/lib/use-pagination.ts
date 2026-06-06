import { useMemo, useState } from "react";

export function usePagination<T>(items: T[], pageSize: number) {
  const [page, setPage] = useState(0);
  const pageCount = Math.max(1, Math.ceil(items.length / pageSize));

  const currentPage = Math.min(page, pageCount - 1);

  const pageItems = useMemo(() => {
    const start = currentPage * pageSize;
    return items.slice(start, start + pageSize);
  }, [items, currentPage, pageSize]);

  return {
    page: currentPage,
    pageCount,
    pageItems,
    pageSize,
    setPage,
    next: () => setPage((value) => Math.min(value + 1, pageCount - 1)),
    prev: () => setPage((value) => Math.max(value - 1, 0)),
    reset: () => setPage(0),
  };
}
