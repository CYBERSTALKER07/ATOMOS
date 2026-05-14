'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { AlertCircle, ArrowLeft, CheckCircle2, FileSpreadsheet, UploadCloud } from 'lucide-react';
import { Button } from '@heroui/react';

import { BentoCard, BentoGrid, BentoSkeleton } from '@/components/BentoGrid';
import { useToast } from '@/components/Toast';
import { apiFetch, useToken } from '@/lib/auth';
import { useNotifications } from '@/lib/useNotifications';

import BentoMappingCard from './_components/BentoMappingCard';
import StagedPreviewGrid from './_components/StagedPreviewGrid';
import type {
  ImportSessionStatus,
  ImportWizardStep,
  MappingAnomaly,
  MappingCandidate,
  MappingDocument,
  MappingResponse,
  ResolvedMappingLink,
  RowsResponse,
  SupplierImportSession,
  SupplierImportStagedRow,
  UploadTicketResponse,
} from './_components/types';

const ACCEPTED_EXTENSIONS = ['.xlsx', '.xls'];
const MAX_UPLOAD_BYTES = 50 * 1024 * 1024;
const ROWS_PAGE_SIZE = 500;

const TARGET_FIELDS: string[] = [
  'supplier_id',
  'warehouse_id',
  'product_id',
  'sku_id',
  'product_name',
  'category_id',
  'quantity_available',
  'unit_price',
  'currency',
  'safety_stock_level',
  'min_stock_level',
  'max_stock_level',
  'h3_cell',
  'updated_at',
];

const STATUS_PROGRESS: Record<ImportSessionStatus, number> = {
  INITIALIZED: 5,
  UPLOADED: 25,
  DISCOVERING: 55,
  DISCOVERED: 80,
  MAPPING_REQUIRED: 85,
  APPROVED: 92,
  APPLYING: 97,
  APPLIED: 100,
  FAILED: 100,
};

const STEP_ORDER: ImportWizardStep[] = ['selection', 'mapping', 'review', 'finalize'];

function prettyState(value: string | undefined): string {
  if (!value) return 'UNKNOWN';
  return value.replaceAll('_', ' ');
}

function bytesToHuman(bytes: number): string {
  if (bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const step = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, step);
  return `${value.toFixed(step === 0 ? 0 : 1)} ${units[step]}`;
}

function parseJSONPayload(raw: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
    return {};
  } catch {
    return {};
  }
}

function parseStep(value: string | null): ImportWizardStep {
  if (!value) return 'selection';
  return STEP_ORDER.includes(value as ImportWizardStep) ? (value as ImportWizardStep) : 'selection';
}

function normalizeImportStatus(value: unknown): ImportSessionStatus | null {
  if (typeof value !== 'string') return null;
  const upper = value.trim().toUpperCase() as ImportSessionStatus;
  if (upper in STATUS_PROGRESS) {
    return upper;
  }
  return null;
}

function chooseBestMappingByHeader(candidates: MappingCandidate[]): Map<string, MappingCandidate> {
  const out = new Map<string, MappingCandidate>();
  for (const candidate of candidates) {
    const key = (candidate.source_column || '').trim();
    if (!key) continue;
    const prev = out.get(key);
    if (!prev || (candidate.confidence ?? 0) > (prev.confidence ?? 0)) {
      out.set(key, candidate);
    }
  }
  return out;
}

function collectHeaders(rows: SupplierImportStagedRow[], suggestions: Map<string, MappingCandidate>, manual: Record<string, string | null>): string[] {
  const set = new Set<string>();
  for (const key of suggestions.keys()) set.add(key);
  for (const key of Object.keys(manual)) set.add(key);

  for (const row of rows.slice(0, 5)) {
    const rawData = row.raw_data;
    if (!rawData || typeof rawData !== 'object' || Array.isArray(rawData)) {
      continue;
    }
    for (const key of Object.keys(rawData as Record<string, unknown>)) {
      set.add(key);
    }
  }

  return Array.from(set).sort((a, b) => a.localeCompare(b));
}

interface UploadDropzoneProps {
  disabled: boolean;
  onSelect: (file: File) => void;
}

