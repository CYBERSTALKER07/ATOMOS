'use client';

import { Table, Pagination, Checkbox, Button } from '@heroui/react';
import type { SortDescriptor, Selection, Key } from 'react-aria-components';
import { motion, AnimatePresence } from 'framer-motion';
import { Skeleton } from './Skeleton';
import EmptyState from './EmptyState';

export interface Column<T> {
  id: string;
  header: React.ReactNode;
  cell: (item: T) => React.ReactNode;
  isRowHeader?: boolean;
  allowsSorting?: boolean;
  width?: number;
  minWidth?: number;
  maxWidth?: number;
  align?: 'left' | 'center' | 'right';
  hideBelow?: 'sm' | 'md' | 'lg';
}

interface DataTableProps<T extends { id: string }> {
  columns: Column<T>[];
  data: T[];
  ariaLabel?: string;
  isLoading?: boolean;
  skeletonRows?: number;

  // Empty state
  emptyIcon?: string;
  emptyHeadline?: string;
  emptyBody?: string;
  emptyAction?: string;
  onEmptyAction?: () => void;

  // Selection
  selectionMode?: 'none' | 'single' | 'multiple';
  selectedKeys?: Selection;
  onSelectionChange?: (keys: Selection) => void;

  // Sorting
  sortDescriptor?: SortDescriptor;
  onSortChange?: (descriptor: SortDescriptor) => void;

  // Row interaction
  onRowAction?: (key: Key) => void;

  // Pagination  
  page?: number;
  totalPages?: number;
  onPageChange?: (page: number) => void;
  pageSize?: number;
  totalItems?: number;
  pageSizeOptions?: number[];
  onPageSizeChange?: (size: number) => void;

  // Styling
  variant?: 'primary' | 'secondary';
  className?: string;
}

const ALIGN_MAP = { left: 'text-left', center: 'text-center', right: 'text-right' } as const;
const HIDE_MAP = { sm: 'hidden sm:table-cell', md: 'hidden md:table-cell', lg: 'hidden lg:table-cell' } as const;

