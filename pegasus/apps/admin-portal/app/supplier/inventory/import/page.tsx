'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { FileSpreadsheet, UploadCloud } from 'lucide-react';
import { Button } from '@heroui/react';

import { BentoCard, BentoGrid, BentoSkeleton } from '@/components/BentoGrid';
import { useToast } from '@/components/Toast';
import { apiFetch, useToken } from '@/lib/auth';
import { useNotifications } from '@/lib/useNotifications';

type ImportSessionStatus =
  | 'INITIALIZED'
  | 'UPLOADED'
  | 'DISCOVERING'
  | 'MAPPING_REQUIRED'
  | 'APPROVED'
  | 'APPLYING'
  | 'APPLIED'
  | 'FAILED';

type UploadPhase = 'idle' | 'ticketing' | 'uploading' | 'signaling' | 'processing' | 'ready' | 'error';

interface SupplierImportSession {
  session_id: string;
  supplier_id: string;
  status: ImportSessionStatus;
  file_name: string;
  gcs_path: string;
  total_rows: number;
  created_at: string;
  updated_at?: string;
}

interface UploadTicketResponse {
  session_id: string;
  supplier_id: string;
  status: ImportSessionStatus;
  file_name: string;
  upload_url: string;
  gcs_path: string;
  content_type: string;
  expires_in_seconds: number;
  max_file_size_bytes: number;
}

const MAX_UPLOAD_BYTES = 50 * 1024 * 1024;
const ACCEPTED_EXTENSIONS = ['.xlsx', '.xls'];

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

function parseSessionIDFromPayload(payload: string): string {
  const trimmed = payload.trim();
  if (!trimmed) return '';
  try {
    const parsed = JSON.parse(trimmed) as { session_id?: string; sessionId?: string };
    return (parsed.session_id || parsed.sessionId || '').trim();
  } catch {
    return '';
  }
}

interface DropzoneProps {
  disabled: boolean;
  phase: UploadPhase;
  onFileSelected: (file: File) => void;
}

function UploadDropzone({ disabled, phase, onFileSelected }: DropzoneProps) {
  const [isDragOver, setIsDragOver] = useState(false);
  const inputRef = useRef<HTMLInputElement | null>(null);

  const handleDrop = (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (disabled) return;
    setIsDragOver(false);
    const file = event.dataTransfer.files?.[0];
    if (file) {
      onFileSelected(file);
    }
  };

  const handleBrowse = () => {
    if (disabled) return;
    inputRef.current?.click();
  };

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={handleBrowse}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          handleBrowse();
        }
      }}
      onDragEnter={(event) => {
        event.preventDefault();
        if (!disabled) setIsDragOver(true);
      }}
      onDragOver={(event) => {
        event.preventDefault();
      }}
      onDragLeave={(event) => {
        event.preventDefault();
        setIsDragOver(false);
      }}
      onDrop={handleDrop}
      className="relative overflow-hidden md-shape-lg p-6 md:p-8 cursor-pointer"
      style={{
        border: `1px dashed ${isDragOver ? 'var(--accent)' : 'var(--border)'}`,
        background: isDragOver ? 'var(--accent-soft)' : 'var(--surface)',
        minHeight: 210,
      }}
      aria-label="Excel upload dropzone"
    >
      <input
        ref={inputRef}
        type="file"
        accept={ACCEPTED_EXTENSIONS.join(',')}
        className="hidden"
        onChange={(event) => {
          const file = event.target.files?.[0];
          if (file) {
            onFileSelected(file);
          }
          event.currentTarget.value = '';
        }}
      />

      <AnimatePresence>
        {isDragOver ? (
          <>
            <motion.div
              key="vortex-ring-outer"
              className="absolute inset-6 rounded-full pointer-events-none"
              style={{ border: '1px solid var(--accent)' }}
              initial={{ opacity: 0.12, scale: 0.9, rotate: 0 }}
              animate={{ opacity: [0.12, 0.45, 0.12], scale: [0.9, 1.05, 0.9], rotate: 360 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 2.2, repeat: Number.POSITIVE_INFINITY, ease: 'linear' }}
            />
            <motion.div
              key="vortex-ring-inner"
              className="absolute inset-14 rounded-full pointer-events-none"
              style={{ border: '1px dashed var(--accent)' }}
              initial={{ opacity: 0.1, scale: 0.85, rotate: 360 }}
              animate={{ opacity: [0.1, 0.36, 0.1], scale: [0.85, 1.02, 0.85], rotate: 0 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 1.8, repeat: Number.POSITIVE_INFINITY, ease: 'linear' }}
            />
          </>
        ) : null}
      </AnimatePresence>

      <div className="relative z-10 h-full flex flex-col items-center justify-center text-center gap-3">
        <UploadCloud size={42} style={{ color: 'var(--accent)' }} aria-hidden="true" />
        <div className="md-typescale-title-medium">Drop Excel File To Start Ingress</div>
        <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
          Direct upload uses a short-lived signed URL; backend remains stateless.
        </p>
        <div className="md-chip md-chip-selected" style={{ cursor: 'default' }}>
          {phase === 'idle' ? 'Awaiting file' : prettyState(phase)}
        </div>
      </div>
    </div>
  );
}

