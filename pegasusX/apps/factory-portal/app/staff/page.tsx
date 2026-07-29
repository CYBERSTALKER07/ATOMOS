'use client';

import Link from 'next/link';
import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { StaffList, type StaffMember } from '@/components/staff/StaffList';

export default function StaffPage() {
  const [staff, setStaff] = useState<StaffMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const res = await apiFetch('/v1/factory/staff');
      if (res.ok) {
        const data = await res.json();
        setStaff(data.staff || []);
      } else {
        setError(`Unable to load staff (${res.status}).`);
      }
    } catch {
      setError('Unable to load staff right now.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  useFactorySessionReconcile(() => {
    void load();
  });

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: payload => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) {
          return;
        }
        void load();
      },
    });

    return () => {
      unsubscribe();
    };
  }, [load]);

  return (
    <PageTransition>
      <PageChrome
        icon="staff"
        title="Factory staff"
        description="Operators and shift coverage registered for this factory node."
        loading={loading}
        skeletonVariant="table"
        error={error && staff.length === 0 ? error : null}
        empty={!loading && !error && staff.length === 0}
        emptyMessage="There are no staff members registered for this factory."
        actions={
          <button type="button" className="portal-btn portal-btn--ghost inline-flex items-center gap-1.5" onClick={() => void load()}>
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        <StaffList staff={staff} />
      </PageChrome>
    </PageTransition>
  );
}
