'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button } from '@heroui/react';

import { BentoCard, BentoGrid, BentoSkeleton } from '@/components/BentoGrid';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import { apiFetch, useToken } from '@/lib/auth';

type ImportSessionState =
  | 'UPLOADED'
  | 'DISCOVERING'
  | 'MAPPING_REQUIRED'
  | 'READY_FOR_REVIEW'
  | 'APPROVED'
  | 'APPLYING'
  | 'APPLIED'
  | 'FAILED'
  | 'EXPIRED';

type ImportRowStatus =
  | 'UNMAPPED'
  | 'MAPPED_EXISTING'
  | 'PENDING_CREATION'
  | 'INVALID'
  | 'READY_FOR_REVIEW'
  | 'APPROVED'
  | 'APPLIED'
  | 'FAILED';

interface ImportSession {
  session_id: string;
  supplier_id?: string;
  warehouse_id?: string;
  file_name?: string;
  content_type?: string;
  object_path?: string;
  state: ImportSessionState;
  total_rows: number;
  processed_rows: number;
  failed_rows: number;
  pending_creation_rows: number;
  created_at?: string;
  updated_at?: string;
}

interface StagedImportRow {
  row_id: string;
  session_id: string;
  row_number: number;
  status: ImportRowStatus;
  source?: Record<string, string>;
  sku_id?: string;
  product_name?: string;
  category_id?: string;
  warehouse_id?: string;
  currency?: string;
  base_price?: number;
  quantity_delta?: number;
  minimum_order_qty?: number;
  step_size?: number;
  volumetric_unit?: number;
  length_cm?: number;
  width_cm?: number;
  height_cm?: number;
  errors?: string[];
  validation_status?: string;
}

interface UploadTicketResponse {
  upload_url: string;
  object_path: string;
  content_type: string;
  expires_in_seconds: number;
}

interface ListResponse<T> {
  data: T[];
  has_more?: boolean;
  limit?: number;
  offset?: number;
  next_offset?: number;
}

const SUPPORTED_EXTENSIONS = ['csv', 'tsv', 'xlsx', 'xls', 'json'] as const;

function prettyState(value: string | undefined): string {
  if (!value) return 'UNKNOWN';
  return value.replaceAll('_', ' ');
}

