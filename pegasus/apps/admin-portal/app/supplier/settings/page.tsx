'use client';

import { useEffect, useState, useCallback } from 'react';
import { Button } from '@heroui/react';
import { useSupplierShift } from '@/hooks/useSupplierShift';

// ── Types ─────────────────────────────────────────────────────────────────

type DayWindow = { open: string; close: string };
type ScheduleKey = 'mon' | 'tue' | 'wed' | 'thu' | 'fri' | 'sat' | 'sun';
type OperatingSchedule = Partial<Record<ScheduleKey, DayWindow>>;

const DAY_LABELS: Record<ScheduleKey, string> = {
  mon: 'Monday',
  tue: 'Tuesday',
  wed: 'Wednesday',
  thu: 'Thursday',
  fri: 'Friday',
  sat: 'Saturday',
  sun: 'Sunday',
};
const DAY_KEYS: ScheduleKey[] = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'];

const DEFAULT_WINDOW: DayWindow = { open: '09:00', close: '18:00' };

// ── Component ─────────────────────────────────────────────────────────────

export default function SupplierSettingsPage() {
  const shift = useSupplierShift();

  const [localSchedule, setLocalSchedule] = useState<OperatingSchedule>({});
  const [enabledDays, setEnabledDays] = useState<Set<ScheduleKey>>(new Set());
  const [localManualOff, setLocalManualOff] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saved' | 'error'>('idle');

  // Sync local form state when the shared context finishes loading
  useEffect(() => {
    if (shift.isLoading) return;
    setLocalManualOff(shift.manualOffShift);
    setLocalSchedule(shift.schedule);
    setEnabledDays(new Set(DAY_KEYS.filter((k) => k in shift.schedule)));
  }, [shift.isLoading, shift.manualOffShift, shift.schedule]);

  const toggleDay = useCallback((day: ScheduleKey) => {
    setEnabledDays((prev) => {
      const next = new Set(prev);
      if (next.has(day)) {
        next.delete(day);
      } else {
        next.add(day);
        setLocalSchedule((s) => ({ ...s, [day]: s[day] ?? DEFAULT_WINDOW }));
      }
      return next;
    });
  }, []);

  const updateWindow = useCallback((day: ScheduleKey, field: 'open' | 'close', value: string) => {
    setLocalSchedule((prev) => ({
      ...prev,
      [day]: { ...(prev[day] ?? DEFAULT_WINDOW), [field]: value },
    }));
  }, []);

  const handleSave = useCallback(async () => {
    if (isSaving) return;
    setIsSaving(true);
    setSaveStatus('idle');

    const finalSchedule: OperatingSchedule = {};
    for (const day of enabledDays) {
      finalSchedule[day] = localSchedule[day] ?? DEFAULT_WINDOW;
    }

    const result = await shift.updateShift(localManualOff, finalSchedule);
    if (result.ok) {
      setSaveStatus('saved');
      setTimeout(() => setSaveStatus('idle'), 3000);
    } else {
      setSaveStatus('error');
    }
    setIsSaving(false);
  }, [enabledDays, isSaving, localManualOff, localSchedule, shift]);

  if (shift.isLoading) {
    return (
      <div className="p-8 flex items-center justify-center" style={{ minHeight: 300 }}>
        <div
          className="rounded-full"
          style={{
            width: 32, height: 32,
            border: '3px solid var(--surface)',
            borderTopColor: 'var(--accent)',
            animation: 'spin 0.7s linear infinite',
          }}
        />
      </div>
    );
  }

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1
        className="md-typescale-headline-small mb-1"
        style={{ color: 'var(--foreground)' }}
      >
        Settings
      </h1>
      <p
        className="md-typescale-body-medium mb-8"
        style={{ color: 'var(--muted)' }}
      >
        Configure your business hours and shift availability.
      </p>

      {/* ── Shift Status Section ── */}
      <section
        className="md-card md-elevation-1 md-shape-md p-5 mb-6"
        style={{ background: 'var(--surface)' }}
      >
        <h2
          className="md-typescale-title-medium mb-1"
          style={{ color: 'var(--foreground)' }}
        >
          Shift Status
        </h2>
        <p
          className="md-typescale-body-small mb-4"
          style={{ color: 'var(--muted)' }}
        >
          Override your business hours. When off-shift, orders will not be processed
          until you return online.
        </p>

        <div className="flex items-center gap-4">
          {/* Toggle pill */}
          <button
            onClick={() => setLocalManualOff((v) => !v)}
            className="flex items-center gap-2 px-4 py-2 rounded-full md-typescale-label-large transition-all cursor-pointer"
            style={{
              background: !localManualOff
                ? 'color-mix(in srgb, var(--success) 15%, transparent)'
                : 'color-mix(in srgb, var(--danger) 15%, transparent)',
              color: !localManualOff ? 'var(--success)' : 'var(--danger)',
              border: `1px solid ${!localManualOff
                ? 'color-mix(in srgb, var(--success) 30%, transparent)'
                : 'color-mix(in srgb, var(--danger) 30%, transparent)'}`,
            }}
          >
            <span
              className="inline-block rounded-full"
              style={{
                width: 10, height: 10,
                background: !localManualOff ? 'var(--success)' : 'var(--danger)',
              }}
            />
            {localManualOff ? 'OFF SHIFT (forced)' : 'ON SHIFT'}
          </button>

          <span
            className="md-typescale-body-small"
            style={{ color: 'var(--muted)' }}
          >
            Current effective status:&nbsp;
            <strong style={{ color: shift.isActive ? 'var(--success)' : 'var(--danger)' }}>
              {shift.isActive ? 'OPEN' : 'CLOSED'}
            </strong>
          </span>
        </div>
      </section>

      {/* ── Business Hours Section ── */}
      <section
        className="md-card md-elevation-1 md-shape-md p-5 mb-6"
        style={{ background: 'var(--surface)' }}
      >
        <h2
          className="md-typescale-title-medium mb-1"
          style={{ color: 'var(--foreground)' }}
        >
          Business Hours
        </h2>
        <p
          className="md-typescale-body-small mb-5"
          style={{ color: 'var(--muted)' }}
        >
          Set open and close times for each day (Uzbekistan time, UTC+5). Unchecked
          days are treated as closed. Leave blank to remain open 24/7.
        </p>

        <div className="flex flex-col gap-3">
          {DAY_KEYS.map((day) => {
            const isEnabled = enabledDays.has(day);
            const window = localSchedule[day] ?? DEFAULT_WINDOW;
            return (
              <div
                key={day}
                className="flex items-center gap-3 py-2 px-3 rounded-lg transition-colors"
                style={{
                  background: isEnabled
                    ? 'var(--surface)'
                    : 'transparent',
                }}
              >
                {/* Enabled checkbox */}
                <input
                  type="checkbox"
                  id={`day-${day}`}
                  checked={isEnabled}
                  onChange={() => toggleDay(day)}
                  className="cursor-pointer"
                  style={{ width: 16, height: 16, accentColor: 'var(--accent)' }}
                />
                <label
                  htmlFor={`day-${day}`}
                  className="md-typescale-body-medium cursor-pointer"
                  style={{
                    color: isEnabled ? 'var(--foreground)' : 'var(--muted)',
                    width: 100,
                    flexShrink: 0,
                  }}
                >
                  {DAY_LABELS[day]}
                </label>

                {isEnabled ? (
                  <div className="flex items-center gap-2 flex-1">
                    <input
                      type="time"
                      value={window.open}
                      onChange={(e) => updateWindow(day, 'open', e.target.value)}
                      className="md-typescale-body-small px-2 py-1 rounded-md"
                      style={{
                        background: 'var(--background)',
                        color: 'var(--foreground)',
                        border: '1px solid var(--border)',
                        outline: 'none',
                      }}
                    />
                    <span
                      className="md-typescale-body-small"
                      style={{ color: 'var(--muted)' }}
                    >
                      to
                    </span>
                    <input
                      type="time"
                      value={window.close}
                      onChange={(e) => updateWindow(day, 'close', e.target.value)}
                      className="md-typescale-body-small px-2 py-1 rounded-md"
                      style={{
                        background: 'var(--background)',
                        color: 'var(--foreground)',
                        border: '1px solid var(--border)',
                        outline: 'none',
                      }}
                    />
                  </div>
                ) : (
                  <span
                    className="md-typescale-body-small"
                    style={{ color: 'var(--muted)' }}
                  >
                    Closed
                  </span>
                )}
              </div>
            );
          })}
        </div>
      </section>

      {/* ── Save Button ── */}
      <div className="flex items-center gap-4">
        <Button
          variant="primary"
          onPress={handleSave}
          isDisabled={isSaving}
          className="md-typescale-label-large px-6 py-2.5"
        >
          {isSaving ? 'Saving…' : 'Save Changes'}
        </Button>

        {saveStatus === 'saved' && (
          <span
            className="md-typescale-body-small"
            style={{ color: 'var(--success)' }}
          >
            Saved successfully
          </span>
        )}
        {saveStatus === 'error' && (
          <span
            className="md-typescale-body-small"
            style={{ color: 'var(--danger)' }}
          >
            Failed to save. Try again.
          </span>
        )}
      </div>

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
      `}</style>
    </div>
  );
}
