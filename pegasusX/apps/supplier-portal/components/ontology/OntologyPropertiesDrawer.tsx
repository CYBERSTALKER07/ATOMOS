"use client";

import { useState } from "react";
import {
  extractObjectPrimaryLabel,
  formatOntologyPropertyValue,
  isMinorUnitPropertyKey,
  type OntologyObject,
  type OntologyTypeDescriptor,
} from "@/lib/ontology-api";
import { usePortalT } from "@/lib/i18n";
import { OntologyMarkingList } from "@/components/ontology/OntologyMarkingsBadge";

export function OntologyPropertiesDrawer({
  object,
  descriptor,
  piiMasked,
  financeMasked,
  isFocusedRoot,
  onTraverse,
}: {
  object: OntologyObject | null;
  descriptor?: OntologyTypeDescriptor;
  piiMasked?: boolean;
  financeMasked?: boolean;
  isFocusedRoot?: boolean;
  onTraverse?: (obj: OntologyObject) => void;
}) {
  const t = usePortalT();
  const [showRawJson, setShowRawJson] = useState(false);

  if (!object) {
    return (
      <div
        className="desk-card p-5"
        style={{ background: "var(--desk-surface)", borderColor: "var(--desk-border)" }}
      >
        <h2 className="bento-card-title">
          {t("supplier_portal.analytics.knowledge_graph.properties_drawer_title")}
        </h2>
        <p className="mt-2 text-sm" style={{ color: "var(--desk-text-secondary)" }}>
          {t("supplier_portal.analytics.knowledge_graph.properties_empty")}
        </p>
      </div>
    );
  }

  const props = object.properties ?? {};
  const currency =
    typeof props.currency === "string" && props.currency.trim() ? props.currency.trim() : "UZS";
  const entries = Object.entries(props).sort(([a], [b]) => a.localeCompare(b));

  return (
    <aside
      aria-label={t("supplier_portal.analytics.knowledge_graph.properties_drawer_title")}
      className="desk-card flex flex-col gap-4 p-5"
      style={{ background: "var(--desk-surface)", borderColor: "var(--desk-border)" }}
    >
      <div className="flex flex-wrap items-start justify-between gap-2 border-b pb-3" style={{ borderColor: "var(--desk-border)" }}>
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span
              className="rounded border px-2 py-0.5 font-mono text-xs font-semibold"
              style={{
                borderColor: "var(--desk-border)",
                background: "var(--desk-surface-raised)",
                color: "var(--desk-text-primary)",
              }}
            >
              {object.object_type}
            </span>
            <span className="font-mono text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
              v{object.version}
            </span>
          </div>
          <h2 className="mt-1.5 text-base font-semibold break-words" style={{ color: "var(--desk-text-primary)" }}>
            {extractObjectPrimaryLabel(object)}
          </h2>
          <p className="font-mono text-xs break-all" style={{ color: "var(--desk-text-secondary)" }}>
            {object.object_id}
          </p>
        </div>

        {onTraverse && !isFocusedRoot ? (
          <button
            type="button"
            className="md-btn md-btn-tonal text-xs px-3 py-1.5"
            onClick={() => onTraverse(object)}
          >
            {t("supplier_portal.analytics.knowledge_graph.traverse_from_node")}
          </button>
        ) : null}
      </div>

      {(piiMasked || financeMasked) ? (
        <div
          className="rounded-lg border px-3 py-2 text-xs"
          style={{
            borderColor: "color-mix(in srgb, var(--desk-warning, #d97706) 40%, transparent)",
            background: "var(--desk-warning-soft, rgba(217, 119, 6, 0.08))",
            color: "var(--desk-text-primary)",
          }}
        >
          {piiMasked ? (
            <p>{t("supplier_portal.analytics.knowledge_graph.redaction_notice_pii")}</p>
          ) : null}
          {financeMasked ? (
            <p>{t("supplier_portal.analytics.knowledge_graph.redaction_notice_finance")}</p>
          ) : null}
        </div>
      ) : null}

      <div className="space-y-1.5">
        <div className="text-xs font-medium" style={{ color: "var(--desk-text-secondary)" }}>
          {t("supplier_portal.analytics.knowledge_graph.markings_label")}
        </div>
        <OntologyMarkingList
          markings={object.security_markings}
          emptyLabel={t("supplier_portal.analytics.knowledge_graph.no_markings")}
        />
      </div>

      <dl className="grid grid-cols-2 gap-2 rounded-lg border p-3 text-xs" style={{ borderColor: "var(--desk-border)", background: "var(--desk-canvas)" }}>
        <div>
          <dt style={{ color: "var(--desk-text-tertiary)" }}>
            {t("supplier_portal.analytics.knowledge_graph.cell_filter_label")}
          </dt>
          <dd className="font-mono mt-0.5" style={{ color: "var(--desk-text-primary)" }}>
            {object.cell_id}
          </dd>
        </div>
        <div>
          <dt style={{ color: "var(--desk-text-tertiary)" }}>
            {t("supplier_portal.analytics.knowledge_graph.version_label")}
          </dt>
          <dd className="font-mono mt-0.5" style={{ color: "var(--desk-text-primary)" }}>
            v{object.version}
          </dd>
        </div>
        <div className="col-span-2">
          <dt style={{ color: "var(--desk-text-tertiary)" }}>
            {t("supplier_portal.analytics.knowledge_graph.updated_at_label")}
          </dt>
          <dd className="font-mono mt-0.5 break-all" style={{ color: "var(--desk-text-secondary)" }}>
            {object.updated_at}
          </dd>
        </div>
      </dl>

      {descriptor ? (
        <div className="rounded-lg border p-3 text-xs space-y-1" style={{ borderColor: "var(--desk-border)" }}>
          <div className="font-semibold" style={{ color: "var(--desk-text-primary)" }}>
            Schema: <span className="font-mono">{descriptor.schema_file}</span> (v{descriptor.version})
          </div>
          <p style={{ color: "var(--desk-text-secondary)" }}>{descriptor.description}</p>
          {descriptor.allowed_outgoing_links && descriptor.allowed_outgoing_links.length > 0 ? (
            <div className="pt-1 font-mono" style={{ color: "var(--desk-text-tertiary)" }}>
              Links: {descriptor.allowed_outgoing_links.join(", ")}
            </div>
          ) : null}
        </div>
      ) : null}

      <div>
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color: "var(--desk-text-secondary)" }}>
            {t("supplier_portal.analytics.knowledge_graph.properties_drawer_title")} ({entries.length})
          </h3>
          <button
            type="button"
            className="text-xs underline"
            style={{ color: "var(--desk-text-secondary)" }}
            onClick={() => setShowRawJson((prev) => !prev)}
          >
            {showRawJson ? "Table" : "JSON"}
          </button>
        </div>

        {showRawJson ? (
          <pre
            className="max-h-80 overflow-auto rounded-lg border p-3 font-mono text-xs"
            style={{
              borderColor: "var(--desk-border)",
              background: "var(--desk-canvas)",
              color: "var(--desk-text-primary)",
            }}
          >
            {JSON.stringify(object, null, 2)}
          </pre>
        ) : entries.length === 0 ? (
          <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
            —
          </p>
        ) : (
          <div className="overflow-x-auto rounded-lg border" style={{ borderColor: "var(--desk-border)" }}>
            <table className="w-full text-xs">
              <tbody>
                {entries.map(([key, val]) => {
                  const formatted = formatOntologyPropertyValue(key, val, currency);
                  const isMoney = isMinorUnitPropertyKey(key);
                  return (
                    <tr key={key} style={{ borderTop: "1px solid var(--desk-border)" }}>
                      <th
                        className="px-2.5 py-2 text-left font-mono font-medium align-top"
                        style={{ color: "var(--desk-text-secondary)" }}
                      >
                        {key}
                      </th>
                      <td
                        className={`px-2.5 py-2 text-right font-mono break-all ${isMoney ? "font-semibold" : ""}`}
                        style={{ color: "var(--desk-text-primary)" }}
                      >
                        {formatted}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </aside>
  );
}
