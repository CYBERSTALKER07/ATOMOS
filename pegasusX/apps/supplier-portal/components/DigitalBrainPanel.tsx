"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { ApiError } from "@pegasusx/api-core";
import type {
  ForecastConfidence,
  SparsityGateResult,
  SupplierKnowledgeGraph,
  TwinOpsRouteView,
} from "@pegasusx/types";
import { brainForecastLine, forecastConfidenceFromDemand, isForecastBlocked } from "@pegasusx/types";
import { SourceChip } from "@pegasusx/ui-kit/portal";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";
import SignalIngestOpsPanel from "@/components/SignalIngestOpsPanel";
import { ForecastAccuracyPanel } from "@/components/settings/planning";
import { createSupplierApi } from "@/lib/api";
import { formatForecastUpdatedAt, isForecastStale } from "@/lib/forecast-confidence";

const api = createSupplierApi();

type TwinPlane = "last_mile" | "factory";

export default function DigitalBrainPanel({ sku = "", retailerId = "" }: { sku?: string; retailerId?: string }) {
  const [confidence, setConfidence] = useState<ForecastConfidence | null>(null);
  const [generatedAt, setGeneratedAt] = useState<string | undefined>();
  const [sparsity, setSparsity] = useState<SparsityGateResult | null>(null);
  const [sparsityError, setSparsityError] = useState<string | null>(null);
  const [retailer, setRetailer] = useState(retailerId);
  const [graph, setGraph] = useState<SupplierKnowledgeGraph | null>(null);
  const [twinPlane, setTwinPlane] = useState<TwinPlane>("last_mile");
  const [routes, setRoutes] = useState<TwinOpsRouteView[] | null>(null);
  const [twinError, setTwinError] = useState<string | null>(null);
  const [agentStatus, setAgentStatus] = useState<string | null>(null);

  useEffect(() => {
    setRetailer(retailerId);
  }, [retailerId]);

  useEffect(() => {
    let cancelled = false;
    api
      .getSupplierDemandToday()
      .then((demand) => {
        if (cancelled) return;
        setGeneratedAt(demand.generated_at);
        setConfidence(forecastConfidenceFromDemand(demand));
      })
      .catch(() => {
        if (!cancelled) {
          setConfidence({ label: "insufficient_history", blocked_reason: "demand_unavailable" });
        }
      });
    api
      .getSupplierKnowledgeGraph()
      .then((resp) => {
        if (!cancelled) setGraph(resp);
      })
      .catch(() => {
        if (!cancelled) setGraph(null);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (twinPlane === "factory") {
      setRoutes(null);
      setTwinError("unavailable");
      return;
    }
    let cancelled = false;
    setTwinError(null);
    api
      .listTwinActiveRoutes()
      .then((rows) => {
        if (!cancelled) setRoutes(rows ?? []);
      })
      .catch((err) => {
        if (!cancelled) {
          setRoutes(null);
          setTwinError(err instanceof Error ? err.message : "twin_unavailable");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [twinPlane]);

  const checkSparsity = useCallback(async () => {
    const id = retailer.trim();
    if (!id) return;
    setSparsityError(null);
    try {
      setSparsity(await api.checkPlanningSparsity(id));
    } catch (err) {
      setSparsity(null);
      setSparsityError(err instanceof Error ? err.message : "sparsity_unavailable");
    }
  }, [retailer]);

  useEffect(() => {
    if (retailerId.trim()) {
      void checkSparsity();
    }
  }, [retailerId, checkSparsity]);

  const invokeAgent = async (action: "approve_insight" | "open_supply_request" | "broadcast_template") => {
    setAgentStatus(null);
    try {
      const resp = await api.invokeGovernedPlanningAgent(
        { action, idempotency_key: `brain-${action}-${Date.now()}`, target_id: sku || "network" },
        `brain-${action}-${Date.now()}`,
      );
      setAgentStatus(`${resp.action} ${resp.status}`);
    } catch (err) {
      const status = err instanceof ApiError ? err.status : 0;
      setAgentStatus(status === 409 ? "flag_off" : err instanceof Error ? err.message : "invoke_failed");
    }
  };

  const blocked = isForecastBlocked(confidence);
  const line = brainForecastLine(confidence, null);
  const nodes = (graph?.nodes ?? []).filter((node) =>
    sku ? `${node.id} ${node.name ?? ""}`.toLowerCase().includes(sku.toLowerCase()) : true,
  );

  return (
    <div className="flex flex-col gap-6" data-testid="gs-u-digital-brain">
      <section className="desk-card p-5">
        <div className="flex items-center justify-between gap-3 mb-3">
          <h2 className="md-typescale-title-medium">Belief health</h2>
          <SourceChip source={blocked ? "unavailable" : "live"} />
        </div>
        {confidence ? (
          <ForecastConfidenceCard
            confidence={confidence}
            updatedAt={formatForecastUpdatedAt(generatedAt)}
            stale={isForecastStale(generatedAt)}
          />
        ) : (
          <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            Loading belief…
          </p>
        )}
        {line ? (
          <p className="md-typescale-body-small mt-2" data-testid="gs-u-brain-forecast-line">
            Accuracy series {line.points.length} points
          </p>
        ) : (
          <p className="md-typescale-body-small mt-2" data-testid="gs-u-brain-no-forecast-line">
            No forecast line
          </p>
        )}
      </section>

      <section className="desk-card p-5">
        <h2 className="md-typescale-title-medium mb-3">Sparsity</h2>
        <div className="flex flex-wrap gap-2 mb-3">
          <input
            className="portal-input"
            value={retailer}
            onChange={(e) => setRetailer(e.target.value)}
            placeholder="Retailer id"
            aria-label="Retailer id"
          />
          <button type="button" className="portal-btn portal-btn--outline" onClick={() => void checkSparsity()}>
            Check
          </button>
        </div>
        {sparsityError ? (
          <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            {sparsityError}
          </p>
        ) : null}
        {sparsity ? (
          <p className="md-typescale-body-small" data-testid="gs-u-sparsity-result">
            {sparsity.allowed ? "allowed" : "blocked"} · {sparsity.label} · {sparsity.completed_orders} orders
            {sparsity.blocked_reason ? (
              <span data-testid="gs-u-sparsity-blocked-reason"> · {sparsity.blocked_reason}</span>
            ) : null}
          </p>
        ) : null}
      </section>

      <section className="desk-card p-5">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-3">
          <h2 className="md-typescale-title-medium">Twin routes</h2>
          <div>
            <button
              type="button"
              className="portal-btn portal-btn--ghost text-sm"
              aria-pressed={twinPlane === "last_mile"}
              onClick={() => setTwinPlane("last_mile")}
            >
              Last-mile
            </button>
            <button
              type="button"
              className="portal-btn portal-btn--ghost text-sm"
              aria-pressed={twinPlane === "factory"}
              onClick={() => setTwinPlane("factory")}
            >
              Factory
            </button>
          </div>
        </div>
        {twinPlane === "factory" || twinError === "unavailable" ? (
          <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            <SourceChip source="unavailable" /> Factory-plane twin is a separate fetch. Not merged with last-mile.
          </p>
        ) : twinError ? (
          <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            {twinError}
          </p>
        ) : routes && routes.length === 0 ? (
          <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            No active last-mile twins
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {(routes ?? []).slice(0, 8).map((row) => (
              <li key={row.RouteID} className="md-typescale-body-small">
                {row.RouteID} · {row.Status} · {row.lateness}
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="desk-card p-5">
        <div className="flex items-center justify-between gap-3 mb-3">
          <h2 className="md-typescale-title-medium">Knowledge graph</h2>
          <Link href="/analytics/knowledge-graph" className="portal-btn portal-btn--ghost text-sm">
            Open graph
          </Link>
        </div>
        {graph == null ? (
          <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            Graph unavailable
          </p>
        ) : nodes.length === 0 ? (
          <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            No nodes{sku ? ` matching ${sku}` : ""}
          </p>
        ) : (
          <ul className="flex flex-col gap-1">
            {nodes.slice(0, 12).map((node) => (
              <li key={node.id} className="md-typescale-body-small">
                {node.type} · {node.name || node.id}
              </li>
            ))}
          </ul>
        )}
      </section>

      <SignalIngestOpsPanel />
      <ForecastAccuracyPanel />

      <section className="desk-card p-5">
        <h2 className="md-typescale-title-medium mb-2">Governed agent</h2>
        <p className="md-typescale-body-small mb-3" style={{ color: "var(--desk-text-secondary)" }}>
          Human-in-the-loop only. Preview / invoke, never an auto-place.
        </p>
        <div className="flex flex-wrap gap-2">
          <button type="button" className="portal-btn portal-btn--outline" onClick={() => void invokeAgent("approve_insight")}>
            Approve insight
          </button>
          <button type="button" className="portal-btn portal-btn--outline" onClick={() => void invokeAgent("open_supply_request")}>
            Open supply request
          </button>
          <button type="button" className="portal-btn portal-btn--outline" onClick={() => void invokeAgent("broadcast_template")}>
            Broadcast template
          </button>
        </div>
        {agentStatus ? (
          <p className="md-typescale-body-small mt-2" data-testid="gs-u-agent-status">
            {agentStatus}
          </p>
        ) : null}
      </section>
    </div>
  );
}
