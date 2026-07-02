"use client";

import { ApiClient, ApiError } from "@pegasusx/api-client";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { SupplierAIRecommendation, SupplierAIRecommendationDecision } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { FormAlert } from "@/components/portal";
import { useEffect, useMemo, useState } from "react";

type StatusFilter = "ALL" | "PENDING" | "ACKNOWLEDGED" | "OVERRIDDEN" | "DISMISSED";

const statusFilters: StatusFilter[] = ["ALL", "PENDING", "ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED"];
const decisions: SupplierAIRecommendationDecision[] = ["ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "REOPENED"];

function formatDateTime(value?: string) {
  if (!value) {
    return "-";
  }
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return value;
  }
  return new Date(timestamp).toLocaleString();
}

function formatPercent(value: number) {
  if (!Number.isFinite(value)) {
    return "-";
  }
  return `${Math.round(value * 100)}%`;
}

function errorToMessage(error: unknown) {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "ai_recommendations_unavailable";
}

function isRestricted(error: unknown) {
  return error instanceof ApiError && (error.status === 401 || error.status === 403);
}

export default function SupplierAIRecommendationsPage() {
  const api = useMemo(() => createSupplierApi(), []);
  const [items, setItems] = useState<SupplierAIRecommendation[]>([]);
  const [filter, setFilter] = useState<StatusFilter>("PENDING");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [restricted, setRestricted] = useState(false);
  const [offline, setOffline] = useState(false);
  const [staleMessage, setStaleMessage] = useState<string | null>(null);
  const [lastLoadedAt, setLastLoadedAt] = useState<string | null>(null);
  const [notes, setNotes] = useState<Record<string, string>>({});
  const [pendingDecision, setPendingDecision] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [refreshTick, setRefreshTick] = useState(0);

  useEffect(() => {
    const updateOfflineState = () => setOffline(typeof navigator !== "undefined" && !navigator.onLine);
    updateOfflineState();
    window.addEventListener("online", updateOfflineState);
    window.addEventListener("offline", updateOfflineState);
    return () => {
      window.removeEventListener("online", updateOfflineState);
      window.removeEventListener("offline", updateOfflineState);
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function loadRecommendations() {
      if (items.length === 0) {
        setLoading(true);
      }
      setError(null);
      setRestricted(false);
      setStaleMessage(null);
      try {
        const response = await api.getSupplierAIRecommendations({
          status: filter === "ALL" ? undefined : filter,
          limit: 50,
        });
        if (!cancelled) {
          setItems(response.items);
          setLastLoadedAt(response.updated_at);
        }
      } catch (loadError) {
        if (cancelled) {
          return;
        }
        if (isRestricted(loadError)) {
          setRestricted(true);
        }
        const message = errorToMessage(loadError);
        if (items.length > 0) {
          setStaleMessage(message);
        } else {
          setError(message);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadRecommendations();
    return () => {
      cancelled = true;
    };
  }, [api, filter, refreshTick]);

  useSupplierSessionReconcile(() => {
    setRefreshTick((tick) => tick + 1);
  });

  async function recordDecision(recommendation: SupplierAIRecommendation, decision: SupplierAIRecommendationDecision) {
    setPendingDecision(`${recommendation.recommendation_id}:${decision}`);
    setFeedback(null);
    try {
      const response = await api.recordSupplierAIRecommendationDecision(
        {
          recommendation_id: recommendation.recommendation_id,
          decision,
          note: notes[recommendation.recommendation_id]?.trim(),
        },
        `ai-rec-${recommendation.recommendation_id}-${decision}-${Date.now()}`,
      );
      setItems((current) => current.map((item) => item.recommendation_id === response.recommendation.recommendation_id ? response.recommendation : item));
      setNotes((current) => ({ ...current, [recommendation.recommendation_id]: "" }));
      setFeedback(`${decision.toLowerCase()} recorded for ${recommendation.aggregate_type} ${recommendation.aggregate_id}.`);
    } catch (decisionError) {
      setFeedback(`Decision failed: ${errorToMessage(decisionError)}`);
    } finally {
      setPendingDecision(null);
    }
  }

  return (
    <PageChrome
      icon="ai"
      title="AI recommendation review"
      description={`Supplier-scoped advisory outputs with explanation evidence and human decision authority.${lastLoadedAt ? ` Last refreshed ${formatDateTime(lastLoadedAt)}.` : ""}`}
      loading={loading && items.length === 0}
      skeletonVariant="table"
      error={error}
      actions={
        <button className="portal-btn portal-btn--outline" type="button" onClick={() => setRefreshTick((value) => value + 1)}>
          Refresh
        </button>
      }
    >
      <section className="desk-card p-4 mb-6">
        <div className="flex flex-wrap gap-3">
          {statusFilters.map((nextFilter) => (
            <button
              className={filter === nextFilter ? "portal-btn portal-btn--primary" : "portal-btn portal-btn--outline"}
              key={nextFilter}
              type="button"
              onClick={() => setFilter(nextFilter)}
            >
              {nextFilter}
            </button>
          ))}
        </div>
      </section>

      {restricted ? <FormAlert variant="error">Access restricted for this supplier session.</FormAlert> : null}
      {offline ? <FormAlert>You are offline. Showing the last loaded recommendations.</FormAlert> : null}
      {staleMessage ? <FormAlert>{staleMessage}</FormAlert> : null}
      {feedback ? <FormAlert>{feedback}</FormAlert> : null}

      {!restricted && !error && items.length === 0 && !loading ? (
        <section className="desk-card p-6">
          <h2 className="md-typescale-title-large">No recommendations</h2>
          <p className="md-typescale-body-medium mt-2" style={{ color: "var(--desk-text-secondary)" }}>
            No {filter.toLowerCase()} advisory rows are available for this supplier.
          </p>
        </section>
      ) : null}

      {items.length > 0 ? (
        <section className="grid gap-4">
          {items.map((recommendation) => (
            <article className="md-card md-shape-md p-5" key={recommendation.recommendation_id}>
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <div className="flex flex-wrap items-center gap-3">
                    <h2 className="md-typescale-title-large">{recommendation.action}</h2>
                    <span className="md-chip">{recommendation.status}</span>
                    <span className="md-chip">Confidence {formatPercent(recommendation.confidence)}</span>
                  </div>
                  <p className="md-typescale-body-medium mt-3 max-w-4xl">{recommendation.explanation}</p>
                  <p className="md-typescale-body-small mt-3" style={{ color: "var(--color-md-outline)" }}>
                    {recommendation.aggregate_type} {recommendation.aggregate_id} · Source {recommendation.source} · Generated {formatDateTime(recommendation.generated_at)}
                  </p>
                </div>
                <div className="text-left lg:text-right">
                  <p className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>Score</p>
                  <p className="md-typescale-title-large">{recommendation.score.toFixed(2)}</p>
                </div>
              </div>

              <div className="mt-5 grid gap-4 lg:grid-cols-2">
                <div>
                  <p className="md-typescale-label-medium mb-2" style={{ color: "var(--color-md-outline)" }}>Reason codes</p>
                  <div className="flex flex-wrap gap-2">
                    {recommendation.reason_codes.map((code) => (
                      <span className="md-chip" key={code}>{code}</span>
                    ))}
                  </div>
                </div>
                <div>
                  <p className="md-typescale-label-medium mb-2" style={{ color: "var(--color-md-outline)" }}>Evidence</p>
                  <div className="grid gap-2">
                    {recommendation.evidence.map((evidence) => (
                      <div className="flex flex-wrap gap-2 md-typescale-body-small" key={`${evidence.label}:${evidence.value}`}>
                        <span style={{ color: "var(--color-md-outline)" }}>{evidence.label}</span>
                        <span>{evidence.value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {(recommendation.decision || recommendation.decided_at) ? (
                <div className="mt-5 md-card md-shape-sm p-3" style={{ background: "var(--color-md-surface-container)" }}>
                  <p className="md-typescale-body-small">
                    Decision {recommendation.decision || recommendation.status} by {recommendation.decided_by || "supplier operator"} at {formatDateTime(recommendation.decided_at)}
                  </p>
                  {recommendation.decision_note ? (
                    <p className="md-typescale-body-small mt-1" style={{ color: "var(--color-md-outline)" }}>{recommendation.decision_note}</p>
                  ) : null}
                </div>
              ) : null}

              <div className="mt-5 grid gap-3 lg:grid-cols-[1fr_auto] lg:items-end">
                <label className="grid gap-2 md-typescale-label-medium">
                  Decision note
                  <textarea
                    className="md-input-outlined min-h-24 p-3 md-typescale-body-medium"
                    value={notes[recommendation.recommendation_id] ?? ""}
                    onChange={(event) => setNotes((current) => ({ ...current, [recommendation.recommendation_id]: event.target.value }))}
                    placeholder="Add operator rationale"
                  />
                </label>
                <div className="flex flex-wrap gap-2">
                  {decisions.map((decision) => {
                    const pendingKey = `${recommendation.recommendation_id}:${decision}`;
                    return (
                      <button
                        className={decision === "OVERRIDDEN" ? "md-btn md-btn-filled" : "md-btn md-btn-outlined"}
                        disabled={pendingDecision !== null}
                        key={decision}
                        type="button"
                        onClick={() => void recordDecision(recommendation, decision)}
                      >
                        {pendingDecision === pendingKey ? "Recording..." : decision}
                      </button>
                    );
                  })}
                </div>
              </div>
            </article>
          ))}
        </section>
      ) : null}
    </PageChrome>
  );
}