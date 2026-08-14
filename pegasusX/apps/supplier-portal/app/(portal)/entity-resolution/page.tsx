"use client";

import { useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { ApiError } from "@pegasusx/api-client";
import type { EntityResolutionExplainResponse, EntityResolutionResolveResponse } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

const TYPES = ["ANY", "ORDER", "RETAILER", "WAREHOUSE", "FACTORY", "DRIVER", "VEHICLE", "SUPPLIER"];

export default function EntityResolutionPage() {
  const [entityType, setEntityType] = useState("ANY");
  const [query, setQuery] = useState("");
  const [entityId, setEntityId] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resolved, setResolved] = useState<EntityResolutionResolveResponse | null>(null);
  const [explain, setExplain] = useState<EntityResolutionExplainResponse | null>(null);

  async function runResolve() {
    setBusy(true);
    setError(null);
    setExplain(null);
    try {
      const resp = await api.resolveSupplierEntity({
        entity_type: entityType,
        query: query.trim() || undefined,
        entity_id: entityId.trim() || undefined,
      });
      setResolved(resp);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : err instanceof Error ? err.message : "Resolve failed");
      setResolved(null);
    } finally {
      setBusy(false);
    }
  }

  async function runExplain(type: string, id: string) {
    setBusy(true);
    setError(null);
    try {
      const resp = await api.explainSupplierEntity({ entity_type: type, entity_id: id });
      setExplain(resp);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Explain failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <PageChrome
      icon="topology"
      title="Entity resolution"
      description="Typed resolve/explain against the supplier graph. Not a search box over raw Spanner."
    >
      <div className="desk-card p-6 space-y-3 max-w-2xl">
        <label className="block space-y-1">
          <span className="md-typescale-label-medium">Type</span>
          <select className="md-input w-full" value={entityType} onChange={(e) => setEntityType(e.target.value)}>
            {TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
        </label>
        <label className="block space-y-1">
          <span className="md-typescale-label-medium">Query</span>
          <input className="md-input w-full" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Label or phone" />
        </label>
        <label className="block space-y-1">
          <span className="md-typescale-label-medium">Entity ID</span>
          <input className="md-input w-full" value={entityId} onChange={(e) => setEntityId(e.target.value)} placeholder="Optional exact id" />
        </label>
        <button type="button" className="md-btn md-btn-filled px-4 py-2" disabled={busy} onClick={() => void runResolve()}>
          {busy ? "Resolving…" : "Resolve"}
        </button>
        {error ? <p className="text-sm text-red-600">{error}</p> : null}
        {resolved ? (
          <div className="space-y-2 text-sm">
            <p>Requested {resolved.requested_type} · {resolved.candidates.length} candidates</p>
            {resolved.resolved ? (
              <p>
                Top: {resolved.resolved.label} ({resolved.resolved.entity_id}) score {resolved.resolved.score}
              </p>
            ) : <p>No deterministic match</p>}
            <ul className="list-disc pl-5">
              {resolved.candidates.map((c) => (
                <li key={c.node_id}>
                  {c.label} · {c.entity_type}/{c.entity_id} · {c.confidence_class}
                  <button
                    type="button"
                    className="ml-2 underline"
                    onClick={() => void runExplain(c.entity_type, c.entity_id)}
                  >
                    Explain
                  </button>
                </li>
              ))}
            </ul>
          </div>
        ) : null}
        {explain ? (
          <div className="text-sm">
            <p className="font-medium">Lineage for {explain.source.label}</p>
            <ul className="list-disc pl-5">
              {explain.projection.edges.map((e) => (
                <li key={`${e.from}-${e.to}-${e.relation}`}>{e.relation}: {e.from} → {e.to}</li>
              ))}
            </ul>
          </div>
        ) : null}
      </div>
    </PageChrome>
  );
}
