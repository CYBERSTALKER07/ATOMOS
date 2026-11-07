import { useCallback, useEffect, useState } from 'react';

import { PayloadTerminalApi } from '../api';
import type { ManifestExceptionItem } from '../components/ExceptionsSheet';
import type { ShowToast } from './useToast';

// ─── Manifest exceptions panel ────────────────────────────────────────────────

export function useManifestExceptions({
  token,
  liveSyncRevision,
  showToast,
}: {
  token: string | null;
  liveSyncRevision: number;
  showToast: ShowToast;
}) {
  const [showExceptionsPanel, setShowExceptionsPanel] = useState(false);
  const [manifestExceptions, setManifestExceptions] = useState<ManifestExceptionItem[]>([]);
  const [loadingExceptions, setLoadingExceptions] = useState(false);

  const loadManifestExceptions = useCallback(async () => {
    if (!token) return;
    setLoadingExceptions(true);
    try {
      const data = await PayloadTerminalApi.getManifestExceptions(token);
      setManifestExceptions(Array.isArray(data.exceptions) ? data.exceptions : []);
    } catch (e: unknown) {
      showToast('ERROR', e instanceof Error ? e.message : 'Failed to load exceptions', 'error');
    } finally {
      setLoadingExceptions(false);
    }
  }, [token, showToast]);

  useEffect(() => {
    if (!token) return;
    void loadManifestExceptions();
  }, [token, liveSyncRevision, loadManifestExceptions]);

  return {
    showExceptionsPanel,
    setShowExceptionsPanel,
    manifestExceptions,
    loadingExceptions,
    loadManifestExceptions,
  };
}
