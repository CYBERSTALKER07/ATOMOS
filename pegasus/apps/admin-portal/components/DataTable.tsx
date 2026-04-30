'use client';

import { Table, Pagination, Checkbox } from '@heroui/react';
import type { SortDescriptor, Selection, Key } from 'react-aria-components';
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
      <div className={`rounded-xl border border-border overflow-hidden ${className || ''}`}>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border">
                {columns.map(col => (
                  <th key={col.id} className={`px-4 py-3 ${ALIGN_MAP[col.align || 'left']}`}>
                    <Skeleton className="h-3 w-16 rounded" />
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {Array.from({ length: skeletonRows }).map((_, i) => (
                <tr key={i} className="border-b border-border last:border-b-0">
                  {columns.map(col => (
                    <td key={col.id} className="px-4 py-3">
                      <Skeleton className="h-3 w-24 rounded" />
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
    <Table aria-label={ariaLabel} variant={variant} className={className}>
      <Table.ScrollContainer>
        <Table.Content
          selectionMode={selectionMode !== 'none' ? selectionMode : undefined}
          selectedKeys={selectedKeys}
          onSelectionChange={onSelectionChange}
          sortDescriptor={sortDescriptor}
          onSortChange={onSortChange}
          onRowAction={onRowAction}
        >
          <Table.Header>
            {selectionMode === 'multiple' && (
              <Table.Column id="selection" width={40}>
                <Checkbox slot="selection" />
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
              <EmptyState
                icon={emptyIcon}
                headline={emptyHeadline}
                body={emptyBody}
                action={emptyAction}
                onAction={onEmptyAction}
              />
            )}
          >
            {(item) => (
              <Table.Row id={item.id}>
                {selectionMode === 'multiple' && (
                  <Table.Cell>
                    <Checkbox slot="selection" />
                  </Table.Cell>
                )}
                {columns.map(col => (
                  <Table.Cell
                    key={col.id}
                    className={[
                      ALIGN_MAP[col.align || 'left'],
                      col.hideBelow ? HIDE_MAP[col.hideBelow] : '',
                    ].join(' ').trim()}
                  >
                    {col.cell(item)}
                  </Table.Cell>
                ))}
              </Table.Row>
            )}
          </Table.Body>
        </Table.Content>
      </Table.ScrollContainer>

      {hasPagination && (
        <Table.Footer className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-3">
            {totalItems !== undefined && (
              <span className="md-typescale-label-small text-muted">
                {totalItems} item{totalItems !== 1 ? 's' : ''}
              </span>
            )}
            {pageSizeOptions && onPageSizeChange && (
              <select
                value={pageSize}
                onChange={e => onPageSizeChange(Number(e.target.value))}
                className="text-sm bg-transparent border border-border rounded-md px-2 py-1 text-foreground"
              >
                {pageSizeOptions.map(opt => (
                  <option key={opt} value={opt}>{opt} / page</option>
                ))}
              </select>
            )}
          </div>
          <Pagination>
            <Pagination.Content>
              <Pagination.Item>
                <Pagination.Previous onPress={() => page! > 1 && onPageChange!(page! - 1)}>
                  <Pagination.PreviousIcon />
                </Pagination.Previous>
              </Pagination.Item>
              {Array.from({ length: Math.min(totalPages!, 7) }, (_, i) => {
                const p = i + 1;
                return (
                  <Pagination.Item key={p}>
                    <Pagination.Link isActive={p === page} onPress={() => onPageChange!(p)}>
                      {p}
                    </Pagination.Link>
                  </Pagination.Item>
                );
              })}
              {totalPages! > 7 && (
                <>
                  <Pagination.Item>
                    <Pagination.Ellipsis />
                  </Pagination.Item>
                  <Pagination.Item>
                    <Pagination.Link
                      isActive={page === totalPages}
                      onPress={() => onPageChange!(totalPages!)}
                    >
                      {totalPages}
                    </Pagination.Link>
                  </Pagination.Item>
                </>
              )}
              <Pagination.Item>
                <Pagination.Next onPress={() => page! < totalPages! && onPageChange!(page! + 1)}>
                  <Pagination.NextIcon />
                </Pagination.Next>
              </Pagination.Item>
            </Pagination.Content>
          </Pagination>
        </Table.Footer>
      )}
    </Table>
  );
}