export default function DataTable<T extends { id: string }>({
  columns,
  data,
  ariaLabel = 'Data table',
  isLoading = false,
  skeletonRows = 5,
  emptyIcon = 'orders',
  emptyHeadline = 'No data found',
  emptyBody,
  emptyAction,
  onEmptyAction,
  selectionMode = 'none',
  selectedKeys,
  onSelectionChange,
  sortDescriptor,
  onSortChange,
  onRowAction,
  page,
  totalPages,
  onPageChange,
  pageSize,
  totalItems,
  pageSizeOptions,
  onPageSizeChange,
  variant,
  className,
}: DataTableProps<T>) {
  const hasPagination = totalPages !== undefined && totalPages > 0 && onPageChange;

  const wrapperClass = `desk-card overflow-hidden ${className || ''}`;

  if (isLoading) {
    return (
      <div className={wrapperClass}>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead style={{ borderBottom: '1px solid var(--desk-border)' }}>
              <tr>
                {columns.map(col => (
                  <th key={col.id} className={`px-4 py-3 ${ALIGN_MAP[col.align || 'left']}`}>
                    <Skeleton className="h-3 w-16 rounded-sm" />
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {Array.from({ length: skeletonRows }).map((_, i) => (
                <tr key={i} style={{ borderBottom: '1px solid var(--desk-border)' }}>
                  {columns.map(col => (
                    <td key={col.id} className="px-4 py-4">
                      <Skeleton className="h-4 w-24 rounded-sm" />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    );
  }

  return (
    <div className={wrapperClass}>
      <Table 
        aria-label={ariaLabel} 
        variant={variant} 
        className="w-full desk-table"
        selectionMode={selectionMode !== 'none' ? selectionMode : undefined}
        selectedKeys={selectedKeys}
        onSelectionChange={onSelectionChange}
        sortDescriptor={sortDescriptor}
        onSortChange={onSortChange}
        onRowAction={onRowAction}
        removeWrapper
      >
        <Table.Header style={{ borderBottom: '1px solid var(--desk-border)' }}>
          {selectionMode === 'multiple' && (
            <Table.Column id="selection" width={48}>
              <Checkbox slot="selection" className="ml-2 desk-checkbox" />
            </Table.Column>
          )}
          {columns.map(col => (
            <Table.Column
              key={col.id}
              id={col.id}
              isRowHeader={col.isRowHeader}
              allowsSorting={col.allowsSorting}
              width={col.width}
              minWidth={col.minWidth}
              maxWidth={col.maxWidth}
              className={[
                'px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.08em]',
                ALIGN_MAP[col.align || 'left'],
                col.hideBelow ? HIDE_MAP[col.hideBelow] : '',
              ].join(' ').trim()}
              style={{ color: 'var(--desk-text-tertiary)' }}
            >
              {col.header}
            </Table.Column>
          ))}
        </Table.Header>
        <Table.Body
          items={data}
          renderEmptyState={() => (
            <div className="py-24">
              <EmptyState
                icon={emptyIcon as any}
                headline={emptyHeadline}
                body={emptyBody}
                action={emptyAction}
                onAction={onEmptyAction}
              />
            </div>
          )}
        >
          {(item) => (
            <Table.Row 
              id={item.id}
              className="group cursor-pointer transition-colors"
              style={{ borderBottom: '1px solid var(--desk-border)' }}
            >
              {selectionMode === 'multiple' && (
                <Table.Cell className="px-4 py-3">
                  <Checkbox slot="selection" className="desk-checkbox" />
                </Table.Cell>
              )}
              {columns.map((col, colIndex) => (
                <Table.Cell
                  key={col.id}
                  className={[
                    'px-4 py-3 text-[13px]',
                    ALIGN_MAP[col.align || 'left'],
                    col.hideBelow ? HIDE_MAP[col.hideBelow] : '',
                  ].join(' ').trim()}
                  style={{ color: 'var(--desk-text-primary)' }}
                >
                  <motion.div
                    initial={{ opacity: 0, x: -8 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ 
                      duration: 0.6,
                      delay: colIndex * 0.04,
                      ease: [0.16, 1, 0.3, 1]
                    }}
                  >
                    {col.cell(item)}
                  </motion.div>
                </Table.Cell>
              ))}
            </Table.Row>
          )}
        </Table.Body>
      </Table>

      {hasPagination && (
        <div
          className="flex items-center justify-between px-4 py-3"
          style={{ borderTop: '1px solid var(--desk-border)' }}
        >
          <div className="flex items-center gap-6">
            {totalItems !== undefined && (
              <span className="text-[12px]" style={{ color: 'var(--desk-text-tertiary)' }}>
                <span style={{ color: 'var(--desk-text-primary)', fontWeight: 600 }}>{totalItems}</span> results
              </span>
            )}
            {pageSizeOptions && onPageSizeChange && (
              <div className="flex items-center gap-2">
                <span className="text-[10px] uppercase tracking-[0.12em]" style={{ color: 'var(--desk-text-tertiary)' }}>Show</span>
                <select
                  value={pageSize}
                  onChange={e => onPageSizeChange(Number(e.target.value))}
                  className="text-[12px] px-2 py-1 rounded-md focus:outline-none cursor-pointer"
                  style={{
                    background: 'var(--desk-surface)',
                    border: '1px solid var(--desk-border)',
                    color: 'var(--desk-text-primary)',
                  }}
                >
                  {pageSizeOptions.map(opt => (
                    <option key={opt} value={opt}>{opt}</option>
                  ))}
                </select>
              </div>
            )}
          </div>
          <div className="flex items-center gap-1">
            <button
              onClick={() => page! > 1 && onPageChange!(page! - 1)}
              disabled={page === 1}
              className="desk-btn-ghost w-8 h-8 p-0 disabled:opacity-30 disabled:cursor-not-allowed"
              aria-label="Previous page"
            >
              <span className="material-symbols-outlined text-[18px]">chevron_left</span>
            </button>
            <div className="flex items-center gap-1 mx-1">
              {Array.from({ length: Math.min(totalPages!, 5) }, (_, i) => {
                const p = i + 1;
                const active = p === page;
                return (
                  <button
                    key={p}
                    onClick={() => onPageChange!(p)}
                    className={active ? 'desk-btn-primary w-8 h-8 p-0 text-[12px]' : 'desk-btn-ghost w-8 h-8 p-0 text-[12px]'}
                  >
                    {p}
                  </button>
                );
              })}
            </div>
            <button
              onClick={() => page! < totalPages! && onPageChange!(page! + 1)}
              disabled={page === totalPages}
              className="desk-btn-ghost w-8 h-8 p-0 disabled:opacity-30 disabled:cursor-not-allowed"
              aria-label="Next page"
            >
              <span className="material-symbols-outlined text-[18px]">chevron_right</span>
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
