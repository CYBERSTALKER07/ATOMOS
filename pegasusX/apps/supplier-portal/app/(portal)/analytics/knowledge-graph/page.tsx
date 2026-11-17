"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { ApiError } from "@pegasusx/api-core";
import {
  ONTOLOGY_LINK_TYPES,
  ONTOLOGY_OBJECT_TYPES,
  ONTOLOGY_SECURITY_MARKINGS,
  clampTraversalDepth,
  createSupplierOntologyApi,
  extractObjectPrimaryLabel,
  formatOntologyMarkingLabel,
  type OntologyNeighborResponse,
  type OntologyObject,
  type OntologyRebuildResponse,
  type OntologyTypeDescriptor,
  type OntologyTypesResponse,
  type SupplierKnowledgeGraph,
} from "@/lib/ontology-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import Icon from "@/components/Icon";
import { OntologyMarkingList } from "@/components/ontology/OntologyMarkingsBadge";
import { OntologyTopologyCanvas } from "@/components/ontology/OntologyTopologyCanvas";
import { OntologyPropertiesDrawer } from "@/components/ontology/OntologyPropertiesDrawer";

function formatApiError(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.payload && typeof err.payload === "object") {
    const payload = err.payload as Record<string, unknown>;
    const detail = [payload.error, payload.message, payload.detail]
      .filter((part): part is string => typeof part === "string" && part.trim() !== "")
      .join(" · ");
    if (detail) return detail;
  }
  return err instanceof Error && err.message.trim() ? err.message : fallback;
}

function objectKey(objectType: string, objectId: string): string {
  return `${objectType}:${objectId}`;
}

export default function KnowledgeGraphPage() {
  return (
    <Suspense fallback={null}>
      <KnowledgeGraphInner />
    </Suspense>
  );
}

function KnowledgeGraphInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const ontologyApi = useMemo(() => createSupplierOntologyApi(), []);

  const initialTypeParam = (searchParams.get("type") || "ALL").trim();
  const initialIdParam = (searchParams.get("id") || "").trim();
  const initialDepthParam = clampTraversalDepth(
    searchParams.get("depth") ? Number(searchParams.get("depth")) : 2,
  );
  const initialCellParam = (searchParams.get("cell_id") || "").trim();
  const initialMarkingParam = (searchParams.get("marking") || "ALL").trim();
  const initialLinkTypeParam = (searchParams.get("link_type") || "ALL").trim();
  const initialQueryParam = (searchParams.get("q") || "").trim();

  const [selectedType, setSelectedType] = useState<string>(initialTypeParam || "ALL");
  const [searchQuery, setSearchQuery] = useState<string>(initialQueryParam);
  const [appliedQuery, setAppliedQuery] = useState<string>(initialQueryParam);
  const [cellFilter, setCellFilter] = useState<string>(initialCellParam);
  const [appliedCell, setAppliedCell] = useState<string>(initialCellParam);
  const [markingFilter, setMarkingFilter] = useState<string>(initialMarkingParam || "ALL");
  const [depth, setDepth] = useState<1 | 2 | 3 | 4>(initialDepthParam);
  const [linkTypeFilter, setLinkTypeFilter] = useState<string>(initialLinkTypeParam || "ALL");

  const [typesCatalog, setTypesCatalog] = useState<OntologyTypesResponse | null>(null);
  const [objects, setObjects] = useState<OntologyObject[]>([]);
  const [totalObjects, setTotalObjects] = useState<number>(0);
  const [objectsPiiMasked, setObjectsPiiMasked] = useState<boolean>(false);
  const [objectsFinanceMasked, setObjectsFinanceMasked] = useState<boolean>(false);

  const [ekg, setEkg] = useState<SupplierKnowledgeGraph | null>(null);

  const [focusedRef, setFocusedRef] = useState<{ objectType: string; objectId: string } | null>(
    initialTypeParam !== "ALL" && initialIdParam
      ? { objectType: initialTypeParam, objectId: initialIdParam }
      : null,
  );
  const [traversalTrail, setTraversalTrail] = useState<Array<{ objectType: string; objectId: string; label: string }>>([]);
  const [neighbors, setNeighbors] = useState<OntologyNeighborResponse | null>(null);
  const [neighborsLoading, setNeighborsLoading] = useState<boolean>(false);
  const [neighborsError, setNeighborsError] = useState<string | null>(null);
  const [inspectedObject, setInspectedObject] = useState<OntologyObject | null>(null);

  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [rebuilding, setRebuilding] = useState<boolean>(false);
  const [rebuildResult, setRebuildResult] = useState<OntologyRebuildResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [showEkgDetails, setShowEkgDetails] = useState<boolean>(false);

  const autoBootstrappedRef = useRef<boolean>(false);

  const descriptorByType = useMemo(() => {
    const map = new Map<string, OntologyTypeDescriptor>();
    for (const desc of typesCatalog?.types ?? []) {
      map.set(desc.object_type, desc);
    }
    return map;
  }, [typesCatalog]);

  const allTypeNames = useMemo(() => {
    const ordered: string[] = [...ONTOLOGY_OBJECT_TYPES];
    for (const desc of typesCatalog?.types ?? []) {
      if (!ordered.includes(desc.object_type)) {
        ordered.push(desc.object_type);
      }
    }
    return ordered;
  }, [typesCatalog]);

  const availableLinkTypes = useMemo(() => {
    const set = new Set<string>(ONTOLOGY_LINK_TYPES);
    for (const lt of typesCatalog?.link_types ?? []) {
      set.add(lt);
    }
    return [...set];
  }, [typesCatalog]);

  const availableMarkings = useMemo(() => {
    const set = new Set<string>(ONTOLOGY_SECURITY_MARKINGS);
    for (const m of typesCatalog?.security_markings ?? []) {
      set.add(m);
    }
    return [...set];
  }, [typesCatalog]);

  const typeCountsFromObjects = useMemo(() => {
    const counts = new Map<string, number>();
    for (const obj of objects) {
      counts.set(obj.object_type, (counts.get(obj.object_type) ?? 0) + 1);
    }
    return counts;
  }, [objects]);

  const traverseObject = useCallback(
    async (
      objectType: string,
      objectId: string,
      opts?: {
        depth?: 1 | 2 | 3 | 4;
        cellId?: string;
        linkType?: string;
        pushTrail?: boolean;
        seedObject?: OntologyObject;
      },
    ) => {
      const targetDepth = opts?.depth ?? depth;
      const targetCell = opts?.cellId !== undefined ? opts.cellId : appliedCell;
      const targetLinkType = opts?.linkType !== undefined ? opts.linkType : linkTypeFilter;

      setFocusedRef({ objectType, objectId });
      if (opts?.seedObject) {
        setInspectedObject(opts.seedObject);
      }
      setNeighborsLoading(true);
      setNeighborsError(null);

      try {
        const res = await ontologyApi.getNeighbors(objectType, objectId, {
          depth: targetDepth,
          cell_id: targetCell || undefined,
          link_type: targetLinkType !== "ALL" ? targetLinkType : undefined,
        });
        setNeighbors(res);
        setInspectedObject(res.root);

        if (opts?.pushTrail !== false) {
          const label = extractObjectPrimaryLabel(res.root);
          setTraversalTrail((prev) => {
            const existingIdx = prev.findIndex(
              (item) => item.objectType === objectType && item.objectId === objectId,
            );
            if (existingIdx >= 0) {
              return prev.slice(0, existingIdx + 1);
            }
            return [...prev, { objectType, objectId, label }];
          });
        }
      } catch (err) {
        setNeighborsError(formatApiError(err, tRef.current("supplier_portal.analytics.knowledge_graph.load_failed")));
      } finally {
        setNeighborsLoading(false);
      }
    },
    [appliedCell, depth, linkTypeFilter, ontologyApi],
  );

  const traverseObjectRef = useRef(traverseObject);
  traverseObjectRef.current = traverseObject;
  const hasLoadedOnceRef = useRef<boolean>(false);

  const loadWorkspace = useCallback(
    async (silent = false) => {
      const runSilent = silent || hasLoadedOnceRef.current;
      if (!runSilent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      setActionError(null);

      try {
        const [typesSettled, objectsSettled, ekgSettled] = await Promise.allSettled([
          ontologyApi.listTypes(),
          ontologyApi.listObjects({
            type: selectedType !== "ALL" ? selectedType : undefined,
            q: appliedQuery || undefined,
            cell_id: appliedCell || undefined,
            marking: markingFilter !== "ALL" ? markingFilter : undefined,
            limit: 100,
          }),
          ontologyApi.getKnowledgeGraph(),
        ]);

        const failures: string[] = [];
        const fallbackMsg = tRef.current("supplier_portal.analytics.knowledge_graph.load_failed");

        if (typesSettled.status === "fulfilled") {
          setTypesCatalog(typesSettled.value);
        } else {
          failures.push(formatApiError(typesSettled.reason, fallbackMsg));
        }

        let loadedObjects: OntologyObject[] = [];
        if (objectsSettled.status === "fulfilled") {
          const objectsRes = objectsSettled.value;
          loadedObjects = objectsRes.objects ?? [];
          setObjects(loadedObjects);
          setTotalObjects(objectsRes.total ?? loadedObjects.length);
          setObjectsPiiMasked(Boolean(objectsRes.pii_masked));
          setObjectsFinanceMasked(Boolean(objectsRes.finance_masked));
        } else {
          failures.push(formatApiError(objectsSettled.reason, fallbackMsg));
        }

        if (ekgSettled.status === "fulfilled") {
          setEkg(ekgSettled.value);
        } else {
          failures.push(formatApiError(ekgSettled.reason, fallbackMsg));
        }

        const anySucceeded =
          typesSettled.status === "fulfilled" ||
          objectsSettled.status === "fulfilled" ||
          ekgSettled.status === "fulfilled";

        if (!anySucceeded) {
          setError(failures[0] ?? fallbackMsg);
          return;
        }
        if (failures.length > 0) {
          setActionError(failures[0]);
        }
        hasLoadedOnceRef.current = true;

        if (!autoBootstrappedRef.current) {
          autoBootstrappedRef.current = true;
          if (initialTypeParam !== "ALL" && initialIdParam) {
            void traverseObjectRef.current(initialTypeParam, initialIdParam, {
              depth: initialDepthParam,
              cellId: initialCellParam,
              linkType: initialLinkTypeParam,
            });
          } else if (loadedObjects.length > 0) {
            const first = loadedObjects[0];
            setInspectedObject(first);
            void traverseObjectRef.current(first.object_type, first.object_id, {
              depth: initialDepthParam,
              cellId: initialCellParam,
              linkType: initialLinkTypeParam,
              seedObject: first,
            });
          }
        }
      } catch (err) {
        setError(formatApiError(err, tRef.current("supplier_portal.analytics.knowledge_graph.load_failed")));
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [
      appliedCell,
      appliedQuery,
      initialCellParam,
      initialDepthParam,
      initialIdParam,
      initialLinkTypeParam,
      initialTypeParam,
      markingFilter,
      ontologyApi,
      selectedType,
    ],
  );

  useEffect(() => {
    void loadWorkspace();
  }, [loadWorkspace]);

  useSupplierSessionReconcile(() => {
    void loadWorkspace(true);
  });

  const handleDepthChange = (nextDepth: 1 | 2 | 3 | 4) => {
    setDepth(nextDepth);
    if (focusedRef) {
      void traverseObject(focusedRef.objectType, focusedRef.objectId, {
        depth: nextDepth,
        pushTrail: false,
      });
    }
  };

  const handleLinkTypeChange = (nextLinkType: string) => {
    setLinkTypeFilter(nextLinkType);
    if (focusedRef) {
      void traverseObject(focusedRef.objectType, focusedRef.objectId, {
        linkType: nextLinkType,
        pushTrail: false,
      });
    }
  };

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setAppliedQuery(searchQuery.trim());
    setAppliedCell(cellFilter.trim());
  };

  const handleResetFilters = () => {
    setSelectedType("ALL");
    setSearchQuery("");
    setAppliedQuery("");
    setCellFilter("");
    setAppliedCell("");
    setMarkingFilter("ALL");
    setLinkTypeFilter("ALL");
    setDepth(2);
    setTraversalTrail([]);
    if (objects.length > 0) {
      const first = objects[0];
      void traverseObject(first.object_type, first.object_id, {
        depth: 2,
        cellId: "",
        linkType: "ALL",
        seedObject: first,
      });
    } else {
      setFocusedRef(null);
      setNeighbors(null);
      setInspectedObject(null);
    }
  };

  const handleRebuild = async () => {
    setRebuilding(true);
    setActionError(null);
    try {
      const idempotencyKey = `ontology-rebuild-${
        typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
          ? crypto.randomUUID()
          : Date.now().toString(36)
      }`;
      const res = await ontologyApi.rebuild(
        { cell_id: appliedCell || cellFilter.trim() || undefined },
        idempotencyKey,
      );
      setRebuildResult(res);
      await loadWorkspace(true);
      if (focusedRef) {
        await traverseObject(focusedRef.objectType, focusedRef.objectId, { pushTrail: false });
      }
    } catch (err) {
      setActionError(formatApiError(err, t("supplier_portal.analytics.knowledge_graph.load_failed")));
    } finally {
      setRebuilding(false);
    }
  };

  const activeDescriptor = inspectedObject
    ? descriptorByType.get(inspectedObject.object_type)
    : undefined;

  const isInspectingRoot = Boolean(
    inspectedObject &&
      neighbors?.root &&
      objectKey(inspectedObject.object_type, inspectedObject.object_id) ===
        objectKey(neighbors.root.object_type, neighbors.root.object_id),
  );

  const anyPiiMasked = objectsPiiMasked || Boolean(neighbors?.pii_masked);
  const anyFinanceMasked = objectsFinanceMasked || Boolean(neighbors?.finance_masked);

  return (
    <PageChrome
      icon="topology"
      title={t("supplier_portal.analytics.knowledge_graph.text.enterprise_knowledge_graph")}
      description={t("supplier_portal.analytics.knowledge_graph.subtitle")}
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            className="md-btn md-btn-filled inline-flex items-center gap-2 px-4 py-2 text-sm"
            disabled={rebuilding}
            onClick={() => void handleRebuild()}
          >
            <Icon name="refresh" size={16} />
            {rebuilding
              ? t("supplier_portal.analytics.knowledge_graph.rebuilding_button")
              : t("supplier_portal.analytics.knowledge_graph.rebuild_button")}
          </button>
          <Link href={"/analytics" as Route} className="md-btn md-btn-text px-3 py-2 text-sm">
            {t("supplier_portal.analytics.knowledge_graph.back_to_analytics")}
          </Link>
        </div>
      }
    >
      <div className="flex flex-col gap-6">
        {actionError ? (
          <div
            role="alert"
            className="desk-card flex items-center justify-between gap-3 px-4 py-3 text-xs"
            style={{
              borderColor: "color-mix(in srgb, var(--desk-danger, #dc2626) 40%, var(--desk-border))",
              background: "var(--desk-danger-soft, rgba(220, 38, 38, 0.08))",
              color: "var(--desk-danger, #dc2626)",
            }}
          >
            <span>{actionError}</span>
            <button type="button" onClick={() => setActionError(null)} className="underline">
              ×
            </button>
          </div>
        ) : null}

        {rebuildResult ? (
          <div
            role="status"
            className="desk-card flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm"
            style={{
              borderColor: "color-mix(in srgb, var(--desk-success, #16a34a) 40%, var(--desk-border))",
              background: "var(--desk-success-soft, rgba(22, 163, 74, 0.08))",
              color: "var(--desk-text-primary)",
            }}
          >
            <div className="flex flex-col gap-0.5">
              <div className="flex flex-wrap items-center gap-2 font-medium">
                <span className="rounded border px-2 py-0.5 font-mono text-xs">{rebuildResult.status}</span>
                <span>
                  {t("supplier_portal.analytics.knowledge_graph.rebuild_success", {
                    objects: rebuildResult.objects_upserted,
                    edges: rebuildResult.edges_upserted,
                    durationMs: rebuildResult.duration_ms,
                    batches: rebuildResult.batches_committed,
                  })}
                </span>
              </div>
              {rebuildResult.tables_scanned?.length > 0 ? (
                <p className="font-mono text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                  {rebuildResult.cell_id} · {rebuildResult.tables_scanned.join(", ")}
                </p>
              ) : null}
            </div>
            <span className="font-mono text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
              {rebuildResult.rebuilt_at}
            </span>
          </div>
        ) : null}

        <KpiStatGrid columns={4}>
          <KpiStatCard
            label={t("supplier_portal.analytics.knowledge_graph.kpi_total_objects")}
            value={totalObjects.toLocaleString()}
          />
          <KpiStatCard
            label={t("supplier_portal.analytics.knowledge_graph.kpi_entity_types")}
            value={allTypeNames.length}
          />
          <KpiStatCard
            label={t("supplier_portal.analytics.knowledge_graph.kpi_ekg_nodes")}
            value={(ekg?.ontology_objects_count ?? ekg?.nodes.length ?? 0).toLocaleString()}
          />
          <KpiStatCard
            label={t("supplier_portal.analytics.knowledge_graph.kpi_ekg_edges")}
            value={(ekg?.ontology_edges_count ?? ekg?.edges.length ?? 0).toLocaleString()}
          />
        </KpiStatGrid>

        {(anyPiiMasked || anyFinanceMasked) ? (
          <div
            className="desk-card flex flex-wrap items-center gap-3 px-4 py-3 text-xs"
            style={{
              borderColor: "color-mix(in srgb, var(--desk-warning, #d97706) 40%, var(--desk-border))",
              background: "var(--desk-warning-soft, rgba(217, 119, 6, 0.08))",
              color: "var(--desk-text-primary)",
            }}
          >
            <Icon name="warning" size={16} />
            <div className="flex flex-wrap gap-3">
              {anyPiiMasked ? (
                <span>{t("supplier_portal.analytics.knowledge_graph.redaction_notice_pii")}</span>
              ) : null}
              {anyFinanceMasked ? (
                <span>{t("supplier_portal.analytics.knowledge_graph.redaction_notice_finance")}</span>
              ) : null}
            </div>
          </div>
        ) : null}

        <PortalSection
          icon="topology"
          title={t("supplier_portal.analytics.knowledge_graph.type_filter_label")}
          description={refreshing ? t("supplier_portal.analytics.knowledge_graph.rebuilding_button") : undefined}
          actions={
            <button
              type="button"
              className="md-btn md-btn-text text-xs px-2.5 py-1"
              onClick={handleResetFilters}
            >
              {t("supplier_portal.analytics.knowledge_graph.reset_focus")}
            </button>
          }
        >
          <div className="flex flex-wrap gap-2" role="group" aria-label={t("supplier_portal.analytics.knowledge_graph.type_filter_label")}>
            <button
              type="button"
              aria-pressed={selectedType === "ALL"}
              onClick={() => setSelectedType("ALL")}
              className="md-chip px-3 py-1 text-xs font-medium"
              style={
                selectedType === "ALL"
                  ? {
                      background: "var(--desk-accent)",
                      color: "var(--desk-accent-on)",
                      borderColor: "var(--desk-accent)",
                    }
                  : undefined
              }
            >
              {t("supplier_portal.analytics.knowledge_graph.filter_all_types")}
            </button>
            {allTypeNames.map((typeName) => {
              const descriptor = descriptorByType.get(typeName);
              const count = descriptor?.count ?? typeCountsFromObjects.get(typeName);
              const active = selectedType === typeName;
              return (
                <button
                  key={typeName}
                  type="button"
                  aria-pressed={active}
                  onClick={() => setSelectedType(typeName)}
                  className="md-chip px-3 py-1 text-xs font-mono"
                  style={
                    active
                      ? {
                          background: "var(--desk-accent)",
                          color: "var(--desk-accent-on)",
                          borderColor: "var(--desk-accent)",
                        }
                      : undefined
                  }
                >
                  {typeName}
                  {typeof count === "number" ? ` (${count})` : ""}
                </button>
              );
            })}
          </div>

          <form onSubmit={handleSearchSubmit} className="grid grid-cols-1 gap-3 pt-2 md:grid-cols-12">
            <div className="md:col-span-4">
              <label htmlFor="ontology-search-input" className="portal-label">
                {t("supplier_portal.analytics.knowledge_graph.search_label")}
              </label>
              <PortalInput
                id="ontology-search-input"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t("supplier_portal.analytics.knowledge_graph.search_placeholder")}
              />
            </div>

            <div className="md:col-span-2">
              <label htmlFor="ontology-cell-input" className="portal-label">
                {t("supplier_portal.analytics.knowledge_graph.cell_filter_label")}
              </label>
              <PortalInput
                id="ontology-cell-input"
                value={cellFilter}
                onChange={(e) => setCellFilter(e.target.value)}
                placeholder="cell-uz"
              />
            </div>

            <div className="md:col-span-3">
              <label htmlFor="ontology-marking-select" className="portal-label">
                {t("supplier_portal.analytics.knowledge_graph.marking_filter_label")}
              </label>
              <PortalSelect
                id="ontology-marking-select"
                value={markingFilter}
                onChange={(e) => setMarkingFilter(e.target.value)}
              >
                <option value="ALL">
                  {t("supplier_portal.analytics.knowledge_graph.all_markings")}
                </option>
                {availableMarkings.map((m) => (
                  <option key={m} value={m}>
                    {formatOntologyMarkingLabel(m)}
                  </option>
                ))}
              </PortalSelect>
            </div>

            <div className="md:col-span-3 flex items-end">
              <button
                type="submit"
                className="portal-btn portal-btn--primary w-full justify-center"
              >
                <Icon name="search" size={16} />
                {t("supplier_portal.analytics.knowledge_graph.apply_filters")}
              </button>
            </div>
          </form>
        </PortalSection>

        <div className="grid grid-cols-1 gap-6 xl:grid-cols-12">
          <div className="flex flex-col gap-6 xl:col-span-8">
            <PortalSection
              icon="topology"
              title={t("supplier_portal.analytics.knowledge_graph.traversal_title")}
              actions={
                <div className="flex flex-wrap items-center gap-2">
                  <div className="flex items-center gap-1" role="group" aria-label={t("supplier_portal.analytics.knowledge_graph.depth_label")}>
                    <span className="text-xs mr-1" style={{ color: "var(--desk-text-secondary)" }}>
                      {t("supplier_portal.analytics.knowledge_graph.depth_label")}:
                    </span>
                    {([1, 2, 3, 4] as const).map((d) => (
                      <button
                        key={d}
                        type="button"
                        aria-pressed={depth === d}
                        onClick={() => handleDepthChange(d)}
                        className="md-chip px-2.5 py-0.5 font-mono text-xs"
                        style={
                          depth === d
                            ? {
                                background: "var(--desk-accent)",
                                color: "var(--desk-accent-on)",
                                borderColor: "var(--desk-accent)",
                              }
                            : undefined
                        }
                      >
                        {d}
                      </button>
                    ))}
                  </div>

                  <PortalSelect
                    id="ontology-link-type-select"
                    aria-label={t("supplier_portal.analytics.knowledge_graph.link_type_label")}
                    value={linkTypeFilter}
                    onChange={(e) => handleLinkTypeChange(e.target.value)}
                    className="!w-auto min-w-[150px] text-xs"
                  >
                    <option value="ALL">
                      {t("supplier_portal.analytics.knowledge_graph.all_link_types")}
                    </option>
                    {availableLinkTypes.map((lt) => (
                      <option key={lt} value={lt}>
                        {lt}
                      </option>
                    ))}
                  </PortalSelect>
                </div>
              }
            >
              {traversalTrail.length > 0 ? (
                <nav aria-label="Traversal breadcrumb" className="flex flex-wrap items-center gap-1.5 text-xs">
                  {traversalTrail.map((step, idx) => (
                    <span key={`${step.objectType}:${step.objectId}`} className="inline-flex items-center gap-1.5">
                      {idx > 0 ? (
                        <span style={{ color: "var(--desk-text-tertiary)" }}>→</span>
                      ) : null}
                      <button
                        type="button"
                        onClick={() =>
                          void traverseObject(step.objectType, step.objectId, { pushTrail: true })
                        }
                        className="rounded border px-2 py-0.5 font-mono hover:underline"
                        style={{
                          borderColor: "var(--desk-border)",
                          background:
                            idx === traversalTrail.length - 1
                              ? "var(--desk-accent-soft)"
                              : "var(--desk-surface-raised)",
                          color: "var(--desk-text-primary)",
                        }}
                      >
                        {step.objectType}:{step.label}
                      </button>
                    </span>
                  ))}
                </nav>
              ) : null}

              {neighborsError ? (
                <div
                  role="alert"
                  className="rounded-lg border px-3 py-2 text-xs"
                  style={{
                    borderColor: "color-mix(in srgb, var(--desk-danger, #dc2626) 40%, transparent)",
                    background: "var(--desk-danger-soft, rgba(220, 38, 38, 0.08))",
                    color: "var(--desk-danger, #dc2626)",
                  }}
                >
                  {neighborsError}
                </div>
              ) : null}

              {neighborsLoading ? (
                <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
                  {t("supplier_portal.analytics.knowledge_graph.rebuilding_button")}
                </p>
              ) : neighbors ? (
                <div className="space-y-4">
                  <OntologyTopologyCanvas
                    neighbors={neighbors}
                    inspectedObject={inspectedObject}
                    onSelectObject={(obj) => setInspectedObject(obj)}
                    onTraverseObject={(obj) =>
                      void traverseObject(obj.object_type, obj.object_id, { seedObject: obj })
                    }
                  />

                  <div className="space-y-2">
                    <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color: "var(--desk-text-secondary)" }}>
                      Neighbors ({neighbors.total_nodes}) · Edges ({neighbors.total_edges})
                    </h3>
                    {neighbors.nodes.length === 0 ? (
                      <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.analytics.knowledge_graph.neighbors_empty", {
                          depth: neighbors.depth,
                        })}
                      </p>
                    ) : (
                      <div className="grid grid-cols-1 gap-2.5 md:grid-cols-2">
                        {neighbors.nodes.map((node) => {
                          const obj = node.object;
                          const isSelected =
                            inspectedObject &&
                            objectKey(inspectedObject.object_type, inspectedObject.object_id) ===
                              objectKey(obj.object_type, obj.object_id);
                          return (
                            <div
                              key={`${obj.object_type}:${obj.object_id}:${node.hop}:${node.via_link_type ?? "edge"}`}
                              className="flex flex-col justify-between gap-2 rounded-xl border p-3"
                              style={{
                                borderColor: isSelected ? "var(--desk-accent)" : "var(--desk-border)",
                                background: "var(--desk-canvas)",
                              }}
                            >
                              <div className="flex items-start justify-between gap-2">
                                <button
                                  type="button"
                                  className="text-left min-w-0 flex-1"
                                  onClick={() => setInspectedObject(obj)}
                                >
                                  <div className="flex flex-wrap items-center gap-1.5">
                                    <span className="font-mono text-xs font-semibold" style={{ color: "var(--desk-text-primary)" }}>
                                      {obj.object_type}
                                    </span>
                                    <span className="rounded border px-1.5 py-0.5 font-mono text-[10px]" style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-secondary)" }}>
                                      {t("supplier_portal.analytics.knowledge_graph.hop_badge", { hop: node.hop })}
                                    </span>
                                    {node.via_link_type ? (
                                      <span className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                                        {node.direction === "INBOUND" ? "←" : "→"} {node.via_link_type}
                                      </span>
                                    ) : null}
                                  </div>
                                  <div className="mt-1 text-sm font-medium truncate" style={{ color: "var(--desk-text-primary)" }}>
                                    {extractObjectPrimaryLabel(obj)}
                                  </div>
                                  <div className="font-mono text-xs truncate" style={{ color: "var(--desk-text-tertiary)" }}>
                                    {obj.object_id}
                                  </div>
                                </button>
                                <button
                                  type="button"
                                  className="md-btn md-btn-tonal shrink-0 px-2.5 py-1 text-xs"
                                  onClick={() =>
                                    void traverseObject(obj.object_type, obj.object_id, { seedObject: obj })
                                  }
                                >
                                  {t("supplier_portal.analytics.knowledge_graph.traverse_from_node")}
                                </button>
                              </div>

                              <OntologyMarkingList markings={obj.security_markings} />
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>

                  {neighbors.edges.length > 0 ? (
                    <div className="overflow-x-auto rounded-lg border" style={{ borderColor: "var(--desk-border)" }}>
                      <table className="desk-table w-full text-xs">
                        <thead>
                          <tr style={{ color: "var(--desk-text-secondary)" }}>
                            <th className="p-2.5 text-left font-medium">
                              {t("supplier_portal.analytics.knowledge_graph.text.from")}
                            </th>
                            <th className="p-2.5 text-left font-medium">
                              {t("supplier_portal.analytics.knowledge_graph.text.relation")}
                            </th>
                            <th className="p-2.5 text-left font-medium">
                              {t("supplier_portal.analytics.knowledge_graph.target_label")}
                            </th>
                          </tr>
                        </thead>
                        <tbody>
                          {neighbors.edges.map((edge, idx) => (
                            <tr
                              key={`${edge.source_object_type}:${edge.source_object_id}-${edge.link_type}-${edge.target_object_type}:${edge.target_object_id}-${idx}`}
                              style={{ borderTop: "1px solid var(--desk-border)" }}
                            >
                              <td className="p-2.5 font-mono">
                                {edge.source_object_type}:{edge.source_object_id}
                              </td>
                              <td className="p-2.5 font-mono font-semibold">{edge.link_type}</td>
                              <td className="p-2.5 font-mono">
                                {edge.target_object_type}:{edge.target_object_id}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ) : null}
                </div>
              ) : (
                <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
                  {t("supplier_portal.analytics.knowledge_graph.traversal_empty", { depth })}
                </p>
              )}
            </PortalSection>

            <PortalSection
              icon="catalog"
              title={`${t("supplier_portal.analytics.knowledge_graph.objects_catalog_title")} (${objects.length})`}
            >
              {objects.length === 0 ? (
                <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
                  {t("supplier_portal.analytics.knowledge_graph.objects_empty")}
                </p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="desk-table w-full text-sm">
                    <thead>
                      <tr style={{ color: "var(--desk-text-secondary)" }}>
                        <th className="p-3 text-left font-medium">
                          {t("supplier_portal.ledger.text.type")}
                        </th>
                        <th className="p-3 text-left font-medium">ID</th>
                        <th className="p-3 text-left font-medium">
                          {t("supplier_portal.analytics.knowledge_graph.text.name")}
                        </th>
                        <th className="p-3 text-left font-medium">
                          {t("supplier_portal.analytics.knowledge_graph.markings_label")}
                        </th>
                        <th className="p-3 text-right font-medium">
                          {t("supplier_portal.analytics.knowledge_graph.version_label")}
                        </th>
                        <th className="p-3 text-right font-medium" />
                      </tr>
                    </thead>
                    <tbody>
                      {objects.map((obj) => {
                        const focused =
                          focusedRef?.objectType === obj.object_type &&
                          focusedRef?.objectId === obj.object_id;
                        return (
                          <tr
                            key={`${obj.cell_id}:${obj.object_type}:${obj.object_id}`}
                            style={{
                              borderTop: "1px solid var(--desk-border)",
                              background: focused ? "var(--desk-accent-soft)" : undefined,
                            }}
                          >
                            <td className="p-3 font-mono text-xs font-semibold">
                              {obj.object_type}
                            </td>
                            <td className="p-3 font-mono text-xs">
                              <button
                                type="button"
                                className="text-left hover:underline"
                                onClick={() => setInspectedObject(obj)}
                              >
                                {obj.object_id}
                              </button>
                            </td>
                            <td className="p-3 text-sm">{extractObjectPrimaryLabel(obj)}</td>
                            <td className="p-3">
                              <OntologyMarkingList
                                markings={obj.security_markings}
                                emptyLabel={t("supplier_portal.analytics.knowledge_graph.no_markings")}
                              />
                            </td>
                            <td className="p-3 text-right font-mono text-xs">v{obj.version}</td>
                            <td className="p-3 text-right">
                              <button
                                type="button"
                                className="md-btn md-btn-tonal px-2.5 py-1 text-xs"
                                onClick={() =>
                                  void traverseObject(obj.object_type, obj.object_id, {
                                    seedObject: obj,
                                  })
                                }
                              >
                                {t("supplier_portal.analytics.knowledge_graph.traverse_from_node")}
                              </button>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </PortalSection>
          </div>

          <div className="xl:col-span-4">
            <OntologyPropertiesDrawer
              object={inspectedObject}
              descriptor={activeDescriptor}
              piiMasked={anyPiiMasked}
              financeMasked={anyFinanceMasked}
              isFocusedRoot={isInspectingRoot}
              onTraverse={(obj) =>
                void traverseObject(obj.object_type, obj.object_id, { seedObject: obj })
              }
            />
          </div>
        </div>

        {ekg ? (
          <PortalSection
            icon="overview"
            title={t("supplier_portal.analytics.knowledge_graph.ekg_unified_title")}
            description={
              ekg.source
                ? `${t("supplier_portal.analytics.knowledge_graph.source_label")}: ${ekg.source}`
                : undefined
            }
            actions={
              <button
                type="button"
                className="md-btn md-btn-text text-xs px-3 py-1"
                onClick={() => setShowEkgDetails((prev) => !prev)}
              >
                {showEkgDetails
                  ? t("supplier_portal.analytics.knowledge_graph.ekg_toggle_hide")
                  : t("supplier_portal.analytics.knowledge_graph.ekg_toggle_show", {
                      nodes: ekg.nodes.length,
                      edges: ekg.edges.length,
                    })}
              </button>
            }
          >
            {showEkgDetails ? (
              <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                <div className="overflow-x-auto">
                  <table className="desk-table w-full text-sm">
                    <thead>
                      <tr style={{ color: "var(--desk-text-secondary)" }}>
                        <th className="p-3 text-left font-medium">ID</th>
                        <th className="p-3 text-left font-medium">
                          {t("supplier_portal.ledger.text.type")}
                        </th>
                        <th className="p-3 text-left font-medium">
                          {t("supplier_portal.analytics.knowledge_graph.text.name")}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {ekg.nodes.map((node) => (
                        <tr key={`${node.type}:${node.id}`} style={{ borderTop: "1px solid var(--desk-border)" }}>
                          <td className="p-3 font-mono text-xs">{node.id}</td>
                          <td className="p-3 font-mono text-xs">{node.type}</td>
                          <td className="p-3 text-sm">{node.name || "—"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                <div className="overflow-x-auto">
                  <table className="desk-table w-full text-sm">
                    <thead>
                      <tr style={{ color: "var(--desk-text-secondary)" }}>
                        <th className="p-3 text-left font-medium">
                          {t("supplier_portal.analytics.knowledge_graph.text.from")}
                        </th>
                        <th className="p-3 text-left font-medium">
                          {t("supplier_portal.analytics.knowledge_graph.text.relation")}
                        </th>
                        <th className="p-3 text-left font-medium">
                          {t("supplier_portal.analytics.knowledge_graph.target_label")}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {ekg.edges.map((edge, index) => (
                        <tr
                          key={`${edge.from}-${edge.relation}-${edge.to}-${index}`}
                          style={{ borderTop: "1px solid var(--desk-border)" }}
                        >
                          <td className="p-3 font-mono text-xs">{edge.from}</td>
                          <td className="p-3 font-mono text-xs">{edge.relation}</td>
                          <td className="p-3 font-mono text-xs">{edge.to}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ) : (
              <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                {ekg.nodes.length} EKG nodes · {ekg.edges.length} EKG edges
              </p>
            )}
          </PortalSection>
        ) : null}
      </div>
    </PageChrome>
  );
}
