"use client";

import Link from "next/link";
import type { Route } from "next";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierKnowledgeGraph } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function KnowledgeGraphPage() {
  const [graph, setGraph] = useState<SupplierKnowledgeGraph | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .getSupplierKnowledgeGraph()
      .then((resp) => {
        if (!cancelled) setGraph(resp);
      })
      .catch(() => {
        if (!cancelled) setError("load_knowledge_graph_failed");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <PageChrome
      icon="topology"
      title="Enterprise knowledge graph"
      description="Read-only supplier planning graph — factories, warehouses, SKUs, and relationships."
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <Link href={"/analytics" as Route} className="md-btn md-btn-text">
          Back to analytics
        </Link>
      }
    >
      {graph ? (
        <div className="flex flex-col gap-6">
          <section className="desk-card p-6">
            <h2 className="bento-card-title">Nodes ({graph.nodes.length})</h2>
            <div className="mt-4 overflow-x-auto">
              <table className="desk-table w-full">
                <thead>
                  <tr style={{ color: "var(--desk-text-secondary)" }}>
                    <th className="md-typescale-label-medium p-3 text-left font-medium">ID</th>
                    <th className="md-typescale-label-medium p-3 text-left font-medium">Type</th>
                    <th className="md-typescale-label-medium p-3 text-left font-medium">Name</th>
                  </tr>
                </thead>
                <tbody>
                  {graph.nodes.map((node) => (
                    <tr key={node.id} style={{ borderTop: "1px solid var(--desk-border)" }}>
                      <td className="p-3 md-typescale-body-medium font-mono text-sm">{node.id}</td>
                      <td className="p-3 md-typescale-body-medium">{node.type}</td>
                      <td className="p-3 md-typescale-body-medium">{node.name || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <section className="desk-card p-6">
            <h2 className="bento-card-title">Edges ({graph.edges.length})</h2>
            <div className="mt-4 overflow-x-auto">
              <table className="desk-table w-full">
                <thead>
                  <tr style={{ color: "var(--desk-text-secondary)" }}>
                    <th className="md-typescale-label-medium p-3 text-left font-medium">From</th>
                    <th className="md-typescale-label-medium p-3 text-left font-medium">Relation</th>
                    <th className="md-typescale-label-medium p-3 text-left font-medium">To</th>
                  </tr>
                </thead>
                <tbody>
                  {graph.edges.map((edge, index) => (
                    <tr key={`${edge.from}-${edge.to}-${index}`} style={{ borderTop: "1px solid var(--desk-border)" }}>
                      <td className="p-3 md-typescale-body-medium font-mono text-sm">{edge.from}</td>
                      <td className="p-3 md-typescale-body-medium">{edge.relation}</td>
                      <td className="p-3 md-typescale-body-medium font-mono text-sm">{edge.to}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </div>
      ) : null}
    </PageChrome>
  );
}
