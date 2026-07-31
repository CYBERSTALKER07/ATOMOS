"use client";

import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import type { ScoredException } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";

const api = createSupplierApi();

export function ScoredExceptionsPanel({ limit = 10 }: { limit?: number }) {
  const [rows, setRows] = useState<ScoredException[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api
      .listScoredExceptions(limit)
      .then((resp) => setRows(resp.exceptions ?? []))
      .catch((err) => setError(err instanceof Error ? err.message : "load_scored_exceptions_failed"))
      .finally(() => setLoading(false));
  }, [limit]);

  useEffect(() => {
    load();
  }, [load]);

  if (loading) {
    return <p className="text-sm text-gray-400">Loading scored exceptions…</p>;
  }

  if (error) {
    return (
      <p className="text-sm text-red-400">
        Scored exceptions unavailable ({error}). Enable CONTROL_TOWER_PLAYBOOKS_ENABLED.
      </p>
    );
  }

  if (rows.length === 0) {
    return <p className="text-sm text-gray-400">No open scored exceptions.</p>;
  }

  return (
    <ul className="divide-y divide-white/10 rounded-lg border border-white/10 bg-white/5">
      {rows.map((row) => (
        <li key={row.exception_id} className="flex flex-wrap items-center gap-2 p-3 text-sm">
          <span className="font-mono text-xs text-emerald-400/90">{row.type}</span>
          <span className="rounded bg-white/10 px-2 py-0.5 text-xs">score {row.score}</span>
          <span className="text-xs text-gray-400">{row.age_minutes}m</span>
          {row.order_id ? (
            <Link href={`/orders/${row.order_id}`} className="text-xs text-emerald-400 underline">
              {row.order_id}
            </Link>
          ) : null}
          {row.top_playbook_name ? (
            <span className="text-xs text-gray-300">{row.top_playbook_name}</span>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