function UploadDropzone({ disabled, onSelect }: UploadDropzoneProps) {
  const [dragOver, setDragOver] = useState(false);

  return (
    <label
      className="relative block md-shape-lg p-8 cursor-pointer"
      onDragEnter={(event) => {
        event.preventDefault();
        if (!disabled) setDragOver(true);
      }}
      onDragOver={(event) => event.preventDefault()}
      onDragLeave={(event) => {
        event.preventDefault();
        setDragOver(false);
      }}
      onDrop={(event) => {
        event.preventDefault();
        setDragOver(false);
        if (disabled) return;
        const file = event.dataTransfer.files?.[0];
        if (file) onSelect(file);
      }}
      style={{
        border: `1px dashed ${dragOver ? 'var(--color-md-primary)' : 'var(--border)'}`,
        background: dragOver ? 'var(--color-md-primary-container)' : 'var(--surface)',
      }}
    >
      <input
        type="file"
        className="hidden"
        accept={ACCEPTED_EXTENSIONS.join(',')}
        disabled={disabled}
        onChange={(event) => {
          const file = event.target.files?.[0];
          if (file) onSelect(file);
          event.currentTarget.value = '';
        }}
      />
      <div className="flex flex-col items-center text-center gap-2">
        <UploadCloud size={40} style={{ color: 'var(--color-md-primary)' }} aria-hidden="true" />
        <h2 className="md-typescale-title-medium">Drop Excel File</h2>
        <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
          Upload streams directly to GCS using a short-lived signed URL.
        </p>
      </div>
    </label>
  );
}

