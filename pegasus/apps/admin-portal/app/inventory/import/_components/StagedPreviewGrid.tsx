'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Sparkles } from 'lucide-react';
import { FixedSizeList, type ListChildComponentProps } from 'react-window';
import { Button } from '@heroui/react';

import { BentoCard, BentoSkeleton } from '@/components/BentoGrid';
import type { MappingAnomaly, ResolvedMappingLink, SupplierImportStagedRow } from './types';

interface StagedPreviewGridProps {
  rows: SupplierImportStagedRow[];
  links: ResolvedMappingLink[];
  anomalies: MappingAnomaly[];
  loading: boolean;
  hasMore: boolean;
  onLoadMore: () => void;
}

interface GridColumn {
  id: string;
  label: string;
  sourceColumn: string;
  targetField?: string | null;
}

const ROW_HEIGHT = 54;
const MIN_ROW_AREA_HEIGHT = 220;
const MAX_ROW_AREA_HEIGHT = 520;

function asRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, unknown>;
}

function stringifyValue(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'string') {
    return value.trim() || '—';
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  try {
    return JSON.stringify(value);
  } catch {
    return '—';
  }
}

export default function StagedPreviewGrid({
  rows,
  links,
  anomalies,
  loading,
  hasMore,
  onLoadMore,
}: StagedPreviewGridProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [containerWidth, setContainerWidth] = useState(920);

  const anomalyColumns = useMemo(() => {
    const set = new Set<string>();
    for (const anomaly of anomalies) {
      const key = (anomaly.column || '').toLowerCase().trim();
      if (key) set.add(key);
    }
    return set;
  }, [anomalies]);

  const columns = useMemo<GridColumn[]>(() => {
    const mapped = links.filter((link) => !link.ignored && link.targetField);
    if (mapped.length > 0) {
      return mapped.map((link) => ({
        id: `${link.sourceColumn}:${link.targetField || ''}`,
        label: link.targetField || link.sourceColumn,
        sourceColumn: link.sourceColumn,
        targetField: link.targetField,
      }));
    }

    const firstRow = rows[0];
    if (!firstRow) {
      return [];
    }

    const rawKeys = Object.keys(asRecord(firstRow.raw_data));
    return rawKeys.map((key) => ({
      id: key,
      label: key,
      sourceColumn: key,
    }));
  }, [links, rows]);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;

    const update = () => {
      setContainerWidth(element.clientWidth || 920);
    };

    update();
    const observer = new ResizeObserver(update);
    observer.observe(element);

    return () => observer.disconnect();
  }, []);

  const tableWidth = Math.max(containerWidth, 180 + columns.length * 210);
  const rowAreaHeight = Math.min(MAX_ROW_AREA_HEIGHT, Math.max(MIN_ROW_AREA_HEIGHT, rows.length * ROW_HEIGHT));
  const gridTemplate = `140px ${columns.map(() => 'minmax(180px, 1fr)').join(' ')}`;

  if (loading) {
    return (
      <BentoCard size="wide" className="p-5">
        <h2 className="md-typescale-title-medium mb-3">Staged Data Preview</h2>
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, idx) => (
            <BentoSkeleton key={`row-skeleton-${idx}`} size="wide" className="h-14" />
          ))}
        </div>
      </BentoCard>
    );
  }

  const itemData = {
    rows,
    columns,
    anomalyColumns,
    gridTemplate,
    tableWidth,
  };

  return (
    <BentoCard size="wide" className="p-5">
      <div className="flex items-center justify-between gap-3 mb-3 flex-wrap">
        <div>
          <h2 className="md-typescale-title-medium">Staged Data Preview</h2>
          <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
            Sticky headers and virtualized rows keep the browser responsive for large import sessions.
          </p>
        </div>
        {hasMore ? (
          <Button variant="outline" onPress={onLoadMore}>Load Next Batch</Button>
        ) : null}
      </div>

      <div
        ref={containerRef}
        className="overflow-x-auto md-shape-md"
        style={{ border: '1px solid var(--border)', background: 'var(--background)' }}
      >
        <div style={{ minWidth: `${tableWidth}px` }}>
          <div
            className="sticky top-0 z-10"
            style={{
              display: 'grid',
              gridTemplateColumns: gridTemplate,
              borderBottom: '1px solid var(--border)',
              background: 'var(--surface)',
            }}
          >
            <div className="px-3 py-2 md-typescale-label-small uppercase" style={{ color: 'var(--muted)' }}>Row</div>
            {columns.map((column: GridColumn) => (
              <div key={`header-${column.id}`} className="px-3 py-2 md-typescale-label-small uppercase" style={{ color: 'var(--muted)' }}>
                {column.label}
              </div>
            ))}
          </div>

          <FixedSizeList
            height={rowAreaHeight}
            itemCount={rows.length}
            itemSize={ROW_HEIGHT}
            width={tableWidth}
            itemData={itemData}
          >
            {VirtualizedRow}
          </FixedSizeList>
        </div>
      </div>
    </BentoCard>
  );
}

function VirtualizedRow({ index, style, data }: ListChildComponentProps<{ rows: SupplierImportStagedRow[]; columns: GridColumn[]; anomalyColumns: Set<string>; gridTemplate: string; tableWidth: number }>) {
  const row = data.rows[index];
  const raw = asRecord(row?.raw_data);
  const cleaned = asRecord(row?.cleaned_data);

  return (
    <div
      style={{ ...style, width: data.tableWidth }}
      className="border-b"
    >
      <div
        style={{ display: 'grid', gridTemplateColumns: data.gridTemplate, minHeight: ROW_HEIGHT, alignItems: 'center' }}
      >
        <div className="px-3 flex items-center gap-2 min-w-0">
          <span className="font-mono md-typescale-body-small" style={{ color: 'var(--muted)' }}>#{row.row_index + 1}</span>
          {row.is_new_product ? (
            <span
              className="md-chip"
              style={{
                cursor: 'default',
                borderColor: 'var(--color-md-primary)',
                color: 'var(--color-md-primary)',
                background: 'var(--color-md-primary-container)',
              }}
            >
              <Sparkles size={12} aria-hidden="true" /> New Product
            </span>
          ) : null}
        </div>

        {data.columns.map((column: GridColumn) => {
          const anomaly = data.anomalyColumns.has(column.sourceColumn.toLowerCase()) ||
            (column.targetField ? data.anomalyColumns.has(column.targetField.toLowerCase()) : false);
          const value = column.targetField && cleaned[column.targetField] !== undefined
            ? cleaned[column.targetField]
            : raw[column.sourceColumn];

          return (
            <div
              key={`${column.id}-${index}`}
              className="px-3 py-2 md-typescale-body-small truncate"
              title={stringifyValue(value)}
              style={anomaly ? {
                background: 'var(--color-md-error-container)',
                color: 'var(--foreground)',
              } : undefined}
            >
              {stringifyValue(value)}
            </div>
          );
        })}
      </div>
    </div>
  );
}
