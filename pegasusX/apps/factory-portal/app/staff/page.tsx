'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PortalField, PortalInput, PortalSelect } from '@/components/portal';
import { StaffList, type StaffMember } from '@/components/staff/StaffList';

const ROLE_OPTIONS = [
  { value: 'FACTORY_OPERATOR', label: 'Factory Operator' },
  { value: 'FACTORY_ADMIN', label: 'Factory Admin' },
  { value: 'FACTORY_SUPERVISOR', label: 'Factory Supervisor' },
];

export default function StaffPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [staff, setStaff] = useState<StaffMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [formName, setFormName] = useState('');
  const [formRole, setFormRole] = useState('FACTORY_OPERATOR');
  const [submitting, setSubmitting] = useState(false);

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

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    const name = formName.trim();
    if (!name) {
      toast('Name is required', 'error');
      return;
    }
    setSubmitting(true);
    try {
      const res = await apiFetch('/v1/factory/staff', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, role: formRole }),
      });
      if (res.ok) {
        toast('Staff member created', 'success');
        setShowForm(false);
        setFormName('');
        setFormRole('FACTORY_OPERATOR');
        void load();
      } else {
        const data = await res.json().catch(() => ({}));
        toast(data.error || 'Failed to create staff member', 'error');
      }
    } catch {
      toast('Network error', 'error');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        icon="staff"
        title={t("factory_portal.staff.text.factory_staff")}
        description={t("factory_portal.residual.text.operators_and_shift_coverage_registered_for_this_factory_node")}
        loading={loading}
        skeletonVariant="table"
        error={error && staff.length === 0 ? error : null}
        empty={!loading && !error && staff.length === 0 && !showForm}
        emptyMessage={t("factory_portal.residual.text.there_are_no_staff_members_registered_for_this_factory")}
        actions={
          <div className="flex items-center gap-2">
            <button type="button" className="portal-btn portal-btn--ghost inline-flex items-center gap-1.5" onClick={() => void load()}>
              <Icon name="refresh" size={16} /> Refresh
            </button>
            <button
              type="button"
              className="portal-btn portal-btn--primary inline-flex items-center gap-1.5"
              onClick={() => setShowForm((v) => !v)}
            >
              <Icon name="add" size={16} />
              {showForm ? 'Cancel' : 'Add staff'}
            </button>
          </div>
        }
      >
        {showForm && (
          <form
            onSubmit={(e) => void handleCreate(e)}
            className="mb-6 rounded-xl border border-[var(--border)] p-5 space-y-4"
            style={{ background: 'var(--surface)' }}
          >
            <h2 className="text-sm font-semibold">{t("factory_portal.staff.text.create_staff_member")}</h2>
            <div className="grid gap-4 sm:grid-cols-2">
              <PortalField id="staff_name" label={t("supplier_portal.analytics.knowledge_graph.text.name")}>
                <PortalInput
                  id="staff_name"
                  value={formName}
                  onChange={(e) => setFormName(e.target.value)}
                  required
                  placeholder={t("factory_portal.staff.text.operator_name")}
                />
              </PortalField>
              <PortalField id="staff_role" label={t("supplier_portal.org_fleet.components.org_member_table.text.role")}>
                <PortalSelect id="staff_role" value={formRole} onChange={(e) => setFormRole(e.target.value)}>
                  {ROLE_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </PortalSelect>
              </PortalField>
            </div>
            <button
              type="submit"
              disabled={submitting}
              className="portal-btn portal-btn--primary disabled:opacity-50"
            >
              {submitting ? 'Creating…' : 'Create'}
            </button>
          </form>
        )}
        <StaffList staff={staff} />
      </PageChrome>
    </PageTransition>
  );
}
