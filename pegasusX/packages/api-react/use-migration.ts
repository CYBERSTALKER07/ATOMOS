import { useCallback, useState } from "react";
import type { ApiClient } from "@pegasusx/api-core";
import type {
  JoinRequest,
  JoinRequestsListResponse,
  MigrationOpeningBalancesResponse,
  MigrationProject,
  MigrationProjectTree,
} from "@pegasusx/types";
import { usePolling } from "./usePolling";

export interface PollResourceResult<T> {
  data: T;
  loading: boolean;
  error: string | null;
  refresh: (silent?: boolean) => Promise<void>;
}

/**
 * Polls the supplier migration project list with automatic background backoff.
 */
export function useMigrationProjects(
  api: Pick<ApiClient, "listMigrationProjects"> | null,
  opts: { enabled?: boolean; pollMs?: number } = {},
): PollResourceResult<MigrationProject[]> {
  const { enabled = true, pollMs = 20_000 } = opts;
  const [data, setData] = useState<MigrationProject[]>([]);
  const [loading, setLoading] = useState<boolean>(enabled);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(
    async (silent = false) => {
      if (!api || !enabled) {
        setLoading(false);
        return;
      }
      if (!silent) setLoading(true);
      try {
        const res = await api.listMigrationProjects();
        setData(res.projects ?? []);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "migration_projects_fetch_failed");
      } finally {
        if (!silent) setLoading(false);
      }
    },
    [api, enabled],
  );

  usePolling(
    async (signal) => {
      if (signal.aborted || !api || !enabled) return;
      await refresh(data.length > 0);
    },
    pollMs,
    [api, enabled, pollMs, refresh, data.length],
    { immediate: enabled, hiddenIntervalMs: 60_000 },
  );

  return { data, loading, error, refresh };
}

/**
 * Polls a single migration project tree (bundles, artifacts, sessions).
 */
export function useMigrationProjectTree(
  api: Pick<ApiClient, "getMigrationProject"> | null,
  projectId: string | null | undefined,
  opts: { enabled?: boolean; pollMs?: number } = {},
): PollResourceResult<MigrationProjectTree | null> {
  const { enabled = true, pollMs = 5_000 } = opts;
  const [data, setData] = useState<MigrationProjectTree | null>(null);
  const [loading, setLoading] = useState<boolean>(Boolean(enabled && projectId));
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(
    async (silent = false) => {
      if (!api || !enabled || !projectId) {
        setLoading(false);
        return;
      }
      if (!silent) setLoading(true);
      try {
        const tree = await api.getMigrationProject(projectId);
        setData(tree);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "migration_project_fetch_failed");
      } finally {
        if (!silent) setLoading(false);
      }
    },
    [api, enabled, projectId],
  );

  usePolling(
    async (signal) => {
      if (signal.aborted || !api || !enabled || !projectId) return;
      await refresh(data !== null);
    },
    pollMs,
    [api, enabled, projectId, pollMs, refresh, data !== null],
    { immediate: Boolean(enabled && projectId), hiddenIntervalMs: 30_000 },
  );

  return { data, loading, error, refresh };
}

/**
 * Polls pending or filtered INN company join requests for Supplier or Retailer admins.
 */
export function useJoinRequests(
  fetchFn: ((status?: string) => Promise<JoinRequestsListResponse>) | null,
  status?: string,
  opts: { enabled?: boolean; pollMs?: number } = {},
): PollResourceResult<JoinRequest[]> & { pendingCount: number } {
  const { enabled = true, pollMs = 20_000 } = opts;
  const [data, setData] = useState<JoinRequest[]>([]);
  const [loading, setLoading] = useState<boolean>(enabled);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(
    async (silent = false) => {
      if (!fetchFn || !enabled) {
        setLoading(false);
        return;
      }
      if (!silent) setLoading(true);
      try {
        const res = await fetchFn(status);
        setData(res.requests ?? []);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "join_requests_fetch_failed");
      } finally {
        if (!silent) setLoading(false);
      }
    },
    [fetchFn, status, enabled],
  );

  usePolling(
    async (signal) => {
      if (signal.aborted || !fetchFn || !enabled) return;
      await refresh(data.length > 0);
    },
    pollMs,
    [fetchFn, status, enabled, pollMs, refresh, data.length],
    { immediate: enabled, hiddenIntervalMs: 60_000 },
  );

  const pendingCount = data.filter((r) => r.status === "PENDING").length;
  return { data, pendingCount, loading, error, refresh };
}

/**
 * Polls retailer migrated nasiya opening balances requiring acknowledgement or dispute.
 */
export function useRetailerNasiyaOpeningBalances(
  api: Pick<ApiClient, "listRetailerNasiyaOpeningBalances"> | null,
  opts: { enabled?: boolean; pollMs?: number } = {},
): PollResourceResult<MigrationOpeningBalancesResponse | null> {
  const { enabled = true, pollMs = 30_000 } = opts;
  const [data, setData] = useState<MigrationOpeningBalancesResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(enabled);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(
    async (silent = false) => {
      if (!api || !enabled) {
        setLoading(false);
        return;
      }
      if (!silent) setLoading(true);
      try {
        const res = await api.listRetailerNasiyaOpeningBalances();
        setData(res);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "nasiya_opening_balances_fetch_failed");
      } finally {
        if (!silent) setLoading(false);
      }
    },
    [api, enabled],
  );

  usePolling(
    async (signal) => {
      if (signal.aborted || !api || !enabled) return;
      await refresh(data !== null);
    },
    pollMs,
    [api, enabled, pollMs, refresh, data !== null],
    { immediate: enabled, hiddenIntervalMs: 60_000 },
  );

  return { data, loading, error, refresh };
}