function ProcessingScanner() {
  return (
    <div className="h-full w-full flex flex-col gap-4">
      <h3 className="md-typescale-title-medium">Discovery In Progress</h3>
      <div className="md-shape-lg p-4" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
        <svg viewBox="0 0 420 220" className="w-full" style={{ height: 220 }} aria-hidden="true">
          <rect x="48" y="18" width="324" height="184" rx="10" fill="none" stroke="var(--border)" strokeWidth="2" />
          <rect x="78" y="52" width="264" height="12" rx="4" fill="var(--accent-soft)" />
          <rect x="78" y="82" width="190" height="10" rx="4" fill="var(--accent-soft)" />
          <rect x="78" y="108" width="228" height="10" rx="4" fill="var(--accent-soft)" />
          <rect x="78" y="134" width="176" height="10" rx="4" fill="var(--accent-soft)" />
          <rect x="78" y="160" width="210" height="10" rx="4" fill="var(--accent-soft)" />
          <motion.rect
            x="62"
            y="44"
            width="296"
            height="3"
            rx="2"
            fill="var(--accent)"
            animate={{ y: [44, 170, 44] }}
            transition={{ duration: 2.5, repeat: Number.POSITIVE_INFINITY, ease: 'easeInOut' }}
          />
        </svg>
      </div>
      <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
        Waiting for AI worker schema discovery. Status updates are streamed from supplier notifications.
      </p>
    </div>
  );
}

