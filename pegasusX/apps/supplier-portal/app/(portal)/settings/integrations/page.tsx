"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import StatusChip from "@/components/StatusChip";
import { OIDCAttachCard } from "@/components/settings/OIDCAttachCard";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type {
  PartnerApiKeyMeta,
  PartnerDeadLetterAttempt,
  PartnerEdiDocument,
  PartnerExportJob,
  PartnerSftpConfig,
  PartnerAs2Config,
  PartnerCoaMap,
  PartnerWebhookSubscription,
} from "@pegasusx/types";

const api = createSupplierApi();

const EVENT_OPTIONS = ["ORDER_CREATED", "ORDER_STATUS_CHANGED", "CLAIM_FILED", "PAYMENT_CLEARED"];

export default function IntegrationsSettingsPage() {
  const t = usePortalT();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [keys, setKeys] = useState<PartnerApiKeyMeta[]>([]);
  const [issuedSecret, setIssuedSecret] = useState<string | null>(null);

  const [hooks, setHooks] = useState<PartnerWebhookSubscription[]>([]);
  const [hookUrl, setHookUrl] = useState("");
  const [hookEvents, setHookEvents] = useState<string[]>(["ORDER_CREATED"]);
  const [dead, setDead] = useState<PartnerDeadLetterAttempt[]>([]);

  const [jobs, setJobs] = useState<PartnerExportJob[]>([]);
  const [resource, setResource] = useState("orders");
  const [format, setFormat] = useState("csv");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const [sftp, setSftp] = useState<PartnerSftpConfig>({ configured: false });
  const [sftpHost, setSftpHost] = useState("");
  const [sftpPort, setSftpPort] = useState("22");
  const [sftpUser, setSftpUser] = useState("");
  const [sftpSecretRef, setSftpSecretRef] = useState("");
  const [sftpDir, setSftpDir] = useState("/");
  const [inboundDir, setInboundDir] = useState("inbound");
  const [outboundDir, setOutboundDir] = useState("outbound");
  const [archiveDir, setArchiveDir] = useState("archive");
  const [ediEnabled, setEdiEnabled] = useState(false);
  const [ediDocs, setEdiDocs] = useState<PartnerEdiDocument[]>([]);
  const [as2, setAs2] = useState<PartnerAs2Config>({ configured: false });
  const [as2Enabled, setAs2Enabled] = useState(false);
  const [ourAs2Id, setOurAs2Id] = useState("");
  const [partnerAs2Id, setPartnerAs2Id] = useState("");
  const [partnerAs2Url, setPartnerAs2Url] = useState("");
  const [ourCertRef, setOurCertRef] = useState("");
  const [ourKeyRef, setOurKeyRef] = useState("");
  const [partnerCertRef, setPartnerCertRef] = useState("");
  const [coaAr, setCoaAr] = useState("62.01");
  const [coaRevenue, setCoaRevenue] = useState("90.01");
  const [coaBank, setCoaBank] = useState("51.01");
  const [coaDefaults, setCoaDefaults] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [k, w, d, e, s, edi, coa, a2] = await Promise.all([
        api.listSupplierPartnerKeys(),
        api.listSupplierPartnerWebhooks(),
        api.listSupplierPartnerDeadLetter(),
        api.listSupplierPartnerExports(),
        api.getSupplierPartnerSftp(),
        api.listSupplierPartnerEdiDocuments().catch(() => ({ documents: [] as PartnerEdiDocument[] })),
        api.getSupplierPartnerCoa().catch(
          (): PartnerCoaMap => ({
            account_ar: "62.01",
            account_revenue: "90.01",
            account_bank_cash: "51.01",
            using_defaults: true,
          }),
        ),
        api.getSupplierPartnerAs2().catch((): PartnerAs2Config => ({ configured: false })),
      ]);
      setKeys(k.keys ?? []);
      setHooks(w.subscriptions ?? []);
      setDead(d.attempts ?? []);
      setJobs(e.jobs ?? []);
      setEdiDocs(edi.documents ?? []);
      setSftp(s);
      setAs2(a2);
      setCoaAr(coa.account_ar || "62.01");
      setCoaRevenue(coa.account_revenue || "90.01");
      setCoaBank(coa.account_bank_cash || "51.01");
      setCoaDefaults(Boolean(coa.using_defaults));
      if (s.configured) {
        setSftpHost(s.host ?? "");
        setSftpPort(String(s.port ?? 22));
        setSftpUser(s.username ?? "");
        setSftpSecretRef(s.secret_ref ?? "");
        setSftpDir(s.remote_dir ?? "/");
        setInboundDir(s.inbound_dir ?? "inbound");
        setOutboundDir(s.outbound_dir ?? "outbound");
        setArchiveDir(s.archive_dir ?? "archive");
        setEdiEnabled(Boolean(s.edi_enabled));
      }
      if (a2.configured) {
        setAs2Enabled(Boolean(a2.as2_enabled));
        setOurAs2Id(a2.our_as2_id ?? "");
        setPartnerAs2Id(a2.partner_as2_id ?? "");
        setPartnerAs2Url(a2.partner_url ?? "");
        setOurCertRef(a2.our_cert_secret_ref ?? "");
        setOurKeyRef(a2.our_key_secret_ref ?? "");
        setPartnerCertRef(a2.partner_cert_secret_ref ?? "");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.failed_to_load_integrations"));
    } finally {
      setLoading(false);
    }
  }, []);

  useSupplierSessionReconcile(load);
  useEffect(() => {
    void load();
  }, [load]);

  async function issueKey() {
    setNotice(null);
    try {
      const issued = await api.issueSupplierPartnerKey({ scopes: ["*"] });
      setIssuedSecret(issued.secret);
      setNotice("API key issued — copy the secret now; it is shown once.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.issue_key_failed"));
    }
  }

  async function revokeKey(keyId: string) {
    setNotice(null);
    try {
      await api.revokeSupplierPartnerKey(keyId);
      setNotice("Key revoked.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.revoke_failed"));
    }
  }

  async function createWebhook() {
    setNotice(null);
    try {
      const created = await api.createSupplierPartnerWebhook({
        url: hookUrl.trim(),
        event_types: hookEvents,
      });
      setNotice(`Webhook created. Signing secret (once): ${created.signing_secret}`);
      setHookUrl("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.create_webhook_failed"));
    }
  }

  async function pingWebhook(id: string) {
    try {
      await api.pingSupplierPartnerWebhook(id);
      setNotice("Ping sent.");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.ping_failed"));
    }
  }

  async function deactivateWebhook(id: string) {
    try {
      await api.deactivateSupplierPartnerWebhook(id);
      setNotice("Webhook deactivated.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.deactivate_failed"));
    }
  }

  async function replayDead(id: string) {
    try {
      await api.replaySupplierPartnerDeadLetter(id);
      setNotice("Dead-letter requeued.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.replay_failed"));
    }
  }

  async function requestExport() {
    setNotice(null);
    try {
      const job = await api.createSupplierPartnerExport({
        resource,
        format,
        from: from || undefined,
        to: to || undefined,
      });
      setNotice(`Export job ${job.job_id} queued (${job.status}).`);
      // Poll a few times for completion
      for (let i = 0; i < 8; i++) {
        await new Promise((r) => setTimeout(r, 1500));
        const latest = await api.getSupplierPartnerExport(job.job_id);
        if (latest.status === "SUCCEEDED" || latest.status === "FAILED") {
          setNotice(
            latest.status === "SUCCEEDED"
              ? `Export ready${latest.download_url ? `: ${latest.download_url}` : ""}`
              : `Export failed: ${latest.error || "unknown"}`,
          );
          break;
        }
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.export_failed"));
    }
  }

  async function saveSftp() {
    setNotice(null);
    try {
      await api.putSupplierPartnerSftp({
        host: sftpHost.trim(),
        port: Number(sftpPort) || 22,
        username: sftpUser.trim(),
        secret_ref: sftpSecretRef.trim(),
        remote_dir: sftpDir.trim() || "/",
        inbound_dir: inboundDir.trim() || "inbound",
        outbound_dir: outboundDir.trim() || "outbound",
        archive_dir: archiveDir.trim() || "archive",
        edi_enabled: ediEnabled,
      });
      setNotice("SFTP / EDI config saved (password/key stays in GSM via secret_ref).");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.sftp_save_failed"));
    }
  }

  async function saveAs2() {
    setNotice(null);
    try {
      const saved = await api.putSupplierPartnerAs2({
        as2_enabled: as2Enabled,
        our_as2_id: ourAs2Id.trim(),
        partner_as2_id: partnerAs2Id.trim(),
        partner_url: partnerAs2Url.trim(),
        our_cert_secret_ref: ourCertRef.trim(),
        our_key_secret_ref: ourKeyRef.trim(),
        partner_cert_secret_ref: partnerCertRef.trim(),
      });
      setAs2(saved);
      setNotice("AS2 config saved (PEM material stays in GSM via secret refs). Not Drummond-certified.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "AS2 save failed");
    }
  }

  async function saveCoa() {
    setNotice(null);
    try {
      const m = await api.putSupplierPartnerCoa({
        account_ar: coaAr.trim(),
        account_revenue: coaRevenue.trim(),
        account_bank_cash: coaBank.trim(),
      });
      setCoaAr(m.account_ar);
      setCoaRevenue(m.account_revenue);
      setCoaBank(m.account_bank_cash);
      setCoaDefaults(Boolean(m.using_defaults));
      setNotice("Chart of accounts saved for journals exports.");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.coa_save_failed"));
    }
  }

  async function replayEdi(id: string) {
    try {
      await api.replaySupplierPartnerEdiDocument(id);
      setNotice("EDI document requeued.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.edi_replay_failed"));
    }
  }

  return (
    <PageChrome
      title={t("portal.nav.integrations")}
      description={t("supplier_portal.residual.text.partner_api_keys_outbound_webhooks_bulk_exports_and_optional_sft")}
    >
      <div className="mx-auto max-w-3xl">
        <OIDCAttachCard />
      </div>
      {loading ? (
        <p className="text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.loading")}</p>
      ) : (
        <div className="mx-auto max-w-3xl space-y-10">
          {error && <p className="text-sm font-semibold text-red-600">{error}</p>}
          {notice && <p className="text-sm text-emerald-700">{notice}</p>}

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">{t("supplier_portal.settings.integrations.text.api_keys")}</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Long-lived <code className="text-xs">{t("supplier_portal.settings.integrations.text.pxk")}</code> keys authenticate{" "}
              <code className="text-xs">/partner/v1/*</code>. The same key is an OAuth2 client — exchange it
              at <code className="text-xs">{t("supplier_portal.settings.integrations.text.post_partner_v1_oauth_token")}</code> (
              <code className="text-xs">{t("supplier_portal.settings.integrations.text.grant_type_client_credentials")}</code>) for a short-lived Bearer JWT.
            </p>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Machine keys for <code className="text-xs">/partner/v1</code>. Secret is shown once at
              issue.
            </p>
            <button
              type="button"
              className="rounded bg-[var(--color-md-primary)] px-3 py-2 text-sm text-white"
              onClick={() => void issueKey()}
            >
              Issue key
            </button>
            {issuedSecret && (
              <pre className="overflow-x-auto rounded border p-3 text-xs">{issuedSecret}</pre>
            )}
            <ul className="divide-y border-t">
              {keys.map((k) => (
                <li key={k.key_id} className="flex flex-wrap items-center gap-3 py-3 text-sm">
                  <span className="font-mono text-xs">{k.key_prefix}…</span>
                  <span>{k.status}</span>
                  <span className="text-xs text-[var(--desk-text-secondary)]">
                    {(k.scopes ?? []).join(", ")}
                  </span>
                  {k.status === "ACTIVE" && (
                    <button
                      type="button"
                      className="text-xs underline"
                      onClick={() => void revokeKey(k.key_id)}
                    >
                      Revoke
                    </button>
                  )}
                </li>
              ))}
              {keys.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_keys_yet")}</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">{t("supplier_portal.settings.integrations.text.webhooks")}</h2>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input
                className="flex-1 rounded border px-3 py-2 text-sm"
                placeholder="https://example.com/hooks/pegasus"
                value={hookUrl}
                onChange={(e) => setHookUrl(e.target.value)}
              />
              <button
                type="button"
                className="rounded border px-3 py-2 text-sm"
                disabled={!hookUrl.trim()}
                onClick={() => void createWebhook()}
              >
                Create
              </button>
            </div>
            <div className="flex flex-wrap gap-3 text-sm">
              {EVENT_OPTIONS.map((ev) => (
                <label key={ev} className="flex items-center gap-1">
                  <input
                    type="checkbox"
                    checked={hookEvents.includes(ev)}
                    onChange={(e) => {
                      setHookEvents((prev) =>
                        e.target.checked ? [...prev, ev] : prev.filter((x) => x !== ev),
                      );
                    }}
                  />
                  {ev}
                </label>
              ))}
            </div>
            <ul className="divide-y border-t">
              {hooks.map((h) => (
                <li key={h.subscription_id} className="space-y-1 py-3 text-sm">
                  <div className="font-mono text-xs break-all">{h.url}</div>
                  <div className="text-xs text-[var(--desk-text-secondary)]">
                    {h.is_active ? "active" : "inactive"} · {(h.event_types ?? []).join(", ") || "*"}
                    {h.secret_prefix ? ` · ${h.secret_prefix}` : ""}
                  </div>
                  <div className="flex gap-3">
                    {h.is_active && (
                      <>
                        <button
                          type="button"
                          className="text-xs underline"
                          onClick={() => void pingWebhook(h.subscription_id)}
                        >
                          Ping
                        </button>
                        <button
                          type="button"
                          className="text-xs underline"
                          onClick={() => void deactivateWebhook(h.subscription_id)}
                        >
                          Deactivate
                        </button>
                      </>
                    )}
                  </div>
                </li>
              ))}
              {hooks.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_webhooks")}</li>
              )}
            </ul>
            <h3 className="pt-2 text-sm font-semibold">{t("supplier_portal.settings.integrations.text.dead_letter")}</h3>
            {dead.length > 0 && (
              <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-xs text-amber-900">
                <div className="flex items-center gap-1.5 font-semibold">
                  <svg className="h-4 w-4 shrink-0 text-amber-700" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                  <span>Dead Letter Queue (DLQ) Alerts ({dead.length})</span>
                </div>
                <p className="mt-1">
                  Deliveries have exhausted maximum retry attempts and routed to the DLQ. Review errors below or click Replay.
                </p>
              </div>
            )}
            <ul className="divide-y border-t">
              {dead.map((a) => (
                <li key={a.attempt_id} className="space-y-1.5 py-3 text-sm">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs">{a.event_type}</span>
                      <StatusChip status={a.status || "FAILED"} />
                    </div>
                    <button
                      type="button"
                      className="text-xs underline font-medium text-[var(--color-md-primary)] hover:opacity-80"
                      onClick={() => void replayDead(a.attempt_id)}
                    >
                      Replay
                    </button>
                  </div>
                  {a.last_error && (
                    <p className="rounded border bg-[var(--desk-canvas,#f8fafc)] p-2 font-mono text-xs text-red-600 break-all">
                      {a.last_error}
                    </p>
                  )}
                </li>
              ))}
              {dead.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_dead_letter_rows")}</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">1C chart of accounts</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Debit/credit accounts used when exporting <code className="text-xs">{t("supplier_portal.settings.integrations.text.journals")}</code>.
              {coaDefaults ? " Using platform defaults until you save." : " Custom map active."}
            </p>
            <div className="grid gap-2 sm:grid-cols-3">
              <label className="text-xs">
                AR (receivable)
                <input
                  className="mt-1 w-full rounded border px-3 py-2 text-sm font-mono"
                  value={coaAr}
                  onChange={(e) => setCoaAr(e.target.value)}
                />
              </label>
              <label className="text-xs">
                Revenue
                <input
                  className="mt-1 w-full rounded border px-3 py-2 text-sm font-mono"
                  value={coaRevenue}
                  onChange={(e) => setCoaRevenue(e.target.value)}
                />
              </label>
              <label className="text-xs">
                Bank / cash
                <input
                  className="mt-1 w-full rounded border px-3 py-2 text-sm font-mono"
                  value={coaBank}
                  onChange={(e) => setCoaBank(e.target.value)}
                />
              </label>
            </div>
            <button
              type="button"
              className="rounded bg-[var(--color-md-primary)] px-3 py-2 text-sm text-white"
              onClick={() => void saveCoa()}
            >
              Save CoA
            </button>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">{t("supplier_portal.settings.integrations.text.exports")}</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Async CSV/JSON/XML for orders, invoices, inventory, ledger, or 1C journals (90-day window
              max).
            </p>
            <div className="flex flex-wrap gap-2">
              <select
                className="rounded border px-2 py-2 text-sm"
                value={resource}
                onChange={(e) => setResource(e.target.value)}
              >
                <option value="orders">{t("supplier_portal.settings.integrations.text.orders")}</option>
                <option value="invoices">{t("supplier_portal.settings.integrations.text.invoices")}</option>
                <option value="inventory">{t("supplier_portal.settings.integrations.text.inventory")}</option>
                <option value="ledger">{t("supplier_portal.settings.integrations.text.ledger")}</option>
                <option value="journals">{t("supplier_portal.settings.integrations.text.journals_1c")}</option>
              </select>
              <select
                className="rounded border px-2 py-2 text-sm"
                value={format}
                onChange={(e) => setFormat(e.target.value)}
              >
                <option value="csv">{t("supplier_portal.settings.integrations.text.csv")}</option>
                <option value="json">{t("supplier_portal.settings.integrations.text.json")}</option>
                <option value="xml">{t("supplier_portal.settings.integrations.text.xml")}</option>
              </select>
              <input
                type="date"
                className="rounded border px-2 py-2 text-sm"
                value={from}
                onChange={(e) => setFrom(e.target.value)}
              />
              <input
                type="date"
                className="rounded border px-2 py-2 text-sm"
                value={to}
                onChange={(e) => setTo(e.target.value)}
              />
              <button
                type="button"
                className="rounded bg-[var(--color-md-primary)] px-3 py-2 text-sm text-white"
                onClick={() => void requestExport()}
              >
                Request export
              </button>
            </div>
            <ul className="divide-y border-t">
              {jobs.map((j) => (
                <li key={j.job_id} className="space-y-2 py-3 text-sm">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-xs">
                        {j.resource}/{j.format}
                      </span>
                      <StatusChip status={j.status} />
                      {j.sftp_status ? (
                        <span className="text-xs text-[var(--desk-text-secondary)]">
                          · sftp:{j.sftp_status}
                        </span>
                      ) : null}
                    </div>
                    {j.created_at && (
                      <span className="text-xs text-[var(--desk-text-secondary)]">
                        {new Date(j.created_at).toLocaleString()}
                      </span>
                    )}
                  </div>
                  {j.status === "FAILED" && (
                    <div className="rounded-md border border-[var(--desk-danger-soft,#fecaca)] bg-[var(--desk-danger-soft,#fef2f2)] p-2.5 text-xs text-[var(--desk-danger,#b91c1c)]">
                      <div className="flex items-center gap-1.5 font-semibold">
                        <svg className="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                        </svg>
                        <span>Export job failed</span>
                      </div>
                      <p className="mt-1 font-mono text-[11px] break-all">
                        {j.error || "Job processing failed after 4 retries (routed to DLQ)"}
                      </p>
                    </div>
                  )}
                  {j.download_url && (
                    <a className="text-xs underline break-all text-blue-600 hover:text-blue-800" href={j.download_url}>
                      Download
                    </a>
                  )}
                  {!j.download_url && j.status === "SUCCEEDED" && (
                    <button
                      type="button"
                      className="text-xs underline"
                      onClick={() =>
                        void api.getSupplierPartnerExport(j.job_id).then((latest) => {
                          if (latest.download_url) window.open(latest.download_url, "_blank");
                        })
                      }
                    >
                      Get download link
                    </button>
                  )}
                </li>
              ))}
              {jobs.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_export_jobs")}</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">{t("supplier_portal.settings.integrations.text.sftp_edi")}</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Optional SFTP for bulk export and EDI-lite (ORDERS in / ORDRSP·DESADV·INVOIC out). Secrets
              stay in GSM via <code className="text-xs">{t("supplier_portal.residual.text.secret_ref")}</code>.
            </p>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={ediEnabled}
                onChange={(e) => setEdiEnabled(e.target.checked)}
              />
              Enable EDI-lite for this supplier
            </label>
            <div className="grid gap-2 sm:grid-cols-2">
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder={t("supplier_portal.settings.integrations.text.host_or_leave_blank_for_local_edi_root")}
                value={sftpHost}
                onChange={(e) => setSftpHost(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder={t("supplier_portal.settings.integrations.text.port")}
                value={sftpPort}
                onChange={(e) => setSftpPort(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder={t("supplier_portal.settings.integrations.text.username")}
                value={sftpUser}
                onChange={(e) => setSftpUser(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder={t("supplier_portal.settings.integrations.text.gsm_secret_name_secret_ref")}
                value={sftpSecretRef}
                onChange={(e) => setSftpSecretRef(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm sm:col-span-2"
                placeholder={t("supplier_portal.settings.integrations.text.remote_base_directory")}
                value={sftpDir}
                onChange={(e) => setSftpDir(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder={t("supplier_portal.settings.integrations.text.inbound_dir")}
                value={inboundDir}
                onChange={(e) => setInboundDir(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder={t("supplier_portal.settings.integrations.text.outbound_dir")}
                value={outboundDir}
                onChange={(e) => setOutboundDir(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm sm:col-span-2"
                placeholder={t("supplier_portal.settings.integrations.text.archive_dir")}
                value={archiveDir}
                onChange={(e) => setArchiveDir(e.target.value)}
              />
            </div>
            <button
              type="button"
              className="rounded border px-3 py-2 text-sm"
              onClick={() => void saveSftp()}
            >
              Save SFTP / EDI config
            </button>
            {sftp.configured && (
              <p className="text-xs text-[var(--desk-text-secondary)]">
                Configured: {sftp.host}:{sftp.port} · EDI {sftp.edi_enabled ? "on" : "off"} ·{" "}
                {sftp.inbound_dir}/{sftp.outbound_dir}/{sftp.archive_dir}
              </p>
            )}
            <h3 className="pt-2 text-sm font-semibold">{t("supplier_portal.settings.integrations.text.recent_edi_documents")}</h3>
            <ul className="divide-y border-t">
              {ediDocs.map((doc) => (
                <li key={doc.document_id} className="flex flex-wrap items-center gap-3 py-3 text-sm">
                  <span className="font-mono text-xs">
                    {doc.direction}/{doc.doc_type}
                  </span>
                  <span className="text-xs">{doc.external_doc_id}</span>
                  <span>{doc.status}</span>
                  {doc.order_id && (
                    <span className="text-xs text-[var(--desk-text-secondary)]">{doc.order_id}</span>
                  )}
                  {(doc.status === "FAILED" || doc.status === "EMITTED") && (
                    <button
                      type="button"
                      className="text-xs underline"
                      onClick={() => void replayEdi(doc.document_id)}
                    >
                      Replay
                    </button>
                  )}
                </li>
              ))}
              {ediDocs.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_edi_documents")}</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">AS2 transport</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Optional RFC 4130 AS2 pipe for the same EDI-lite ORDERS/ORDRSP/DESADV/INVOIC bytes.
              Receive at <code className="text-xs">POST /partner/v1/as2</code>. Not Drummond-certified.
            </p>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={as2Enabled}
                onChange={(e) => setAs2Enabled(e.target.checked)}
              />
              Enable AS2 for this supplier
            </label>
            <div className="grid gap-2 sm:grid-cols-2">
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder="Our AS2-ID"
                value={ourAs2Id}
                onChange={(e) => setOurAs2Id(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder="Partner AS2-ID"
                value={partnerAs2Id}
                onChange={(e) => setPartnerAs2Id(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm sm:col-span-2"
                placeholder="Partner AS2 URL (HTTPS)"
                value={partnerAs2Url}
                onChange={(e) => setPartnerAs2Url(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder="Our cert secret ref"
                value={ourCertRef}
                onChange={(e) => setOurCertRef(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder="Our key secret ref"
                value={ourKeyRef}
                onChange={(e) => setOurKeyRef(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm sm:col-span-2"
                placeholder="Partner cert secret ref"
                value={partnerCertRef}
                onChange={(e) => setPartnerCertRef(e.target.value)}
              />
            </div>
            <button type="button" className="rounded border px-3 py-2 text-sm" onClick={() => void saveAs2()}>
              Save AS2 config
            </button>
            {as2.configured && (
              <p className="text-xs text-[var(--desk-text-secondary)]">
                Configured: {as2.our_as2_id} ↔ {as2.partner_as2_id} · AS2 {as2.as2_enabled ? "on" : "off"}
              </p>
            )}
          </section>
        </div>
      )}
    </PageChrome>
  );
}
