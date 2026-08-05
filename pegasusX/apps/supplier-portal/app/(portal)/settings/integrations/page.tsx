"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type {
  PartnerApiKeyMeta,
  PartnerDeadLetterAttempt,
  PartnerEdiDocument,
  PartnerExportJob,
  PartnerSftpConfig,
  PartnerWebhookSubscription,
} from "@pegasusx/types";

const api = createSupplierApi();

const EVENT_OPTIONS = ["ORDER_CREATED", "ORDER_STATUS_CHANGED", "CLAIM_FILED", "PAYMENT_CLEARED"];

export default function IntegrationsSettingsPage() {
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

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [k, w, d, e, s, edi] = await Promise.all([
        api.listSupplierPartnerKeys(),
        api.listSupplierPartnerWebhooks(),
        api.listSupplierPartnerDeadLetter(),
        api.listSupplierPartnerExports(),
        api.getSupplierPartnerSftp(),
        api.listSupplierPartnerEdiDocuments().catch(() => ({ documents: [] as PartnerEdiDocument[] })),
      ]);
      setKeys(k.keys ?? []);
      setHooks(w.subscriptions ?? []);
      setDead(d.attempts ?? []);
      setJobs(e.jobs ?? []);
      setEdiDocs(edi.documents ?? []);
      setSftp(s);
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
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load integrations");
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
      setError(err instanceof Error ? err.message : "Issue key failed");
    }
  }

  async function revokeKey(keyId: string) {
    setNotice(null);
    try {
      await api.revokeSupplierPartnerKey(keyId);
      setNotice("Key revoked.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Revoke failed");
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
      setError(err instanceof Error ? err.message : "Create webhook failed");
    }
  }

  async function pingWebhook(id: string) {
    try {
      await api.pingSupplierPartnerWebhook(id);
      setNotice("Ping sent.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ping failed");
    }
  }

  async function deactivateWebhook(id: string) {
    try {
      await api.deactivateSupplierPartnerWebhook(id);
      setNotice("Webhook deactivated.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Deactivate failed");
    }
  }

  async function replayDead(id: string) {
    try {
      await api.replaySupplierPartnerDeadLetter(id);
      setNotice("Dead-letter requeued.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Replay failed");
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
      setError(err instanceof Error ? err.message : "Export failed");
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
      setError(err instanceof Error ? err.message : "SFTP save failed");
    }
  }

  async function replayEdi(id: string) {
    try {
      await api.replaySupplierPartnerEdiDocument(id);
      setNotice("EDI document requeued.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "EDI replay failed");
    }
  }

  return (
    <PageChrome
      title="Integrations"
      description="Partner API keys, outbound webhooks, bulk exports, and optional SFTP drop."
    >
      {loading ? (
        <p className="text-sm text-[var(--desk-text-secondary)]">Loading…</p>
      ) : (
        <div className="mx-auto max-w-3xl space-y-10">
          {error && <p className="text-sm font-semibold text-red-600">{error}</p>}
          {notice && <p className="text-sm text-emerald-700">{notice}</p>}

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">API keys</h2>
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
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">No keys yet.</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">Webhooks</h2>
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
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">No webhooks.</li>
              )}
            </ul>
            <h3 className="pt-2 text-sm font-semibold">Dead letter</h3>
            <ul className="divide-y border-t">
              {dead.map((a) => (
                <li key={a.attempt_id} className="flex flex-wrap items-center gap-3 py-3 text-sm">
                  <span className="font-mono text-xs">{a.event_type}</span>
                  <span className="text-xs text-[var(--desk-text-secondary)]">
                    {a.last_error || a.status}
                  </span>
                  <button
                    type="button"
                    className="text-xs underline"
                    onClick={() => void replayDead(a.attempt_id)}
                  >
                    Replay
                  </button>
                </li>
              ))}
              {dead.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">No dead-letter rows.</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">Exports</h2>
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
                <option value="orders">orders</option>
                <option value="invoices">invoices</option>
                <option value="inventory">inventory</option>
                <option value="ledger">ledger</option>
                <option value="journals">journals (1C)</option>
              </select>
              <select
                className="rounded border px-2 py-2 text-sm"
                value={format}
                onChange={(e) => setFormat(e.target.value)}
              >
                <option value="csv">csv</option>
                <option value="json">json</option>
                <option value="xml">xml</option>
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
                <li key={j.job_id} className="space-y-1 py-3 text-sm">
                  <div>
                    {j.resource}/{j.format} · {j.status}
                    {j.sftp_status ? ` · sftp:${j.sftp_status}` : ""}
                  </div>
                  {j.download_url && (
                    <a className="text-xs underline break-all" href={j.download_url}>
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
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">No export jobs.</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">SFTP + EDI</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Optional SFTP for bulk export and EDI-lite (ORDERS in / ORDRSP·DESADV·INVOIC out). Secrets
              stay in GSM via <code className="text-xs">secret_ref</code>.
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
                placeholder="Host (or leave blank for local EDI root)"
                value={sftpHost}
                onChange={(e) => setSftpHost(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder="Port"
                value={sftpPort}
                onChange={(e) => setSftpPort(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder="Username"
                value={sftpUser}
                onChange={(e) => setSftpUser(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder="GSM secret name (secret_ref)"
                value={sftpSecretRef}
                onChange={(e) => setSftpSecretRef(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm sm:col-span-2"
                placeholder="Remote base directory"
                value={sftpDir}
                onChange={(e) => setSftpDir(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder="Inbound dir"
                value={inboundDir}
                onChange={(e) => setInboundDir(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm"
                placeholder="Outbound dir"
                value={outboundDir}
                onChange={(e) => setOutboundDir(e.target.value)}
              />
              <input
                className="rounded border px-3 py-2 text-sm sm:col-span-2"
                placeholder="Archive dir"
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
            <h3 className="pt-2 text-sm font-semibold">Recent EDI documents</h3>
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
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">No EDI documents.</li>
              )}
            </ul>
          </section>
        </div>
      )}
    </PageChrome>
  );
}