export default function InventoryImportWizardPage() {
  const { toast } = useToast();
  const token = useToken();
  const notifications = useNotifications();
  const handledNotificationIds = useRef<Set<string>>(new Set());

  const [phase, setPhase] = useState<UploadPhase>('idle');
  const [activeTicket, setActiveTicket] = useState<UploadTicketResponse | null>(null);
  const [activeSession, setActiveSession] = useState<SupplierImportSession | null>(null);
  const [lastFile, setLastFile] = useState<{ name: string; size: number } | null>(null);
  const [lastError, setLastError] = useState('');
  const [refreshing, setRefreshing] = useState(false);

  const phaseLabel = useMemo(() => {
    switch (phase) {
      case 'ticketing':
        return 'Requesting Signed Ticket';
      case 'uploading':
        return 'Streaming File To GCS';
      case 'signaling':
        return 'Signaling Upload Completion';
      case 'processing':
        return 'Discovery Pending';
      case 'ready':
        return 'Discovery Advanced';
      case 'error':
        return 'Upload Failed';
      default:
        return 'Idle';
    }
  }, [phase]);

  const refreshSession = useCallback(async (sessionID?: string) => {
    const resolvedSessionID = (sessionID || activeTicket?.session_id || '').trim();
    if (!resolvedSessionID) return;

    setRefreshing(true);
    try {
      const res = await apiFetch(`/v1/supplier/inventory/imports/${resolvedSessionID}`);
      const body = await res.json().catch(() => ({} as { error?: string }));
      if (!res.ok) {
        throw new Error(body.error || 'Failed to refresh import status');
      }

      const session = body as SupplierImportSession;
      setActiveSession(session);
      if (session.status !== 'UPLOADED' && session.status !== 'DISCOVERING' && session.status !== 'INITIALIZED') {
        setPhase('ready');
      }
    } catch (error) {
      toast((error as Error).message, 'error');
    } finally {
      setRefreshing(false);
    }
  }, [activeTicket?.session_id, toast]);

  const onFileSelected = useCallback(async (file: File) => {
    if (!token) return;

    const extension = `.${(file.name.split('.').pop() || '').toLowerCase()}`;
    if (!ACCEPTED_EXTENSIONS.includes(extension)) {
      toast('Only Excel files (.xlsx or .xls) are supported for ingress.', 'error');
      return;
    }
    if (file.size > MAX_UPLOAD_BYTES) {
      toast('File exceeds 50MB upload limit.', 'error');
      return;
    }

    setLastError('');
    setLastFile({ name: file.name, size: file.size });

    try {
      setPhase('ticketing');
      const ticketRes = await apiFetch('/v1/supplier/inventory/imports/', {
        method: 'POST',
        body: JSON.stringify({
          file_name: file.name,
          file_size_bytes: file.size,
        }),
      });
      const ticketBody = await ticketRes.json().catch(() => ({} as { error?: string }));
      if (!ticketRes.ok) {
        throw new Error(ticketBody.error || 'Failed to acquire upload ticket');
      }

      const ticket = ticketBody as UploadTicketResponse;
      setActiveTicket(ticket);
      setActiveSession({
        session_id: ticket.session_id,
        supplier_id: ticket.supplier_id,
        status: ticket.status,
        file_name: ticket.file_name,
        gcs_path: ticket.gcs_path,
        total_rows: 0,
        created_at: '',
        updated_at: '',
      });

      setPhase('uploading');
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

      setPhase('signaling');
      const uploadedSignalRes = await apiFetch(`/v1/supplier/inventory/imports/${ticket.session_id}/uploaded`, {
        method: 'POST',
        body: JSON.stringify({ gcs_path: ticket.gcs_path }),
      });
      const uploadedSignalBody = await uploadedSignalRes.json().catch(() => ({} as { error?: string }));
      if (!uploadedSignalRes.ok) {
        throw new Error(uploadedSignalBody.error || 'Failed to signal upload completion');
      }

      setPhase('processing');
      await refreshSession(ticket.session_id);
      toast('File uploaded. Discovery has been triggered.', 'success');
    } catch (error) {
      const message = (error as Error).message;
      setLastError(message);
      setPhase('error');
      toast(message, 'error');
    }
  }, [refreshSession, toast, token]);

  useEffect(() => {
    if (phase !== 'processing' || !activeTicket?.session_id) {
      return;
    }

    const timer = setInterval(() => {
      void refreshSession(activeTicket.session_id);
    }, 5000);

    return () => clearInterval(timer);
  }, [activeTicket?.session_id, phase, refreshSession]);

  useEffect(() => {
    if (!activeTicket?.session_id) {
      return;
    }

    for (const notification of notifications.items) {
      if (handledNotificationIds.current.has(notification.id)) {
        continue;
      }
      handledNotificationIds.current.add(notification.id);

      const normalizedType = notification.type.toUpperCase();
      if (normalizedType !== 'STATUS_UPDATE' && normalizedType !== 'INVENTORY_IMPORT_STATUS_UPDATE') {
        continue;
      }

      const payloadSessionID = parseSessionIDFromPayload(notification.payload || '');
      if (payloadSessionID && payloadSessionID !== activeTicket.session_id) {
        continue;
      }

      void refreshSession(activeTicket.session_id);
    }
  }, [activeTicket?.session_id, notifications.items, refreshSession]);

  const disableUpload = phase === 'ticketing' || phase === 'uploading' || phase === 'signaling';
  const canRefresh = Boolean(activeTicket?.session_id) && !refreshing;

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
            <h1 className="md-typescale-headline-medium">Inventory Import Ingress</h1>
            <p className="md-typescale-body-small mt-2" style={{ color: 'var(--muted)' }}>
              Phase 3 bridge: Signed URL ticket, direct GCS upload, and discovery trigger.
            </p>
          </div>
          <Link href="/supplier/inventory">
            <Button variant="outline">Back To Inventory</Button>
          </Link>
        </div>
      </header>

      <BentoGrid>
        <BentoCard size="control" className="p-5">
          <div className="h-full flex flex-col gap-4">
            <div>
              <h2 className="md-typescale-title-medium">1. Signed Upload Ticket</h2>
              <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
                The file never passes through backend pods. A short-lived URL streams it directly to GCS.
              </p>
            </div>

            <UploadDropzone disabled={disableUpload} phase={phase} onFileSelected={(file) => {
              void onFileSelected(file);
            }} />

            <div className="flex flex-wrap gap-2">
              <Button variant="outline" isDisabled={!canRefresh} isLoading={refreshing} onPress={() => {
                void refreshSession();
              }}>
                Refresh Status
              </Button>
              <div className="md-chip md-chip-selected" style={{ cursor: 'default' }}>
                {phaseLabel}
              </div>
            </div>
          </div>
        </BentoCard>

        <BentoCard size="list" className="p-5">
          <div className="h-full flex flex-col gap-3">
            <h2 className="md-typescale-title-medium">2. Upload Contract</h2>
            <div className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
              <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Allowed Types</div>
              <div className="md-typescale-body-small">{ACCEPTED_EXTENSIONS.join(', ')}</div>
            </div>
            <div className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
              <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Max File Size</div>
              <div className="md-typescale-body-small">{bytesToHuman(MAX_UPLOAD_BYTES)}</div>
            </div>
            {lastFile ? (
              <div className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                <div className="flex items-center gap-2">
                  <FileSpreadsheet size={16} style={{ color: 'var(--accent)' }} aria-hidden="true" />
                  <span className="md-typescale-label-medium">{lastFile.name}</span>
                </div>
                <div className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
                  {bytesToHuman(lastFile.size)}
                </div>
              </div>
            ) : null}
            {lastError ? (
              <div className="p-3 md-shape-md md-typescale-body-small" style={{ border: '1px solid var(--danger)', color: 'var(--danger)' }}>
                {lastError}
              </div>
            ) : null}
          </div>
        </BentoCard>

        {phase === 'processing' ? <BentoSkeleton size="anchor" /> : (
          <BentoCard size="anchor" className="p-5">
            <div className="h-full flex flex-col gap-4">
              <h2 className="md-typescale-title-medium">3. Session Snapshot</h2>
              {activeTicket ? (
                <>
                  <div className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                    <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Session ID</div>
                    <div className="font-mono md-typescale-body-small break-all">{activeTicket.session_id}</div>
                  </div>
                  <div className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                    <div className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>GCS Path</div>
                    <div className="font-mono md-typescale-body-small break-all">{activeTicket.gcs_path}</div>
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Current Status</span>
                    <span className="md-chip md-chip-selected" style={{ cursor: 'default' }}>
                      {prettyState(activeSession?.status || activeTicket.status)}
                    </span>
                  </div>
                </>
              ) : (
                <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                  Start by dropping an Excel file to create a new import session.
                </p>
              )}
            </div>
          </BentoCard>
        )}

        <BentoCard size="wide" className="p-5">
          {phase === 'processing' ? (
            <ProcessingScanner />
          ) : (
            <div className="h-full flex flex-col gap-3">
              <h2 className="md-typescale-title-medium">4. Discovery Trigger</h2>
              <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                After direct GCS upload, frontend calls the uploaded signal endpoint and backend emits to Kafka topic
                {' '}
                <span className="font-mono">inventory.import.events</span>
                .
              </p>
              {activeSession?.status === 'MAPPING_REQUIRED' ? (
                <div className="p-3 md-shape-md" style={{ border: '1px solid var(--accent)', background: 'var(--accent-soft)' }}>
                  <div className="md-typescale-label-medium">Discovery Complete</div>
                  <p className="md-typescale-body-small mt-1">
                    Initial schema discovery finished. Continue to mapping in the next phase.
                  </p>
                </div>
              ) : (
                <div className="p-3 md-shape-md" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                  <div className="md-typescale-label-medium">Waiting For Processing</div>
                  <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
                    The page listens via useNotifications for STATUS_UPDATE events and refreshes session state.
                  </p>
                </div>
              )}
            </div>
          )}
        </BentoCard>
      </BentoGrid>
    </div>
  );
}
