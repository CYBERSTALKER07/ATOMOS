"use client";

import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { usePortalT } from "@/lib/i18n";
import {
  useMigrationProjects,
  useMigrationProjectTree,
} from "@pegasusx/api-react";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { SUPPLIER_MIGRATION_REFRESH_EVENTS } from "@/lib/supplier-ws-events";
import { useSupplierWsRefresh } from "@/lib/use-supplier-ws-refresh";
import { PageChrome } from "@/components/PageChrome";
import { MigrationProjectSidebar } from "@/components/migrations/MigrationProjectSidebar";
import { MigrationBundleUploader } from "@/components/migrations/MigrationBundleUploader";
import { MigrationSorterTree } from "@/components/migrations/MigrationSorterTree";
import { MigrationColumnMapper } from "@/components/migrations/MigrationColumnMapper";
import { MigrationDryRunAndCutoverPanel } from "@/components/migrations/MigrationDryRunAndCutoverPanel";
import { domainQueryToPack } from "@/components/migrations/pack-labels";

export default function MigrationsPage() {
  return (
    <Suspense fallback={null}>
      <MigrationsPageInner />
    </Suspense>
  );
}

function MigrationsPageInner() {
  const t = usePortalT();
  const router = useRouter();
  const params = useSearchParams();
  const api = useMemo(() => createSupplierApi(), []);
  const supplierId = useMemo(() => supplierScopeId(), []);

  const queryProjectId = params.get("projectId");
  const querySessionId = params.get("sessionId");
  const queryDomain = params.get("domain");
  const preferredPack = domainQueryToPack(queryDomain);

  const [warehouses, setWarehouses] = useState<Array<{ warehouse_id: string; name: string }>>([]);
  const [actionError, setActionError] = useState<string | null>(null);
  const [pendingProjectId, setPendingProjectId] = useState<string | null>(null);
  const [pendingSessionId, setPendingSessionId] = useState<string | null>(null);
  const [liveProgress, setLiveProgress] = useState<{ processed: number; total: number } | null>(
    null,
  );

  const {
    data: projects,
    loading: projectsLoading,
    error: projectsError,
    refresh: refreshProjects,
  } = useMigrationProjects(api, { pollMs: 25_000 });

  const activeProjectId = pendingProjectId || queryProjectId || projects[0]?.project_id || null;
  const effectiveSessionId =
    pendingSessionId !== null ? pendingSessionId || null : querySessionId;

  useEffect(() => {
    if (pendingProjectId !== null && queryProjectId === pendingProjectId) {
      setPendingProjectId(null);
    }
  }, [pendingProjectId, queryProjectId]);

  useEffect(() => {
    if (pendingSessionId !== null && (querySessionId ?? "") === pendingSessionId) {
      setPendingSessionId(null);
    }
  }, [pendingSessionId, querySessionId]);

  const {
    data: tree,
    error: treeError,
    refresh: refreshTree,
  } = useMigrationProjectTree(api, activeProjectId, {
    enabled: Boolean(activeProjectId),
    pollMs: 8_000,
  });

  useEffect(() => {
    void api
      .getSupplierTopology()
      .then((topo) => {
        const list = (topo.warehouses ?? []).map((w) => ({
          warehouse_id: w.warehouse_id,
          name: w.name || w.warehouse_id,
        }));
        setWarehouses(list);
      })
      .catch(() => {
        // Optional helper dropdown
      });
  }, [api]);

  const setQueryParams = useCallback(
    (patch: { projectId?: string | null; bundleId?: string | null; sessionId?: string | null }) => {
      if (patch.projectId !== undefined) {
        setPendingProjectId(patch.projectId ?? null);
      }
      if (patch.sessionId !== undefined) {
        setPendingSessionId(patch.sessionId ?? "");
      }
      const next = new URLSearchParams(params.toString());
      if (patch.projectId !== undefined) {
        if (patch.projectId) next.set("projectId", patch.projectId);
        else next.delete("projectId");
      }
      if (patch.bundleId !== undefined) {
        if (patch.bundleId) next.set("bundleId", patch.bundleId);
        else next.delete("bundleId");
      }
      if (patch.sessionId !== undefined) {
        if (patch.sessionId) next.set("sessionId", patch.sessionId);
        else next.delete("sessionId");
      }
      const qs = next.toString();
      router.replace(qs ? `/migrations?${qs}` : "/migrations");
    },
    [params, router],
  );

  const activeTree =
    tree && activeProjectId && tree.project.project_id === activeProjectId ? tree : null;

  const selectedSession = useMemo(() => {
    if (!activeTree?.sessions?.length) return null;
    if (effectiveSessionId) {
      const exact = activeTree.sessions.find((s) => s.session_id === effectiveSessionId);
      if (exact) return exact;
    }
    if (preferredPack) {
      const byPack = activeTree.sessions.find((s) => s.pack === preferredPack);
      if (byPack) return byPack;
    }
    return activeTree.sessions[0] ?? null;
  }, [activeTree?.sessions, effectiveSessionId, preferredPack]);

  const refreshAll = useCallback(async () => {
    setLiveProgress(null);
    await Promise.all([refreshProjects(true), refreshTree(true)]);
  }, [refreshProjects, refreshTree]);

  useSupplierWsRefresh(
    () => {
      void refreshAll();
    },
    {
      eventTypes: SUPPLIER_MIGRATION_REFRESH_EVENTS,
      debounceMs: 400,
    },
  );

  const combinedError = actionError || projectsError || treeError;

  return (
    <PageChrome
      icon="inventory"
      title={t("supplier_portal.migrations.title")}
      description={t("supplier_portal.migrations.subtitle")}
      loading={projectsLoading && projects.length === 0}
      error={combinedError}
    >
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        <div className="lg:col-span-4">
          <MigrationProjectSidebar
            api={api}
            supplierId={supplierId}
            projects={projects}
            selectedProjectId={activeProjectId}
            warehouses={warehouses}
            onSelectProject={(pid) => {
              setActionError(null);
              setQueryParams({ projectId: pid, bundleId: null, sessionId: null });
            }}
            onProjectCreated={(created) => {
              setActionError(null);
              void refreshProjects(true);
              setQueryParams({ projectId: created.project_id, bundleId: null, sessionId: null });
            }}
            onError={setActionError}
          />
        </div>

        <div className="lg:col-span-8 flex flex-col gap-6">
          {!activeProjectId ? (
            <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-6 text-sm text-[var(--muted)]">
              {t("supplier_portal.migrations.no_projects")}
            </div>
          ) : (
            <>
              <MigrationBundleUploader
                key={activeProjectId}
                api={api}
                projectId={activeProjectId}
                onBundleSorted={(res) => {
                  setActionError(null);
                  const firstSession = res.sessions?.[0]?.session_id ?? null;
                  setQueryParams({
                    projectId: res.bundle.project_id || activeProjectId,
                    bundleId: res.bundle.bundle_id,
                    sessionId: firstSession,
                  });
                  void refreshAll();
                }}
                onError={setActionError}
              />

              {activeTree?.bundles.map((bundleTree) => (
                <MigrationSorterTree
                  key={bundleTree.bundle.bundle_id}
                  api={api}
                  projectId={activeProjectId}
                  bundleTree={bundleTree}
                  sessions={activeTree.sessions}
                  selectedSessionId={selectedSession?.session_id ?? null}
                  onSelectSession={(sid) =>
                    setQueryParams({
                      projectId: activeProjectId,
                      bundleId: bundleTree.bundle.bundle_id,
                      sessionId: sid,
                    })
                  }
                  onSortUpdated={() => {
                    setActionError(null);
                    void refreshAll();
                  }}
                  onError={setActionError}
                />
              ))}

              {selectedSession && (
                <MigrationColumnMapper
                  key={selectedSession.session_id}
                  api={api}
                  sessionId={selectedSession.session_id}
                  onMappingConfirmed={() => {
                    setActionError(null);
                    void refreshAll();
                  }}
                  onError={setActionError}
                />
              )}

              {activeTree && (
                <MigrationDryRunAndCutoverPanel
                  key={activeProjectId}
                  api={api}
                  tree={activeTree}
                  selectedSession={selectedSession}
                  liveApplyProgress={liveProgress}
                  onRefresh={refreshAll}
                  onError={setActionError}
                />
              )}
            </>
          )}
        </div>
      </div>
    </PageChrome>
  );
}
