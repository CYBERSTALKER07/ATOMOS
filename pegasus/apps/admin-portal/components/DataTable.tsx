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

  if (isLoading) {
    return (
      <div className={`glass-premium border-none overflow-hidden ${className || ''}`}>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-white/5 border-b border-white/10">
              <tr>
                {columns.map(col => (
                  <th key={col.id} className={`px-6 py-5 ${ALIGN_MAP[col.align || 'left']}`}>
                    <Skeleton className="h-3 w-16 rounded-full opacity-30" />
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {Array.from({ length: skeletonRows }).map((_, i) => (
                <tr key={i} className="border-b border-white/5 last:border-b-0">
                  {columns.map(col => (
                    <td key={col.id} className="px-6 py-6">
                      <Skeleton className="h-4 w-24 rounded-full opacity-20" />
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
    <div className={`glass-premium border-none overflow-hidden ${className || ''}`}>
      <Table 
        aria-label={ariaLabel} 
        variant={variant} 
        className="w-full"
        selectionMode={selectionMode !== 'none' ? selectionMode : undefined}
        selectedKeys={selectedKeys}
        onSelectionChange={onSelectionChange}
        sortDescriptor={sortDescriptor}
        onSortChange={onSortChange}
        onRowAction={onRowAction}
        removeWrapper
      >
        <Table.Header className="bg-white/5 backdrop-blur-md border-b border-white/10">
          {selectionMode === 'multiple' && (
            <Table.Column id="selection" width={48}>
              <Checkbox slot="selection" className="ml-2" />
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
                'px-6 py-5 md-typescale-label-medium font-bold text-desk-text-secondary uppercase tracking-widest',
                ALIGN_MAP[col.align || 'left'],
                col.hideBelow ? HIDE_MAP[col.hideBelow] : '',
              ].join(' ').trim()}
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
              className="group border-b border-white/5 last:border-b-0 hover:bg-white/[0.04] transition-all duration-300 cursor-pointer active-press"
            >
              {selectionMode === 'multiple' && (
                <Table.Cell className="px-6 py-4">
                  <Checkbox slot="selection" />
                </Table.Cell>
              )}
              {columns.map((col, colIndex) => (
                <Table.Cell
                  key={col.id}
                  className={[
                    'px-6 py-5 md-typescale-body-medium text-white transition-all duration-500',
                    ALIGN_MAP[col.align || 'left'],
                    col.hideBelow ? HIDE_MAP[col.hideBelow] : '',
                  ].join(' ').trim()}
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
        <div className="flex items-center justify-between px-8 py-6 border-t border-white/10 bg-white/[0.02]">
          <div className="flex items-center gap-8">
            {totalItems !== undefined && (
              <span className="md-typescale-label-medium text-desk-text-tertiary font-medium">
                {totalItems} <span className="opacity-50 font-normal ml-1">results found</span>
              </span>
            )}
            {pageSizeOptions && onPageSizeChange && (
              <div className="flex items-center gap-3">
                <span className="md-typescale-label-small text-desk-text-tertiary uppercase tracking-wider opacity-50">Show</span>
                <select
                  value={pageSize}
                  onChange={e => onPageSizeChange(Number(e.target.value))}
                  className="md-typescale-label-medium bg-black/20 border border-white/10 rounded-lg px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-desk-accent/20 transition-all cursor-pointer hover:bg-black/30"
                >
                  {pageSizeOptions.map(opt => (
                    <option key={opt} value={opt} className="bg-slate-900">{opt}</option>
                  ))}
                </select>
              </div>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button
              isIconOnly
              variant="light"
              onPress={() => page! > 1 && onPageChange!(page! - 1)}
              disabled={page === 1}
              className="w-10 h-10 rounded-xl hover:bg-white/10 transition-colors disabled:opacity-30 active-press"
            >
              <span className="material-symbols-outlined text-xl text-desk-text-primary">chevron_left</span>
            </Button>
            
            <div className="flex items-center gap-1.5 mx-2">
              {Array.from({ length: Math.min(totalPages!, 5) }, (_, i) => {
                const p = i + 1;
                return (
                  <Button
                    key={p}
                    isIconOnly
                    variant={p === page ? "solid" : "light"}
                    color={p === page ? "primary" : "default"}
                    onPress={() => onPageChange!(p)}
                    className={`w-10 h-10 rounded-xl font-bold transition-all duration-500 ${
                      p === page 
                        ? "bg-desk-accent text-white shadow-xl shadow-desk-accent/20 scale-110" 
                        : "hover:bg-white/10 text-desk-text-secondary"
                    }`}
                  >
                    {p}
                  </Button>
                );
              })}
            </div>

            <Button
              isIconOnly
              variant="light"
              onPress={() => page! < totalPages! && onPageChange!(page! + 1)}
              disabled={page === totalPages}
              className="w-10 h-10 rounded-xl hover:bg-white/10 transition-colors disabled:opacity-30 active-press"
            >
              <span className="material-symbols-outlined text-xl text-desk-text-primary">chevron_right</span>
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
