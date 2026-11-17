"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiError } from "@pegasusx/api-core";
import Icon from "@/components/Icon";
import {
  createSupplierAuditIamApi,
  type AuditActorType,
  type AuditEvent,
  type ChainVerifyResult,
  type SupplierAuditIamApi,
} from "@/lib/audit-iam-api";
import { usePortalT } from "@/lib/i18n";

export interface ObjectHistoryPanelProps {
  objectType: string;
  objectId: string;
  fallbackObjectTypes?: string[];
  entityId?: string;
  limit?: number;
  title?: string;
  compact?: boolean;
  autoVerifyChain?: boolean;
  apiClient?: SupplierAuditIamApi;
  onClose?: () => void;
}

function formatWhen(iso: string): string {
  try {
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(iso));
  } catch {
    return iso;
  }
}

function formatJsonPreview(raw?: string): string {
  if (!raw) return "";
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

function shortHash(hash?: string): string {
  const clean = (hash || "").trim();
  if (!clean) return "—";
  if (clean.length <= 14) return clean;
  return `${clean.slice(0, 8)}…${clean.slice(-4)}`;
}

export function ActorTypeBadge({ actorType }: { actorType: AuditActorType | string }) {
  const t = usePortalT();
  const normalized = (actorType || "system").toLowerCase();

  let label = t("supplier_portal.audit.actor_system");
  let style: React.CSSProperties = {
    borderColor: "var(--desk-border)",
    background: "var(--desk-surface-subtle)",
    color: "var(--desk-text-secondary)",
  };

  if (normalized === "human") {
    label = t("supplier_portal.audit.actor_human");
    style = {
      borderColor: "var(--desk-border-strong)",
      background: "var(--desk-surface)",
      color: "var(--desk-text-primary)",
    };
  } else if (normalized === "agent") {
    label = t("supplier_portal.audit.actor_agent");
    style = {
      borderColor: "var(--desk-accent)",
      background: "var(--desk-accent)",
      color: "var(--desk-accent-on)",
    };
  } else if (normalized === "connector") {
    label = t("supplier_portal.audit.actor_connector");
    style = {
      borderColor: "var(--desk-border-strong)",
      background: "var(--desk-surface-sunken)",
      color: "var(--desk-text-primary)",
    };
  }

  return (
    <span
      data-actor-type={normalized}
      className="inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-mono font-semibold uppercase tracking-wide"
      style={style}
    >
      {label}
    </span>
  );
}

export function ObjectHistoryPanel({
  objectType,
  objectId,
  fallbackObjectTypes,
  entityId,
  limit = 100,
  title,
  compact = false,
  autoVerifyChain = false,
  apiClient,
  onClose,
}: ObjectHistoryPanelProps) {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;

  const auditApi = useMemo(
    () => apiClient ?? createSupplierAuditIamApi(),
    [apiClient],
  );

  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [piiMasked, setPiiMasked] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedDiffs, setExpandedDiffs] = useState<Record<string, boolean>>({});
  const [verifying, setVerifying] = useState<boolean>(false);
  const [verifyResult, setVerifyResult] = useState<ChainVerifyResult | null>(null);
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const fallbackTypesKey = (fallbackObjectTypes ?? []).join("|");

  const loadHistory = useCallback(async () => {
    if (!objectType.trim() || !objectId.trim()) {
      setEvents([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    setVerifyResult(null);
    setVerifyError(null);
    try {
      let res = await auditApi.getObjectHistory(objectType, objectId, {
        entity: entityId,
        limit,
      });
      let loaded = res.events ?? [];
      if (loaded.length === 0 && fallbackTypesKey) {
        for (const altType of fallbackTypesKey.split("|")) {
          if (!altType.trim() || altType.toLowerCase() === objectType.toLowerCase()) continue;
          const altRes = await auditApi.getObjectHistory(altType, objectId, {
            entity: entityId,
            limit,
          });
          if ((altRes.events ?? []).length > 0) {
            res = altRes;
            loaded = altRes.events ?? [];
            break;
          }
        }
      }
      setEvents(loaded);
      setPiiMasked(Boolean(res.pii_masked));

      if (autoVerifyChain) {
        const firstHashChained = loaded.find(
          (ev) => ev.source !== "legacy_audit_log" && Boolean(ev.day_bucket),
        );
        if (firstHashChained) {
          try {
            const chain = await auditApi.verifyDayChain({
              date: firstHashChained.day_bucket,
              entity: entityId || firstHashChained.tenant_id,
            });
            setVerifyResult(chain);
          } catch {
            // Keep history usable if anchor/verify check cannot run.
          }
        }
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setError(tRef.current("supplier_portal.audit.permission_denied"));
      } else {
        setError(
          err instanceof Error
            ? err.message
            : tRef.current("supplier_portal.audit.load_failed"),
        );
      }
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [auditApi, autoVerifyChain, entityId, fallbackTypesKey, limit, objectId, objectType]);

  useEffect(() => {
    void loadHistory();
  }, [loadHistory]);

  const latestChainBucket = useMemo(() => {
    const ev = events.find(
      (item) => item.source !== "legacy_audit_log" && Boolean(item.day_bucket),
    );
    return ev ? { dayBucket: ev.day_bucket, tenantId: ev.tenant_id } : null;
  }, [events]);

  const handleVerifyDayChain = async () => {
    if (!latestChainBucket) return;
    setVerifying(true);
    setVerifyError(null);
    try {
      const res = await auditApi.verifyDayChain({
        date: latestChainBucket.dayBucket,
        entity: entityId || latestChainBucket.tenantId,
      });
      setVerifyResult(res);
    } catch (err) {
      setVerifyError(
        err instanceof Error
          ? err.message
          : t("supplier_portal.audit.chain_broken"),
      );
    } finally {
      setVerifying(false);
    }
  };

  const toggleDiff = (eventId: string) => {
    setExpandedDiffs((prev) => ({ ...prev, [eventId]: !prev[eventId] }));
  };

  return (
    <section
      data-testid="object-history-panel"
      className={`rounded-xl border ${compact ? "p-3" : "p-4"} space-y-3`}
      style={{
        borderColor: "var(--desk-border)",
        background: "var(--desk-surface)",
      }}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3
              className="text-sm font-semibold"
              style={{ color: "var(--desk-text-primary)" }}
            >
              {title || t("supplier_portal.audit.panel_title")}
            </h3>
            <span
              className="rounded-full border px-2 py-0.5 text-[11px] font-mono"
              style={{
                borderColor: "var(--desk-border)",
                color: "var(--desk-text-secondary)",
              }}
            >
              {objectType} · {objectId}
            </span>
          </div>
          <p
            className="mt-1 text-xs"
            style={{ color: "var(--desk-text-tertiary)" }}
          >
            {piiMasked
              ? t("supplier_portal.audit.pii_masked_note")
              : t("supplier_portal.audit.pii_unmasked_note")}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {verifyResult ? (
            <span
              data-testid="chain-verify-badge"
              className="inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-[11px] font-mono font-semibold"
              style={{
                borderColor: verifyResult.valid
                  ? "var(--desk-border-strong)"
                  : "var(--desk-danger)",
                background: verifyResult.valid
                  ? "var(--desk-success-soft)"
                  : "var(--desk-danger-soft)",
                color: verifyResult.valid
                  ? "var(--desk-text-primary)"
                  : "var(--desk-danger)",
              }}
              title={verifyResult.root_hash}
            >
              <Icon
                name={verifyResult.valid ? "verified" : "warning"}
                size={13}
              />
              <span>
                {verifyResult.valid
                  ? t("supplier_portal.audit.chain_valid")
                  : t("supplier_portal.audit.chain_broken")}
              </span>
              <span className="opacity-75">
                ({verifyResult.day_bucket} · {shortHash(verifyResult.root_hash)})
              </span>
            </span>
          ) : null}

          {latestChainBucket ? (
            <button
              type="button"
              disabled={verifying}
              onClick={() => void handleVerifyDayChain()}
              className="rounded-lg border px-2.5 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
              style={{
                borderColor: "var(--desk-border)",
                color: "var(--desk-text-primary)",
              }}
            >
              {verifying
                ? t("supplier_portal.audit.verifying_chain")
                : t("supplier_portal.audit.verify_day_chain")}
            </button>
          ) : null}

          <button
            type="button"
            disabled={loading}
            onClick={() => void loadHistory()}
            className="rounded-lg border px-2.5 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
            style={{
              borderColor: "var(--desk-border)",
              color: "var(--desk-text-secondary)",
            }}
            aria-label={t("common.action.refresh")}
          >
            <Icon name="refresh" size={13} className={loading ? "animate-spin" : ""} />
          </button>

          {onClose ? (
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border px-2.5 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)]"
              style={{
                borderColor: "var(--desk-border)",
                color: "var(--desk-text-secondary)",
              }}
            >
              {t("supplier_portal.audit.close_history")}
            </button>
          ) : null}
        </div>
      </div>

      {verifyError ? (
        <p className="text-xs" role="alert" style={{ color: "var(--desk-danger)" }}>
          {verifyError}
        </p>
      ) : null}

      {error ? (
        <div
          className="rounded-lg border p-3 text-xs"
          role="alert"
          style={{
            borderColor: "var(--desk-border-strong)",
            background: "var(--desk-surface-subtle)",
            color: "var(--desk-text-primary)",
          }}
        >
          {error}
        </div>
      ) : loading ? (
        <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
          {t("supplier_portal.audit.loading")}
        </p>
      ) : events.length === 0 ? (
        <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
          {t("supplier_portal.audit.empty")}
        </p>
      ) : (
        <ol
          className="space-y-3 border-l pl-4"
          style={{ borderColor: "var(--desk-border)" }}
        >
          {events.map((ev) => {
            const hasDiff = Boolean(ev.before_json || ev.after_json || ev.diff_gcs_uri);
            const isDiffOpen = Boolean(expandedDiffs[ev.event_id]);
            const isLegacy = ev.source === "legacy_audit_log";

            return (
              <li key={ev.event_id} className="relative space-y-1.5">
                <span
                  className="absolute -left-[1.3rem] top-1.5 h-2 w-2 rounded-full"
                  style={{ background: "var(--desk-accent)" }}
                />
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <span
                      className="text-xs font-semibold font-mono"
                      style={{ color: "var(--desk-text-primary)" }}
                    >
                      {ev.action}
                    </span>
                    <ActorTypeBadge actorType={ev.actor_type} />
                    {ev.legal_hold ? (
                      <span
                        className="rounded-full border px-2 py-0.5 text-[10px] font-mono font-bold uppercase"
                        style={{
                          borderColor: "var(--desk-border-strong)",
                          background: "var(--desk-surface-sunken)",
                          color: "var(--desk-text-primary)",
                        }}
                        title={ev.legal_hold_reason || undefined}
                      >
                        {t("supplier_portal.audit.legal_hold_badge")}
                      </span>
                    ) : null}
                    {isLegacy ? (
                      <span
                        className="rounded-full border px-2 py-0.5 text-[10px] font-mono"
                        style={{
                          borderColor: "var(--desk-border)",
                          color: "var(--desk-text-tertiary)",
                        }}
                      >
                        {t("supplier_portal.audit.legacy_source")}
                      </span>
                    ) : null}
                  </div>
                  <span
                    className="text-[11px] whitespace-nowrap"
                    style={{ color: "var(--desk-text-tertiary)" }}
                  >
                    {formatWhen(ev.created_at)}
                  </span>
                </div>

                <div
                  className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] font-mono"
                  style={{ color: "var(--desk-text-secondary)" }}
                >
                  <span>
                    {ev.actor_id}
                    {ev.role ? ` (${ev.role})` : ""}
                  </span>
                  {ev.trace_id ? (
                    <span>
                      {t("supplier_portal.audit.trace_id")}: {ev.trace_id}
                    </span>
                  ) : null}
                  {!isLegacy && ev.seq_num > 0 ? (
                    <span>
                      {t("supplier_portal.audit.seq_num", { seq: ev.seq_num })}
                    </span>
                  ) : null}
                  {!isLegacy && ev.entry_hash ? (
                    <span title={`${ev.prev_hash} -> ${ev.entry_hash}`}>
                      {shortHash(ev.prev_hash)} → {shortHash(ev.entry_hash)}
                    </span>
                  ) : null}
                  {ev.reason_code ? <span>reason:{ev.reason_code}</span> : null}
                  {ev.approval_id ? <span>approval:{ev.approval_id}</span> : null}
                </div>

                {hasDiff ? (
                  <div>
                    <button
                      type="button"
                      onClick={() => toggleDiff(ev.event_id)}
                      className="text-[11px] font-medium underline"
                      style={{ color: "var(--desk-text-secondary)" }}
                    >
                      {isDiffOpen
                        ? t("supplier_portal.audit.hide_payload")
                        : t("supplier_portal.audit.show_payload")}
                    </button>

                    {isDiffOpen ? (
                      <div className="mt-2 grid grid-cols-1 gap-2 md:grid-cols-2">
                        {ev.before_json ? (
                          <div
                            className="rounded-lg border p-2"
                            style={{
                              borderColor: "var(--desk-border)",
                              background: "var(--desk-surface-subtle)",
                            }}
                          >
                            <div
                              className="mb-1 text-[10px] font-semibold uppercase"
                              style={{ color: "var(--desk-text-tertiary)" }}
                            >
                              {t("supplier_portal.audit.before_label")}
                            </div>
                            <pre className="max-h-48 overflow-auto font-mono text-[11px] whitespace-pre-wrap break-all">
                              {formatJsonPreview(ev.before_json)}
                            </pre>
                          </div>
                        ) : null}
                        {ev.after_json ? (
                          <div
                            className="rounded-lg border p-2"
                            style={{
                              borderColor: "var(--desk-border)",
                              background: "var(--desk-surface-subtle)",
                            }}
                          >
                            <div
                              className="mb-1 text-[10px] font-semibold uppercase"
                              style={{ color: "var(--desk-text-tertiary)" }}
                            >
                              {t("supplier_portal.audit.after_label")}
                            </div>
                            <pre className="max-h-48 overflow-auto font-mono text-[11px] whitespace-pre-wrap break-all">
                              {formatJsonPreview(ev.after_json)}
                            </pre>
                          </div>
                        ) : null}
                        {ev.diff_gcs_uri ? (
                          <div
                            className="col-span-full rounded-lg border px-2.5 py-1.5 text-[11px] font-mono"
                            style={{
                              borderColor: "var(--desk-border)",
                              color: "var(--desk-text-secondary)",
                            }}
                          >
                            {t("supplier_portal.audit.offloaded_diff")}:{" "}
                            {ev.diff_gcs_uri}
                          </div>
                        ) : null}
                      </div>
                    ) : null}
                  </div>
                ) : null}
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
}
