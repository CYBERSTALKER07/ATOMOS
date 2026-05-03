'use client';

import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { apiFetch } from '@/lib/auth';
import { buildSupplierShiftIdempotencyKey } from '@/app/supplier/_shared/idempotency';

type DayWindow = { open: string; close: string };
type ScheduleKey = 'mon' | 'tue' | 'wed' | 'thu' | 'fri' | 'sat' | 'sun';
type OperatingSchedule = Partial<Record<ScheduleKey, DayWindow>>;

interface SupplierShiftState {
  isActive: boolean | null;
  manualOffShift: boolean;
  schedule: OperatingSchedule;
  isLoading: boolean;
  /** Quick shift toggle — flips manual_off_shift only */
  toggleShift: () => Promise<void>;
  /** Full shift update — manual_off_shift + schedule */
  updateShift: (manualOff: boolean, sched: OperatingSchedule) => Promise<{ ok: boolean }>;
}

const SupplierShiftContext = createContext<SupplierShiftState | null>(null);

export function SupplierShiftProvider({ children }: { children: ReactNode }) {
  const [isActive, setIsActive] = useState<boolean | null>(null);
  const [manualOffShift, setManualOffShift] = useState(false);
  const [schedule, setSchedule] = useState<OperatingSchedule>({});
  const [isLoading, setIsLoading] = useState(true);
  const [isToggling, setIsToggling] = useState(false);

  useEffect(() => {
    apiFetch('/v1/supplier/profile')
      .then((r) => (r.ok ? r.json() : null))
      .then(
        (data: {
          is_active?: boolean;
          manual_off_shift?: boolean;
          operating_schedule?: OperatingSchedule;
        } | null) => {
          if (!data) return;
          setIsActive(data.is_active ?? true);
          setManualOffShift(data.manual_off_shift ?? false);
          setSchedule(data.operating_schedule ?? {});
        },
      )
      .catch(() => {})
      .finally(() => setIsLoading(false));
  }, []);

  const toggleShift = useCallback(async () => {
    if (isToggling || isActive === null) return;
    setIsToggling(true);
    try {
      const payload = { manual_off_shift: isActive };
      const res = await apiFetch('/v1/supplier/shift', {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierShiftIdempotencyKey(payload),
        },
        body: JSON.stringify(payload),
      });
      if (res.ok) {
        const data: { is_active: boolean; manual_off_shift: boolean } = await res.json();
        setIsActive(data.is_active);
        setManualOffShift(data.manual_off_shift);
      }
    } finally {
      setIsToggling(false);
    }
  }, [isActive, isToggling]);

  const updateShift = useCallback(
    async (manualOff: boolean, sched: OperatingSchedule): Promise<{ ok: boolean }> => {
      try {
        const payload = {
          manual_off_shift: manualOff,
          operating_schedule: sched,
        };
        const res = await apiFetch('/v1/supplier/shift', {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
            'Idempotency-Key': buildSupplierShiftIdempotencyKey(payload),
          },
          body: JSON.stringify(payload),
        });
        if (res.ok) {
          const data: { is_active: boolean; manual_off_shift: boolean } = await res.json();
          setIsActive(data.is_active);
          setManualOffShift(data.manual_off_shift);
          setSchedule(sched);
          return { ok: true };
        }
        return { ok: false };
      } catch {
        return { ok: false };
      }
    },
    [],
  );

  return (
    <SupplierShiftContext.Provider
      value={{ isActive, manualOffShift, schedule, isLoading, toggleShift, updateShift }}
    >
      {children}
    </SupplierShiftContext.Provider>
  );
}

export function useSupplierShift(): SupplierShiftState {
  const ctx = useContext(SupplierShiftContext);
  if (!ctx) throw new Error('useSupplierShift must be used within SupplierShiftProvider');
  return ctx;
}