export default function InventoryImportCommandCenterPage() {
  const token = useToken();
  const { toast } = useToast();
  const notifications = useNotifications();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const sessionID = (searchParams.get('session_id') || '').trim();
  const step = parseStep(searchParams.get('step'));

  const [session, setSession] = useState<SupplierImportSession | null>(null);
  const [mappingDoc, setMappingDoc] = useState<MappingDocument | null>(null);
  const [rows, setRows] = useState<SupplierImportStagedRow[]>([]);
  const [rowsHasMore, setRowsHasMore] = useState(false);
  const [nextOffset, setNextOffset] = useState(0);

  const [uploading, setUploading] = useState(false);
  const [savingMapping, setSavingMapping] = useState(false);
  const [finalizing, setFinalizing] = useState(false);
  const [loadingRows, setLoadingRows] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const [lastTicket, setLastTicket] = useState<UploadTicketResponse | null>(null);
  const [lastFile, setLastFile] = useState<{ name: string; size: number } | null>(null);
  const [lastError, setLastError] = useState('');

  const [manualAssignments, setManualAssignments] = useState<Record<string, string | null>>({});
  const [ignoredHeaders, setIgnoredHeaders] = useState<Record<string, boolean>>({});

  const [workerProgress, setWorkerProgress] = useState<number | null>(null);
  const handledNotificationIDs = useRef<Set<string>>(new Set());

  const updateRouteState = useCallback((next: { sessionID?: string; step?: ImportWizardStep }) => {
    const params = new URLSearchParams(searchParams.toString());
    const nextSessionID = next.sessionID !== undefined ? next.sessionID : sessionID;
    const nextStep = next.step !== undefined ? next.step : step;

    if (nextSessionID) params.set('session_id', nextSessionID);
    else params.delete('session_id');

    params.set('step', nextStep);
    router.replace(`${pathname}?${params.toString()}`);
  }, [pathname, router, searchParams, sessionID, step]);

  const loadSession = useCallback(async (targetSessionID = sessionID) => {
    if (!targetSessionID) {
      setSession(null);
      return;
    }

    setRefreshing(true);
    try {
      const res = await apiFetch(`/v1/supplier/inventory/imports/${targetSessionID}`);
      const body = await res.json().catch(() => ({} as { error?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to load import session');
      }
      const payload = body as SupplierImportSession;
      setSession(payload);
      if (payload.status === 'FAILED' && !lastError) {
        setLastError('Discovery failed. Review error summary and retry mapping.');
      }
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setRefreshing(false);
    }
  }, [lastError, sessionID, toast]);

  const loadMapping = useCallback(async (targetSessionID = sessionID) => {
    if (!targetSessionID) {
      setMappingDoc(null);
      return;
    }

    try {
      const res = await apiFetch(`/v1/supplier/inventory/imports/${targetSessionID}/mapping`);
      const body = await res.json().catch(() => ({} as { error?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to load mapping');
      }
      const payload = body as MappingResponse;
      setMappingDoc((payload.mapping_json || null) as MappingDocument | null);
    } catch (error) {
      toast((error as Error).message, 'error');
    }
  }, [sessionID, toast]);

  const loadRows = useCallback(async (targetSessionID = sessionID, offset = 0, append = false) => {
    if (!targetSessionID) {
      setRows([]);
      setRowsHasMore(false);
      setNextOffset(0);
      return;
    }

    setLoadingRows(true);
    try {
      const query = new URLSearchParams({
        limit: String(ROWS_PAGE_SIZE),
        offset: String(offset),
      });
      const res = await apiFetch(`/v1/supplier/inventory/imports/${targetSessionID}/rows?${query.toString()}`);
      const body = await res.json().catch(() => ({} as { error?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to load staged rows');
      }
      const payload = body as RowsResponse;
      setRows((prev) => (append ? [...prev, ...(payload.data || [])] : payload.data || []));
      setRowsHasMore(Boolean(payload.has_more));
      setNextOffset(payload.next_offset || 0);
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setLoadingRows(false);
    }
  }, [sessionID, toast]);

  const refreshAll = useCallback(async (targetSessionID = sessionID) => {
    await loadSession(targetSessionID);
    await loadMapping(targetSessionID);
    if (step === 'review') {
      await loadRows(targetSessionID, 0, false);
    }
  }, [loadMapping, loadRows, loadSession, sessionID, step]);

  useEffect(() => {
    if (!sessionID) {
      setSession(null);
      setMappingDoc(null);
      setRows([]);
      setRowsHasMore(false);
      setNextOffset(0);
      setWorkerProgress(null);
      return;
    }

    void loadSession(sessionID);
    void loadMapping(sessionID);
    if (step === 'review') {
      void loadRows(sessionID, 0, false);
    }
  }, [loadMapping, loadRows, loadSession, sessionID, step]);

  useEffect(() => {
    if (!sessionID || !['UPLOADED', 'DISCOVERING', 'MAPPING_REQUIRED', 'DISCOVERED'].includes(session?.status || '')) {
      return;
    }

    const timer = setInterval(() => {
      void refreshAll(sessionID);
    }, 6000);

    return () => clearInterval(timer);
  }, [refreshAll, session?.status, sessionID]);

  useEffect(() => {
    if (!sessionID) return;

    for (const notification of notifications.items) {
      if (handledNotificationIDs.current.has(notification.id)) {
        continue;
      }
      handledNotificationIDs.current.add(notification.id);

      const normalizedType = (notification.type || '').toUpperCase();
      if (
        normalizedType !== 'IMPORT_STATUS' &&
        normalizedType !== 'INVENTORY_IMPORT_STATUS_UPDATE' &&
        normalizedType !== 'IMPORT_PROGRESS' &&
        normalizedType !== 'STATUS_UPDATE'
      ) {
        continue;
      }

      const payload = parseJSONPayload(notification.payload || '');
      const payloadSessionID = String(payload.session_id || payload.sessionId || '').trim();
      if (payloadSessionID && payloadSessionID !== sessionID) {
        continue;
      }

      if (normalizedType === 'IMPORT_PROGRESS') {
        const progressFromPayload = Number(payload.progress ?? payload.percent ?? 0);
        if (!Number.isNaN(progressFromPayload) && progressFromPayload >= 0 && progressFromPayload <= 100) {
          setWorkerProgress(progressFromPayload);
        }
      }

      const statusFromPayload = normalizeImportStatus(payload.status);
      if (statusFromPayload && statusFromPayload in STATUS_PROGRESS) {
        setWorkerProgress(STATUS_PROGRESS[statusFromPayload]);
      }

      void refreshAll(sessionID);
    }
  }, [notifications.items, refreshAll, sessionID]);

  const suggestedMappings = useMemo(() => mappingDoc?.mappings || [], [mappingDoc]);
  const anomalyList = useMemo(() => mappingDoc?.anomalies || [], [mappingDoc]);

  const bestSuggestions = useMemo(
    () => chooseBestMappingByHeader(suggestedMappings),
    [suggestedMappings],
  );

  const sourceHeaders = useMemo(
    () => collectHeaders(rows, bestSuggestions, manualAssignments),
    [rows, bestSuggestions, manualAssignments],
  );

  const resolvedLinks = useMemo<ResolvedMappingLink[]>(() => {
    return sourceHeaders.map((header) => {
      const suggestion = bestSuggestions.get(header);
      const manual = Object.prototype.hasOwnProperty.call(manualAssignments, header)
        ? manualAssignments[header]
        : undefined;
      const ignored = Boolean(ignoredHeaders[header]);

      const targetField = ignored
        ? null
        : (manual !== undefined ? manual : suggestion?.target_field || null);

      return {
        sourceColumn: header,
        targetField,
        confidence: manual !== undefined ? 1 : (suggestion?.confidence ?? 0),
        reason: manual !== undefined ? 'manual_override' : suggestion?.reason,
        deterministic: suggestion?.deterministic,
        manual: manual !== undefined,
        ignored,
      };
    });
  }, [bestSuggestions, ignoredHeaders, manualAssignments, sourceHeaders]);

  const derivedProgress = useMemo(() => {
    if (workerProgress !== null) return workerProgress;
    if (!session?.status) return 0;
    return STATUS_PROGRESS[session.status] ?? 0;
  }, [session?.status, workerProgress]);

  const canStepToMapping = Boolean(sessionID);
  const canStepToReview = canStepToMapping && ['MAPPING_REQUIRED', 'DISCOVERED', 'APPROVED', 'APPLYING', 'APPLIED', 'FAILED'].includes(session?.status || '');
  const canStepToFinalize = canStepToReview;

  const setStep = useCallback((nextStep: ImportWizardStep) => {
    if (nextStep === 'mapping' && !canStepToMapping) return;
    if (nextStep === 'review' && !canStepToReview) return;
    if (nextStep === 'finalize' && !canStepToFinalize) return;
    updateRouteState({ step: nextStep });
  }, [canStepToFinalize, canStepToMapping, canStepToReview, updateRouteState]);

  const handleFileSelected = useCallback(async (file: File) => {
    if (!token) return;

    const extension = `.${(file.name.split('.').pop() || '').toLowerCase()}`;
    if (!ACCEPTED_EXTENSIONS.includes(extension)) {
      toast('Only Excel files (.xlsx or .xls) are supported.', 'error');
      return;
    }
    if (file.size > MAX_UPLOAD_BYTES) {
      toast('File exceeds 50MB upload limit.', 'error');
      return;
    }

    setUploading(true);
    setLastError('');
    setLastFile({ name: file.name, size: file.size });

    try {
      const createRes = await apiFetch('/v1/supplier/inventory/imports/', {
        method: 'POST',
        body: JSON.stringify({
          file_name: file.name,
          file_size_bytes: file.size,
        }),
      });
      const createBody = await createRes.json().catch(() => ({} as { error?: string }));
      if (!createRes.ok) {
        throw new Error(createBody.error || 'Failed to create import session');
      }

      const ticket = createBody as UploadTicketResponse;
      setLastTicket(ticket);
      setWorkerProgress(10);

      const uploadRes = await fetch(ticket.upload_url, {
        method: 'PUT',
        headers: {
          'Content-Type': ticket.content_type,
        },
        body: file,
      });
      if (!uploadRes.ok) {
        throw new Error(`GCS upload failed with status ${uploadRes.status}`);
      }

      setWorkerProgress(35);
      const uploadedRes = await apiFetch(`/v1/supplier/inventory/imports/${ticket.session_id}/uploaded`, {
        method: 'POST',
        body: JSON.stringify({ gcs_path: ticket.gcs_path }),
      });
      const uploadedBody = await uploadedRes.json().catch(() => ({} as { error?: string }));
      if (!uploadedRes.ok) {
        throw new Error(uploadedBody.error || 'Failed to trigger discovery');
      }

      updateRouteState({ sessionID: ticket.session_id, step: 'mapping' });
      await refreshAll(ticket.session_id);
      toast('Upload complete. AI discovery is now running.', 'success');
    } catch (error) {
      const message = (error as Error).message;
      setLastError(message);
      toast(message, 'error');
    } finally {
      setUploading(false);
    }
  }, [refreshAll, toast, token, updateRouteState]);

  const handleAssign = useCallback((header: string, targetField: string | null, manual = false) => {
    setIgnoredHeaders((prev) => ({ ...prev, [header]: false }));
    if (!manual) return;
    setManualAssignments((prev) => ({ ...prev, [header]: targetField }));
  }, []);

  const handleToggleIgnore = useCallback((header: string) => {
    setIgnoredHeaders((prev) => ({ ...prev, [header]: !prev[header] }));
  }, []);

  const handleSaveMapping = useCallback(async () => {
    if (!sessionID) return;
    setSavingMapping(true);

    try {
      const payload = {
        mapping_json: {
          mappings: resolvedLinks
            .filter((link) => !link.ignored && link.targetField)
            .map((link) => ({
              source_column: link.sourceColumn,
              target_field: link.targetField,
              confidence: link.confidence,
              reason: link.reason || (link.manual ? 'manual_override' : undefined),
              deterministic: Boolean(link.deterministic),
            })),
          anomalies: anomalyList,
          ignored_columns: resolvedLinks.filter((link) => link.ignored).map((link) => link.sourceColumn),
          generated_by: 'portal_manual_review',
          generated_at: new Date().toISOString(),
        },
      };

      const res = await apiFetch(`/v1/supplier/inventory/imports/${sessionID}/mapping`, {
        method: 'POST',
        body: JSON.stringify(payload),
      });
      const body = await res.json().catch(() => ({} as { error?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to save mapping');
      }

      await refreshAll(sessionID);
      toast('Mapping saved. Continue to staged review.', 'success');
      setStep('review');
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setSavingMapping(false);
    }
  }, [anomalyList, refreshAll, resolvedLinks, sessionID, setStep, toast]);

  const handleFinalize = useCallback(async () => {
    if (!sessionID) return;
    setFinalizing(true);
    try {
      const res = await apiFetch(`/v1/supplier/inventory/imports/${sessionID}/apply`, {
        method: 'POST',
      });
      const body = await res.json().catch(() => ({} as { error?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to apply import');
      }
      await refreshAll(sessionID);
      toast('Import applied to production inventory.', 'success');
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setFinalizing(false);
    }
  }, [refreshAll, sessionID, toast]);

  const handleLoadMoreRows = useCallback(() => {
    if (!sessionID || !rowsHasMore || loadingRows) return;
    void loadRows(sessionID, nextOffset, true);
  }, [loadRows, loadingRows, nextOffset, rowsHasMore, sessionID]);

  if (!token) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="md-card md-card-elevated p-6" style={{ background: 'var(--color-md-error-container)', color: 'var(--foreground)' }}>
          Unauthorized — supplier credentials required
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <header className="mb-6 pb-4" style={{ borderBottom: '1px solid var(--border)' }}>
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <div>
            <h1 className="md-typescale-headline-medium">Inventory Import Command Center</h1>
            <p className="md-typescale-body-small mt-2" style={{ color: 'var(--muted)' }}>
              Human-in-the-loop wizard for AI mapping review, staged validation, and finalization.
            </p>
          </div>
          <Link href="/supplier/inventory">
            <Button variant="outline">
              <ArrowLeft size={14} aria-hidden="true" /> Back To Inventory
            </Button>
          </Link>
        </div>
      </header>

      <div className="mb-4">
        <div className="md-shape-full overflow-hidden" style={{ height: 6, background: 'var(--surface)' }}>
          <div
            style={{
              width: `${Math.min(100, Math.max(0, derivedProgress))}%`,
              height: '100%',
              background: session?.status === 'FAILED' ? 'var(--color-md-warning)' : 'var(--color-md-primary)',
              transition: 'width 300ms ease',
              animation: 'md-progress-indeterminate 1.8s ease-in-out infinite',
            }}
          />
        </div>
        <div className="flex items-center justify-between mt-2">
          <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
            {session?.status ? `Status: ${prettyState(session.status)}` : 'No active session'}
          </span>
          <span className="md-typescale-body-small font-mono" style={{ color: 'var(--muted)' }}>
            {Math.round(derivedProgress)}%
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-2 mb-6">
        {STEP_ORDER.map((wizardStep, index) => {
          const enabled = wizardStep === 'selection'
            || (wizardStep === 'mapping' && canStepToMapping)
            || (wizardStep === 'review' && canStepToReview)
            || (wizardStep === 'finalize' && canStepToFinalize);
          const active = wizardStep === step;

          return (
            <button
              key={wizardStep}
              onClick={() => setStep(wizardStep)}
              disabled={!enabled}
              className="md-chip w-full justify-center py-2"
              style={active ? {
                borderColor: 'var(--color-md-primary)',
                color: 'var(--color-md-primary)',
                background: 'var(--color-md-primary-container)',
              } : undefined}
            >
              {index + 1}. {prettyState(wizardStep).toLowerCase()}
            </button>
          );
        })}
      </div>

      <BentoGrid>
        {step === 'selection' ? (
          <>
            <BentoCard size="control" className="p-5">
              <h2 className="md-typescale-title-medium mb-3">Selection</h2>
              <UploadDropzone disabled={uploading} onSelect={(file) => void handleFileSelected(file)} />
              <div className="mt-4 flex flex-wrap gap-2">
                <span className="md-chip md-chip-selected" style={{ cursor: 'default' }}>
                  {uploading ? 'Uploading...' : 'Ready'}
                </span>
                <span className="md-chip" style={{ cursor: 'default' }}>
                  Max {bytesToHuman(MAX_UPLOAD_BYTES)}
                </span>
              </div>
            </BentoCard>

            <BentoCard size="list" className="p-5">
              <h2 className="md-typescale-title-medium mb-3">Upload Contract</h2>
              <div className="space-y-3">
                <div className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                  <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Allowed Formats</div>
                  <div className="md-typescale-body-small">{ACCEPTED_EXTENSIONS.join(', ')}</div>
                </div>
                {lastFile ? (
                  <div className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                    <div className="flex items-center gap-2">
                      <FileSpreadsheet size={14} aria-hidden="true" style={{ color: 'var(--color-md-primary)' }} />
                      <span className="md-typescale-label-medium">{lastFile.name}</span>
                    </div>
                    <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>{bytesToHuman(lastFile.size)}</p>
                  </div>
                ) : null}
                {lastTicket ? (
                  <div className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                    <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Session ID</div>
                    <div className="md-typescale-body-small font-mono break-all">{lastTicket.session_id}</div>
                  </div>
                ) : null}
              </div>
            </BentoCard>

            <BentoCard size="anchor" className="p-5">
              <h2 className="md-typescale-title-medium mb-2">Discovery Watch</h2>
              <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                Worker events are consumed via useNotifications. IMPORT_STATUS and IMPORT_PROGRESS frames update this page live.
              </p>
              <div className="mt-4 p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Next Step</p>
                <p className="md-typescale-body-small mt-1">Once uploaded, continue to Mapping to review AI links.</p>
              </div>
            </BentoCard>
          </>
        ) : null}

        {step === 'mapping' ? (
          <>
            {!sessionID ? <BentoSkeleton size="anchor" /> : (
              <BentoMappingCard
                headers={sourceHeaders}
                targetFields={TARGET_FIELDS}
                links={resolvedLinks}
                onAssign={handleAssign}
                onToggleIgnore={handleToggleIgnore}
              />
            )}

            <BentoCard size="list" className="p-5">
              <h2 className="md-typescale-title-medium mb-3">Mapping Actions</h2>
              <div className="space-y-3">
                <Button variant="primary" onPress={() => void handleSaveMapping()} isDisabled={!sessionID || savingMapping}>
                  {savingMapping ? 'Saving Mapping...' : 'Save Mapping And Continue'}
                </Button>
                <Button variant="outline" onPress={() => setStep('review')} isDisabled={!canStepToReview}>
                  Go To Review
                </Button>
                <Button variant="outline" onPress={() => void refreshAll()} isDisabled={!sessionID || refreshing}>
                  {refreshing ? 'Refreshing...' : 'Refresh Session'}
                </Button>
              </div>
              <div className="mt-4 p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Active Session</div>
                <div className="md-typescale-body-small font-mono break-all">{sessionID || 'none'}</div>
              </div>
            </BentoCard>

            <BentoCard size="wide" className="p-5">
              <h2 className="md-typescale-title-medium mb-3">AI Findings</h2>
              {anomalyList.length === 0 ? (
                <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>No anomalies reported yet.</p>
              ) : (
                <ul className="space-y-2">
                  {anomalyList.map((anomaly, idx) => (
                    <li key={`${anomaly.kind}-${idx}`} className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                      <p className="md-typescale-label-medium">{anomaly.kind}</p>
                      <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>{anomaly.detail}</p>
                    </li>
                  ))}
                </ul>
              )}
            </BentoCard>
          </>
        ) : null}

        {step === 'review' ? (
          <>
            <StagedPreviewGrid
              rows={rows}
              links={resolvedLinks}
              anomalies={anomalyList as MappingAnomaly[]}
              loading={loadingRows}
              hasMore={rowsHasMore}
              onLoadMore={handleLoadMoreRows}
            />

            <BentoCard size="list" className="p-5">
              <h2 className="md-typescale-title-medium mb-3">Review Controls</h2>
              <div className="space-y-3">
                <Button variant="outline" onPress={() => setStep('mapping')}>Back To Mapping</Button>
                <Button variant="primary" onPress={() => setStep('finalize')} isDisabled={!canStepToFinalize}>
                  Continue To Finalize
                </Button>
                <Button variant="outline" onPress={() => void loadRows(sessionID, 0, false)} isDisabled={!sessionID || loadingRows}>
                  {loadingRows ? 'Loading Preview...' : 'Reload Preview'}
                </Button>
              </div>

              <div className="mt-4 p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Rows Loaded</div>
                <div className="md-typescale-body-small">{rows.length}</div>
              </div>
            </BentoCard>
          </>
        ) : null}

        {step === 'finalize' ? (
          <>
            <BentoCard size="control" className="p-5">
              <h2 className="md-typescale-title-medium mb-3">Finalize</h2>
              <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                Approve this staged import to hand off to the atomic apply pipeline.
              </p>
              <div className="mt-4 flex flex-wrap gap-2">
                <Button
                  variant="primary"
                  onPress={() => void handleFinalize()}
                  isDisabled={!sessionID || finalizing || session?.status === 'APPLIED' || session?.status === 'APPLYING'}
                >
                  {finalizing ? 'Committing...' : 'Commit To Spanner'}
                </Button>
                <Button variant="outline" onPress={() => setStep('review')}>Back To Review</Button>
              </div>
            </BentoCard>

            <BentoCard size="anchor" className="p-5">
              <h2 className="md-typescale-title-medium mb-3">Session Status</h2>
              <div className="space-y-2">
                <div className="md-chip md-chip-selected" style={{ cursor: 'default' }}>
                  {prettyState(session?.status)}
                </div>
                {session?.status === 'APPLIED' ? (
                  <div className="p-3 md-shape-md flex items-start gap-2" style={{ border: '1px solid var(--color-md-primary)', background: 'var(--color-md-primary-container)' }}>
                    <CheckCircle2 size={16} aria-hidden="true" style={{ color: 'var(--color-md-primary)' }} />
                    <p className="md-typescale-body-small">Import has been applied successfully.</p>
                  </div>
                ) : null}
                {session?.status === 'FAILED' ? (
                  <div className="p-3 md-shape-md flex items-start gap-2" style={{ border: '1px solid var(--color-md-warning)', background: 'var(--color-md-error-container)' }}>
                    <AlertCircle size={16} aria-hidden="true" style={{ color: 'var(--color-md-warning)' }} />
                    <p className="md-typescale-body-small">Import failed. Resolve errors before retrying.</p>
                  </div>
                ) : null}
              </div>
            </BentoCard>
          </>
        ) : null}
      </BentoGrid>

      {session?.status === 'FAILED' ? (
        <div className="mt-6 p-4 md-shape-lg" style={{ border: '1px solid var(--color-md-warning)', background: 'var(--color-md-error-container)' }}>
          <h3 className="md-typescale-title-small">Failed Discovery Summary</h3>
          <pre className="mt-2 overflow-auto text-xs" style={{ color: 'var(--foreground)' }}>
            {JSON.stringify(session.error_summary || { error: lastError || 'Unknown discovery failure' }, null, 2)}
          </pre>
        </div>
      ) : null}
    </div>
  );
}
