'use client';

import { useMemo } from 'react';
import { Link2, Link2Off } from 'lucide-react';
import { Button } from '@heroui/react';

import { BentoCard } from '@/components/BentoGrid';
import type { ResolvedMappingLink } from './types';

interface BentoMappingCardProps {
  headers: string[];
  targetFields: string[];
  links: ResolvedMappingLink[];
  onAssign: (header: string, targetField: string | null, manual?: boolean) => void;
  onToggleIgnore: (header: string) => void;
}

const HIGH_CONFIDENCE_THRESHOLD = 0.9;

function lineColor(confidence: number): string {
  if (confidence < HIGH_CONFIDENCE_THRESHOLD) {
    return 'var(--color-md-warning)';
  }
  return 'var(--color-md-primary)';
}

export default function BentoMappingCard({
  headers,
  targetFields,
  links,
  onAssign,
  onToggleIgnore,
}: BentoMappingCardProps) {
  const rowHeight = 46;
  const canvasWidth = 840;
  const canvasHeight = Math.max(headers.length, targetFields.length) * rowHeight + 36;

  const linkMap = useMemo(() => {
    const map = new Map<string, ResolvedMappingLink>();
    for (const link of links) {
      map.set(link.sourceColumn, link);
    }
    return map;
  }, [links]);

  const targetIndex = useMemo(() => {
    const map = new Map<string, number>();
    targetFields.forEach((field, idx) => map.set(field, idx));
    return map;
  }, [targetFields]);

  return (
    <BentoCard size="anchor" className="p-0 overflow-hidden">
      <div className="px-5 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
        <h2 className="md-typescale-title-medium">AI Mapping Handshake</h2>
        <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
          Drag source headers to Pegasus fields or override assignments manually. Connections under 90% confidence are highlighted.
        </p>
      </div>

      <div className="overflow-x-auto p-4" style={{ background: 'var(--surface)' }}>
        <div className="relative" style={{ minWidth: `${canvasWidth}px`, height: `${canvasHeight}px` }}>
          <svg
            className="absolute inset-0 pointer-events-none"
            width={canvasWidth}
            height={canvasHeight}
            viewBox={`0 0 ${canvasWidth} ${canvasHeight}`}
            fill="none"
            aria-hidden="true"
          >
            {headers.map((header, sourceIdx) => {
              const link = linkMap.get(header);
              if (!link || !link.targetField || link.ignored) {
                return null;
              }
              const targetIdx = targetIndex.get(link.targetField);
              if (targetIdx === undefined) {
                return null;
              }

              const y1 = 24 + sourceIdx * rowHeight;
              const y2 = 24 + targetIdx * rowHeight;
              const x1 = 304;
              const x2 = 544;
              const color = lineColor(link.confidence);

              return (
                <g key={`${header}-${link.targetField}`}>
                  <path
                    d={`M ${x1} ${y1} C ${x1 + 84} ${y1}, ${x2 - 84} ${y2}, ${x2} ${y2}`}
                    stroke={color}
                    strokeWidth={link.confidence < HIGH_CONFIDENCE_THRESHOLD ? 2.5 : 2}
                    strokeDasharray={link.confidence < HIGH_CONFIDENCE_THRESHOLD ? '4 4' : undefined}
                    opacity={0.95}
                  />
                  <circle cx={x1} cy={y1} r={3} fill={color} />
                  <circle cx={x2} cy={y2} r={3} fill={color} />
                </g>
              );
            })}
          </svg>

          <div className="absolute inset-0 grid grid-cols-[1fr_1fr] gap-10">
            <div className="px-2 py-2">
              <h3 className="md-typescale-label-large mb-3" style={{ color: 'var(--muted)' }}>Source Columns</h3>
              <div className="space-y-2">
                {headers.map((header) => {
                  const link = linkMap.get(header);
                  const ignored = Boolean(link?.ignored);
                  const confidence = link?.confidence ?? 0;

                  return (
                    <div
                      key={header}
                      draggable
                      onDragStart={(event) => {
                        event.dataTransfer.setData('text/plain', header);
                        event.dataTransfer.effectAllowed = 'move';
                      }}
                      className="h-10 px-3 flex items-center justify-between gap-2 md-shape-md"
                      style={{
                        border: `1px solid ${ignored ? 'var(--border)' : 'var(--color-md-primary-container)'}`,
                        background: ignored ? 'var(--surface)' : 'var(--color-md-primary-container)',
                        opacity: ignored ? 0.6 : 1,
                        cursor: 'grab',
                      }}
                    >
                      <div className="min-w-0">
                        <p className="md-typescale-body-small truncate" title={header}>{header}</p>
                      </div>

                      <div className="flex items-center gap-2">
                        {link?.targetField ? (
                          <span
                            className="md-chip"
                            style={{
                              cursor: 'default',
                              borderColor: confidence < HIGH_CONFIDENCE_THRESHOLD ? 'var(--color-md-warning)' : 'var(--color-md-primary)',
                              color: confidence < HIGH_CONFIDENCE_THRESHOLD ? 'var(--color-md-warning)' : 'var(--color-md-primary)',
                            }}
                          >
                            {(confidence * 100).toFixed(0)}%
                          </span>
                        ) : null}
                        <Button
                          variant="outline"
                          size="sm"
                          onPress={() => onToggleIgnore(header)}
                        >
                          {ignored ? <Link2 size={14} aria-hidden="true" /> : <Link2Off size={14} aria-hidden="true" />}
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            <div className="px-2 py-2">
              <h3 className="md-typescale-label-large mb-3" style={{ color: 'var(--muted)' }}>Pegasus Target Fields</h3>
              <div className="space-y-2">
                {targetFields.map((field) => (
                  <div
                    key={field}
                    onDragOver={(event) => {
                      event.preventDefault();
                      event.dataTransfer.dropEffect = 'move';
                    }}
                    onDrop={(event) => {
                      event.preventDefault();
                      const header = event.dataTransfer.getData('text/plain');
                      if (header) {
                        onAssign(header, field, true);
                      }
                    }}
                    className="h-10 px-3 flex items-center justify-between gap-2 md-shape-md"
                    style={{ border: '1px solid var(--border)', background: 'var(--background)' }}
                  >
                    <span className="md-typescale-body-small font-mono" title={field}>{field}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="px-5 pb-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {headers.map((header) => {
            const link = linkMap.get(header);
            return (
              <div key={`select-${header}`} className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--background)' }}>
                <div className="md-typescale-label-small mb-2" style={{ color: 'var(--muted)' }}>{header}</div>
                <select
                  value={link?.targetField || ''}
                  onChange={(event) => onAssign(header, event.target.value || null, true)}
                  disabled={Boolean(link?.ignored)}
                  className="md-input-outlined w-full"
                >
                  <option value="">Unmapped</option>
                  {targetFields.map((field) => (
                    <option key={`${header}-${field}`} value={field}>
                      {field}
                    </option>
                  ))}
                </select>
              </div>
            );
          })}
        </div>
      </div>
    </BentoCard>
  );
}
