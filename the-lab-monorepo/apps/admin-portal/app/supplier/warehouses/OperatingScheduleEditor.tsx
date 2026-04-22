'use client';

import React, { useState, useEffect } from 'react';

// ── Types ─────────────────────────────────────────────────────────────────

interface DayWindow {
  open: string;   // "HH:MM"
  close: string;  // "HH:MM"
}

type ScheduleMap = Record<string, DayWindow>;

/** New envelope format: { is_24h: bool, schedules: {...} } */
interface ScheduleEnvelope {
  is_24h: boolean;
  schedules: ScheduleMap;
}

interface Props {
  /** JSON string from backend — supports both legacy flat map and new envelope */
  value: string;
  onChange: (json: string) => void;
}

// ── Helpers ───────────────────────────────────────────────────────────────

const DAYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const;
const LABELS: Record<string, string> = {
  mon: 'Monday',
  tue: 'Tuesday',
  wed: 'Wednesday',
  thu: 'Thursday',
  fri: 'Friday',
  sat: 'Saturday',
  sun: 'Sunday',
};

export function parseSchedule(json: string): ScheduleEnvelope {
  try {
    const parsed = JSON.parse(json);
    if (typeof parsed === 'object' && parsed !== null) {
      // New envelope format
      if ('is_24h' in parsed && 'schedules' in parsed) {
        return { is_24h: !!parsed.is_24h, schedules: parsed.schedules || {} };
      }
      // Legacy flat map
      return { is_24h: false, schedules: parsed };
    }
  } catch { /* ignore */ }
  return { is_24h: false, schedules: {} };
}

export function serializeSchedule(envelope: ScheduleEnvelope): string {
  const clean: ScheduleMap = {};
  for (const [day, window] of Object.entries(envelope.schedules)) {
    if (window.open && window.close) {
      clean[day] = window;
    }
  }
  return JSON.stringify({ is_24h: envelope.is_24h, schedules: clean });
}

// ── Component ─────────────────────────────────────────────────────────────

export default function OperatingScheduleEditor({ value, onChange }: Props) {
  const [envelope, setEnvelope] = useState<ScheduleEnvelope>(() => parseSchedule(value));
  const [errors, setErrors] = useState<Record<string, string>>({});

  const schedule = envelope.schedules;
  const is24h = envelope.is_24h;

  // Sync outward whenever envelope changes
  useEffect(() => {
    onChange(serializeSchedule(envelope));
  }, [envelope]); // eslint-disable-line react-hooks/exhaustive-deps

  const toggle24h = () => {
    setEnvelope(prev => ({ ...prev, is_24h: !prev.is_24h }));
  };

  const isClosed = (day: string) => !(day in schedule);

  const toggleDay = (day: string) => {
    setEnvelope(prev => {
      const next = { ...prev.schedules };
      if (day in next) {
        delete next[day];
        setErrors(e => { const n = { ...e }; delete n[day]; return n; });
      } else {
        next[day] = { open: '09:00', close: '18:00' };
      }
      return { ...prev, schedules: next };
    });
  };

  const updateTime = (day: string, field: 'open' | 'close', val: string) => {
    setEnvelope(prev => {
      const next = { ...prev.schedules };
      next[day] = { ...next[day], [field]: val };
      if (next[day].open && next[day].close && next[day].close <= next[day].open) {
        setErrors(e => ({ ...e, [day]: 'Close must be after open' }));
      } else {
        setErrors(e => { const n = { ...e }; delete n[day]; return n; });
      }
      return { ...prev, schedules: next };
    });
  };

  const fieldStyle: React.CSSProperties = {
    background: 'var(--field-background)',
    color: 'var(--field-foreground)',
    border: '1px solid var(--field-border)',
    borderRadius: '6px',
  };

  return (
    <div className="space-y-1">
      <p className="md-typescale-label-medium mb-3" style={{ color: 'var(--muted)' }}>
        Operating Schedule (UTC+5)
      </p>

      {/* 24/7 Master Toggle */}
      <div
        className="flex items-center justify-between p-3 rounded-lg mb-3"
        style={{
          background: is24h
            ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
            : 'var(--surface)',
          border: `1px solid ${is24h ? 'var(--accent)' : 'var(--border)'}`,
        }}
      >
        <div className="flex items-center gap-2">
          <span className="md-typescale-label-medium" style={{ color: is24h ? 'var(--accent)' : 'var(--foreground)' }}>
            Perpetual Node (24/7)
          </span>
          {is24h && (
            <span
              className="md-typescale-label-small px-2 py-0.5 rounded-full"
              style={{ background: 'var(--accent-soft)', color: 'var(--accent)' }}
            >
              Always Open
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={toggle24h}
          className="w-10 h-6 rounded-full transition-colors relative"
          style={{ background: is24h ? 'var(--accent)' : 'var(--border)' }}
        >
          <div
            className="absolute top-1 w-4 h-4 rounded-full transition-transform"
            style={{ background: 'white', transform: is24h ? 'translateX(22px)' : 'translateX(4px)' }}
          />
        </button>
      </div>

      {/* Day-by-day schedule — grayed out when 24/7 is active */}
      <div
        className="space-y-2"
        style={{ opacity: is24h ? 0.4 : 1, pointerEvents: is24h ? 'none' : 'auto' }}
      >
        {DAYS.map(day => (
          <div key={day} className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => toggleDay(day)}
              className="w-20 text-left md-typescale-label-medium shrink-0"
              style={{ color: isClosed(day) ? 'var(--muted)' : 'var(--foreground)' }}
            >
              {LABELS[day]}
            </button>

            {isClosed(day) ? (
              <span className="md-typescale-label-small px-2 py-1 rounded" style={{ color: 'var(--danger)', background: 'color-mix(in srgb, var(--danger) 12%, transparent)' }}>
                Closed
              </span>
            ) : (
              <>
                <input
                  type="time"
                  value={schedule[day]?.open || ''}
                  onChange={e => updateTime(day, 'open', e.target.value)}
                  className="px-2 py-1.5 md-typescale-body-small w-28"
                  style={fieldStyle}
                />
                <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>to</span>
                <input
                  type="time"
                  value={schedule[day]?.close || ''}
                  onChange={e => updateTime(day, 'close', e.target.value)}
                  className="px-2 py-1.5 md-typescale-body-small w-28"
                  style={fieldStyle}
                />
              </>
            )}

            {errors[day] && (
              <span className="md-typescale-label-small" style={{ color: 'var(--danger)' }}>
                {errors[day]}
              </span>
            )}
          </div>
        ))}
      </div>

      <p className="md-typescale-label-small pt-2" style={{ color: 'var(--muted)' }}>
        {is24h
          ? 'Perpetual Node — valid target for midnight factory transfers and 24/7 dispatch.'
          : 'Click a day name to toggle closed/open. Schedule drives ETA prediction and shift automation.'
        }
      </p>
    </div>
  );
}