function toIso(value?: string): string {
  if (!value) return '\u2014';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

export default function InventoryImportWizardPage() {
  const { toast } = useToast();
  const token = useToken();

  const [sessions, setSessions] = useState<ImportSession[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(true);
  const [selectedSessionId, setSelectedSessionId] = useState<string>('');
  const [selectedSession, setSelectedSession] = useState<ImportSession | null>(null);
  const [rows, setRows] = useState<StagedImportRow[]>([]);
  const [rowsLoading, setRowsLoading] = useState(false);

  const [fileName, setFileName] = useState('inventory-import.csv');
  const [warehouseId, setWarehouseId] = useState('');
  const [extension, setExtension] = useState<(typeof SUPPORTED_EXTENSIONS)[number]>('csv');
  const [uploadTicket, setUploadTicket] = useState<UploadTicketResponse | null>(null);
  const [creatingSession, setCreatingSession] = useState(false);
  const [mappingDraft, setMappingDraft] = useState('[]');
  const [actionBusy, setActionBusy] = useState<'approve' | 'apply' | 'mapping' | 'status' | ''>('');

  const selectedSessionState = selectedSession?.state;

  const canApprove = useMemo(() => {
    if (!selectedSessionState) return false;
    return selectedSessionState === 'READY_FOR_REVIEW' || selectedSessionState === 'MAPPING_REQUIRED';
  }, [selectedSessionState]);

  const canApply = useMemo(() => selectedSessionState === 'APPROVED', [selectedSessionState]);

  const fetchSessions = useCallback(async () => {
    setSessionsLoading(true);
    try {
      const params = new URLSearchParams({ limit: '25', offset: '0' });
      const res = await apiFetch(`/v1/supplier/inventory/import?${params}`);
      if (!res.ok) {
        const body = await res.json().catch(() => ({} as { error?: string }));
        throw new Error(body.error || 'Failed to load import sessions');
      }
      const body = (await res.json()) as ListResponse<ImportSession>;
      const data = body.data || [];
      setSessions(data);
      if (!selectedSessionId && data.length > 0) {
        setSelectedSessionId(data[0].session_id);
      }
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setSessionsLoading(false);
    }
  }, [selectedSessionId, toast]);

  const fetchSessionDetail = useCallback(async (sessionId: string) => {
    if (!sessionId) return;
    try {
      const res = await apiFetch(`/v1/supplier/inventory/import/${sessionId}`);
      if (!res.ok) {
        const body = await res.json().catch(() => ({} as { error?: string }));
        throw new Error(body.error || 'Failed to load session detail');
      }
      const body = (await res.json()) as ImportSession;
      setSelectedSession(body);
    } catch (error) {
      toast((error as Error).message, 'error');
    }
  }, [toast]);

  const fetchRows = useCallback(async (sessionId: string) => {
    if (!sessionId) return;
    setRowsLoading(true);
    try {
      const params = new URLSearchParams({ limit: '50', offset: '0' });
      const res = await apiFetch(`/v1/supplier/inventory/import/${sessionId}/rows?${params}`);
      if (!res.ok) {
        const body = await res.json().catch(() => ({} as { error?: string }));
        throw new Error(body.error || 'Failed to load staged rows');
      }
      const body = (await res.json()) as ListResponse<StagedImportRow>;
      const data = body.data || [];
      setRows(data);
      if (data.length > 0) {
        setMappingDraft(JSON.stringify(data, null, 2));
      }
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setRowsLoading(false);
    }
  }, [toast]);

  const refreshStatus = useCallback(async () => {
    if (!selectedSessionId) return;
    setActionBusy('status');
    try {
      const res = await apiFetch(`/v1/supplier/inventory/import/${selectedSessionId}/status`);
      if (!res.ok) {
        const body = await res.json().catch(() => ({} as { error?: string }));
        throw new Error(body.error || 'Failed to refresh status');
      }
      await fetchSessionDetail(selectedSessionId);
      toast('Session status refreshed', 'success');
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setActionBusy('');
    }
  }, [fetchSessionDetail, selectedSessionId, toast]);

  useEffect(() => {
    if (!token) return;
    fetchSessions();
  }, [token, fetchSessions]);

  useEffect(() => {
    if (!token || !selectedSessionId) {
      setSelectedSession(null);
      setRows([]);
      return;
    }
    fetchSessionDetail(selectedSessionId);
    fetchRows(selectedSessionId);
  }, [token, selectedSessionId, fetchRows, fetchSessionDetail]);

  async function handleGenerateUploadTicket() {
    try {
      const res = await apiFetch(`/v1/supplier/inventory/import/upload-ticket?ext=${extension}`);
      if (!res.ok) {
        const body = await res.json().catch(() => ({} as { error?: string }));
        throw new Error(body.error || 'Failed to generate upload ticket');
      }
      const body = (await res.json()) as UploadTicketResponse;
      setUploadTicket(body);
      toast('Upload ticket generated', 'success');
    } catch (error) {
      toast((error as Error).message, 'error');
    }
  }

  async function handleCreateSession() {
    if (!uploadTicket?.object_path) {
      toast('Generate an upload ticket before creating a session', 'error');
      return;
    }

    setCreatingSession(true);
    try {
      const payload: {
        file_name: string;
        content_type: string;
        object_path: string;
        warehouse_id?: string;
      } = {
        file_name: fileName.trim(),
        content_type: uploadTicket.content_type,
        object_path: uploadTicket.object_path,
      };
      if (warehouseId.trim()) {
        payload.warehouse_id = warehouseId.trim();
      }

      const res = await apiFetch('/v1/supplier/inventory/import', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
      const body = await res.json().catch(() => ({} as { error?: string; session_id?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to create import session');
      }

      const createdSession = body as ImportSession;
      if (createdSession.session_id) {
        setSelectedSessionId(createdSession.session_id);
      }
      setUploadTicket(null);
      setWarehouseId('');
      await fetchSessions();
      toast('Import session created', 'success');
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setCreatingSession(false);
    }
  }

  async function handlePatchMapping() {
    if (!selectedSessionId) {
      toast('Select a session first', 'error');
      return;
    }

    let rowsPayload: StagedImportRow[];
    try {
      const parsed = JSON.parse(mappingDraft) as unknown;
      if (!Array.isArray(parsed)) {
        throw new Error('Mapping payload must be a JSON array of rows');
      }
      rowsPayload = parsed as StagedImportRow[];
      if (rowsPayload.length === 0) {
        throw new Error('Mapping payload cannot be empty');
      }
    } catch (error) {
      toast((error as Error).message, 'error');
      return;
    }

    setActionBusy('mapping');
    try {
      const res = await apiFetch(`/v1/supplier/inventory/import/${selectedSessionId}/mapping`, {
        method: 'PATCH',
        body: JSON.stringify({ rows: rowsPayload }),
      });
      const body = await res.json().catch(() => ({} as { error?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to update mapping');
      }
      await fetchSessionDetail(selectedSessionId);
      await fetchRows(selectedSessionId);
      await fetchSessions();
      toast('Mapping persisted', 'success');
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setActionBusy('');
    }
  }

  async function handleApprove() {
    if (!selectedSessionId) return;
    setActionBusy('approve');
    try {
      const res = await apiFetch(`/v1/supplier/inventory/import/${selectedSessionId}/approve`, {
        method: 'POST',
      });
      const body = await res.json().catch(() => ({} as { error?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to approve session');
      }
      await fetchSessionDetail(selectedSessionId);
      await fetchRows(selectedSessionId);
      await fetchSessions();
      toast('Session approved', 'success');
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setActionBusy('');
    }
  }

  async function handleApply() {
    if (!selectedSessionId) return;
    setActionBusy('apply');
    try {
      const res = await apiFetch(`/v1/supplier/inventory/import/${selectedSessionId}/apply`, {
        method: 'POST',
      });
      const body = await res.json().catch(() => ({} as { error?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to apply session');
      }
      await fetchSessionDetail(selectedSessionId);
      await fetchRows(selectedSessionId);
      await fetchSessions();
      toast('Import apply finished', 'success');
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setActionBusy('');
    }
  }

  if (!token) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="md-card md-card-elevated p-6" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}>
          Unauthorized \u2014 supplier credentials required
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <header className="mb-6 pb-4" style={{ borderBottom: '1px solid var(--border)' }}>
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <div>
            <h1 className="md-typescale-headline-medium">Inventory Import Wizard</h1>
            <p className="md-typescale-body-small mt-2" style={{ color: 'var(--muted)' }}>
              Stage file imports, map rows, review, approve, and apply inventory updates.
            </p>
          </div>
          <Link href="/supplier/inventory">
            <Button variant="outline">Back To Inventory</Button>
          </Link>
        </div>
      </header>

      {sessionsLoading ? (
        <BentoGrid>
          <BentoSkeleton size="control" />
          <BentoSkeleton size="list" />
          <BentoSkeleton size="anchor" />
          <BentoSkeleton size="wide" />
          <BentoSkeleton size="wide" />
        </BentoGrid>
      ) : (
        <BentoGrid>
          <BentoCard size="control" className="p-5">
            <div className="h-full flex flex-col gap-3">
              <h2 className="md-typescale-title-medium">1. Upload Ticket + Session</h2>
              <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
                <label className="flex flex-col gap-1">
                  <span className="md-typescale-label-small">File Name</span>
                  <input className="md-input-outlined" value={fileName} onChange={(e) => setFileName(e.target.value)} />
                </label>
                <label className="flex flex-col gap-1">
                  <span className="md-typescale-label-small">Extension</span>
                  <select
                    className="md-input-outlined"
                    value={extension}
                    onChange={(e) => setExtension(e.target.value as (typeof SUPPORTED_EXTENSIONS)[number])}
                  >
                    {SUPPORTED_EXTENSIONS.map((item) => (
                      <option key={item} value={item}>
                        {item.toUpperCase()}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="flex flex-col gap-1">
                  <span className="md-typescale-label-small">Warehouse ID (optional)</span>
                  <input className="md-input-outlined" value={warehouseId} onChange={(e) => setWarehouseId(e.target.value)} />
                </label>
                <div className="flex items-end gap-2">
                  <Button variant="outline" onPress={handleGenerateUploadTicket}>Generate Ticket</Button>
                  <Button variant="primary" isLoading={creatingSession} onPress={handleCreateSession}>Create Session</Button>
                </div>
              </div>
              <div className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                Upload URL and object path are generated from backend ticketing. File transfer itself can be handled by external tooling.
              </div>
              {uploadTicket && (
                <div className="p-3 md-shape-md" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
                  <div className="md-typescale-label-small">Ticket Ready</div>
                  <div className="md-typescale-body-small font-mono break-all mt-1">{uploadTicket.object_path}</div>
                  <div className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                    {uploadTicket.content_type} \u2022 expires in {uploadTicket.expires_in_seconds}s
                  </div>
                </div>
              )}
            </div>
          </BentoCard>

          <BentoCard size="list" className="p-0 overflow-hidden">
            <div className="px-4 py-3" style={{ borderBottom: '1px solid var(--border)' }}>
              <h2 className="md-typescale-title-medium">2. Sessions</h2>
            </div>
            {sessions.length === 0 ? (
              <EmptyState icon="inventory" headline="No import sessions" body="Create a new session to begin staged import." />
            ) : (
              <div className="overflow-auto h-full">
                <table className="md-table">
                  <thead>
                    <tr>
                      <th>Session</th>
                      <th>State</th>
                      <th className="text-right">Rows</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sessions.map((session) => (
                      <tr
                        key={session.session_id}
                        className="cursor-pointer"
                        onClick={() => setSelectedSessionId(session.session_id)}
                        style={{
                          background: selectedSessionId === session.session_id ? 'var(--surface)' : 'transparent',
                        }}
                      >
                        <td>
                          <div className="font-mono md-typescale-label-small">{session.session_id.slice(0, 8)}</div>
                          <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                            {toIso(session.updated_at)}
                          </div>
                        </td>
                        <td>{prettyState(session.state)}</td>
                        <td className="text-right font-mono">{session.total_rows}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </BentoCard>

          <BentoCard size="anchor" className="p-5">
            {!selectedSession ? (
              <EmptyState icon="orders" headline="No selected session" body="Choose a session to inspect status and actions." />
            ) : (
              <div className="h-full flex flex-col gap-4">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h2 className="md-typescale-title-medium">3. Review + Actions</h2>
                    <div className="md-typescale-body-small font-mono">{selectedSession.session_id}</div>
                  </div>
                  <div className="md-chip md-chip-selected" style={{ cursor: 'default' }}>
                    {prettyState(selectedSession.state)}
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="p-3 md-shape-md" style={{ background: 'var(--surface)' }}>
                    <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Total Rows</div>
                    <div className="md-kpi-value">{selectedSession.total_rows}</div>
                  </div>
                  <div className="p-3 md-shape-md" style={{ background: 'var(--surface)' }}>
                    <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Processed Rows</div>
                    <div className="md-kpi-value">{selectedSession.processed_rows}</div>
                  </div>
                  <div className="p-3 md-shape-md" style={{ background: 'var(--surface)' }}>
                    <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Failed Rows</div>
                    <div className="md-kpi-value">{selectedSession.failed_rows}</div>
                  </div>
                  <div className="p-3 md-shape-md" style={{ background: 'var(--surface)' }}>
                    <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Pending Creation</div>
                    <div className="md-kpi-value">{selectedSession.pending_creation_rows}</div>
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" isLoading={actionBusy === 'status'} onPress={refreshStatus}>Refresh Status</Button>
                  <Button variant="outline" isLoading={rowsLoading} onPress={() => fetchRows(selectedSession.session_id)}>Reload Rows</Button>
                  <Button variant="primary" isDisabled={!canApprove} isLoading={actionBusy === 'approve'} onPress={handleApprove}>
                    Approve Session
                  </Button>
                  <Button variant="primary" isDisabled={!canApply} isLoading={actionBusy === 'apply'} onPress={handleApply}>
                    Apply Session
                  </Button>
                </div>
              </div>
            )}
          </BentoCard>

          <BentoCard size="wide" className="p-0 overflow-hidden">
            <div className="px-4 py-3" style={{ borderBottom: '1px solid var(--border)' }}>
              <h2 className="md-typescale-title-medium">4. Staged Rows Preview</h2>
            </div>
            {rowsLoading ? (
              <div className="p-6 md-typescale-body-small" style={{ color: 'var(--muted)' }}>Loading staged rows...</div>
            ) : rows.length === 0 ? (
              <EmptyState icon="inventory" headline="No staged rows yet" body="Persist mapping rows, then reload this preview." />
            ) : (
              <div className="overflow-auto">
                <table className="md-table">
                  <thead>
                    <tr>
                      <th>#</th>
                      <th>Status</th>
                      <th>SKU</th>
                      <th>Product</th>
                      <th className="text-right">Delta</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row) => (
                      <tr key={row.row_id}>
                        <td className="font-mono">{row.row_number}</td>
                        <td>{prettyState(row.status)}</td>
                        <td className="font-mono md-typescale-label-small">{row.sku_id || '\u2014'}</td>
                        <td>{row.product_name || '\u2014'}</td>
                        <td className="text-right font-mono">{row.quantity_delta || 0}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </BentoCard>

          <BentoCard size="wide" className="p-5">
            <div className="h-full flex flex-col gap-3">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="md-typescale-title-medium">5. Mapping Patch Editor</h2>
                  <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                    Edit and submit row mapping payload as JSON array. This shell enables backend integration testing and staged operator workflows.
                  </p>
                </div>
                <Button variant="outline" onPress={() => setMappingDraft(JSON.stringify(rows, null, 2))}>Load Preview Rows</Button>
              </div>
              <textarea
                value={mappingDraft}
                onChange={(e) => setMappingDraft(e.target.value)}
                className="md-input-outlined font-mono"
                style={{ minHeight: 240, resize: 'vertical' }}
              />
              <div className="flex justify-end">
                <Button variant="primary" isLoading={actionBusy === 'mapping'} onPress={handlePatchMapping}>
                  Persist Mapping
                </Button>
              </div>
            </div>
          </BentoCard>
        </BentoGrid>
      )}
    </div>
  );
}
