'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useToken } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import { useLocale } from '@/hooks/useLocale';
import { Button } from '@heroui/react';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface AuditLogRow {
  log_id: string;
  actor_id: string;
  actor_role: string;
  action: string;
  resource_type: string;
  resource_id: string;
  metadata: string;
  created_at: string;
}

interface AuditLogResponse {
  data: AuditLogRow[];
  limit: number;
  offset: number;
}

export default function AuditLogPage() {
  const token = useToken();
  const { toast } = useToast();
  const { locale, t } = useLocale();

  const [rows, setRows] = useState<AuditLogRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [resourceType, setResourceType] = useState('');
  const [action, setAction] = useState('');
  const [limit, setLimit] = useState(50);
  const [offset, setOffset] = useState(0);

  const query = useMemo(() => {
    const params = new URLSearchParams();
    params.set('limit', String(limit));
    params.set('offset', String(offset));
    if (resourceType.trim()) params.set('resource_type', resourceType.trim());
    if (action.trim()) params.set('action', action.trim());
    return params.toString();
  }, [action, limit, offset, resourceType]);

  const load = useCallback(async () => {
    if (!token) return;
    const loadFailedMessage = t('supplier_portal.admin.audit_log.error.load_failed');
    setLoading(true);
    try {
      const res = await fetch(`${API}/v1/admin/audit-log?${query}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error(loadFailedMessage);
      const payload = (await res.json()) as AuditLogResponse;
      setRows(payload.data || []);
    } catch (e) {
      toast(e instanceof Error ? e.message : loadFailedMessage, 'error');
    } finally {
      setLoading(false);
    }
  }, [query, t, token, toast]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto px-4 py-6">
      <div>
        <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
          {t('supplier_portal.admin.audit_log.title')}
        </h1>
        <p className="md-typescale-body-small mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
          {t('supplier_portal.admin.audit_log.subtitle')}
        </p>
      </div>

      <div className="md-card md-elevation-1 md-shape-md p-4 flex flex-wrap gap-3" style={{ background: 'var(--color-md-surface)' }}>
        <input className="md-input-outlined px-3 py-2" placeholder={t('supplier_portal.admin.audit_log.filter.resource_type')} value={resourceType} onChange={(e) => setResourceType(e.target.value)} />
        <input className="md-input-outlined px-3 py-2" placeholder={t('supplier_portal.admin.audit_log.filter.action')} value={action} onChange={(e) => setAction(e.target.value)} />
        <input className="md-input-outlined px-3 py-2 w-28" type="number" min="1" max="500" value={limit} onChange={(e) => setLimit(Number(e.target.value) || 50)} />
        <Button variant="outline" onPress={() => { setOffset(0); void load(); }}>
          {t('supplier_portal.admin.audit_log.action.apply')}
        </Button>
      </div>

      <div className="md-card md-elevation-1 md-shape-md overflow-hidden" style={{ background: 'var(--color-md-surface)' }}>
        {loading ? (
          <div className="p-6">{t('supplier_portal.admin.audit_log.state.loading')}</div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                <th className="text-left px-4 py-3">{t('supplier_portal.admin.audit_log.table.time')}</th>
                <th className="text-left px-4 py-3">{t('supplier_portal.admin.audit_log.table.actor')}</th>
                <th className="text-left px-4 py-3">{t('supplier_portal.admin.audit_log.table.action')}</th>
                <th className="text-left px-4 py-3">{t('supplier_portal.admin.audit_log.table.resource')}</th>
                <th className="text-left px-4 py-3">{t('supplier_portal.admin.audit_log.table.resource_id')}</th>
                <th className="text-left px-4 py-3">{t('supplier_portal.admin.audit_log.table.metadata')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.log_id} className="border-b last:border-b-0" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                  <td className="px-4 py-3 text-xs">{new Date(r.created_at).toLocaleString(locale)}</td>
                  <td className="px-4 py-3 text-xs">{r.actor_role} • {r.actor_id}</td>
                  <td className="px-4 py-3 text-xs">{r.action}</td>
                  <td className="px-4 py-3 text-xs">{r.resource_type}</td>
                  <td className="px-4 py-3 font-mono text-xs">{r.resource_id}</td>
                  <td className="px-4 py-3 text-xs max-w-[380px] truncate" title={r.metadata}>{r.metadata || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="flex items-center justify-between">
        <Button variant="outline" isDisabled={offset === 0} onPress={() => setOffset((o) => Math.max(0, o - limit))}>
          {t('supplier_portal.admin.audit_log.action.previous')}
        </Button>
        <span className="text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
          {t('supplier_portal.admin.audit_log.pagination.offset', { offset })}
        </span>
        <Button variant="outline" isDisabled={rows.length < limit} onPress={() => setOffset((o) => o + limit)}>
          {t('supplier_portal.admin.audit_log.action.next')}
        </Button>
      </div>
    </div>
  );
}
