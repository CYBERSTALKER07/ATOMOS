"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  Loader2,
  Layers,
  AlertTriangle,
  CheckCircle2,
  Info,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";

type PackStatus = {
  id: string;
  name: string;
  description: string;
  hard_deps?: string[];
  soft_deps?: string[];
  always_on?: boolean;
  enabled: boolean;
  config?: Record<string, unknown>;
};

type CapabilitiesResponse = {
  retailer_id: string;
  capabilities: string[];
  packs: PackStatus[];
};

type EvalResult = {
  status: string;
  pack_id?: string;
  message?: string;
  missing_hard_deps?: string[];
  missing_soft_deps?: string[];
  enable_all?: string[];
};

export default function CapabilitiesPage() {
  const t = usePortalT();
  const router = useRouter();
  const [data, setData] = useState<CapabilitiesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyPack, setBusyPack] = useState<string | null>(null);
  const [banner, setBanner] = useState<{ kind: "ok" | "err" | "warn"; text: string } | null>(null);
  const [pendingWarn, setPendingWarn] = useState<{ packId: string; eval: EvalResult } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/capabilities");
      if (!res.ok) {
        throw new Error(`load_failed_${res.status}`);
      }
      const json = (await res.json()) as CapabilitiesResponse;
      setData(json);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.failed_to_load_capabilities"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const enablePack = async (packId: string, opts?: { acceptSoft?: boolean; enableDeps?: boolean }) => {
    setBusyPack(packId);
    setBanner(null);
    setPendingWarn(null);
    try {
      const res = await apiFetch(`/v1/retailer/capabilities/${packId}/enable`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "Idempotency-Key": `cap-en-${packId}-${Date.now()}` },
        body: JSON.stringify({
          accept_soft_deps: !!opts?.acceptSoft,
          enable_deps: !!opts?.enableDeps,
          config: {},
        }),
      });
      const json = (await res.json()) as EvalResult & { capabilities?: string[]; status?: string };
      if (res.status === 409) {
        if (json.status === "WARN") {
          setPendingWarn({ packId, eval: json });
          setBanner({ kind: "warn", text: json.message || "This pack has recommended dependencies." });
        } else {
          setPendingWarn({ packId, eval: json });
          setBanner({
            kind: "err",
            text: json.message || `Cannot enable ${packId} without dependencies.`,
          });
        }
        return;
      }
      if (!res.ok) {
        throw new Error(json.message || `enable_failed_${res.status}`);
      }
      setBanner({ kind: "ok", text: `${packId} enabled` });
      await load();
    } catch (e) {
      setBanner({ kind: "err", text: e instanceof Error ? e.message : t("retailer_desktop.residual.text.enable_failed") });
    } finally {
      setBusyPack(null);
    }
  };

  const disablePack = async (packId: string) => {
    setBusyPack(packId);
    setBanner(null);
    try {
      const res = await apiFetch(`/v1/retailer/capabilities/${packId}/disable`, {
        method: "POST",
        headers: { "Idempotency-Key": `cap-dis-${packId}-${Date.now()}` },
      });
      const json = (await res.json()) as EvalResult;
      if (res.status === 409) {
        setBanner({ kind: "err", text: json.message || "Disable blocked by dependent packs" });
        return;
      }
      if (!res.ok) {
        throw new Error(json.message || `disable_failed_${res.status}`);
      }
      setBanner({ kind: "ok", text: `${packId} disabled` });
      await load();
    } catch (e) {
      setBanner({ kind: "err", text: e instanceof Error ? e.message : t("retailer_desktop.residual.text.disable_failed") });
    } finally {
      setBusyPack(null);
    }
  };

  return (
    <PageChrome
      title={t("retailer_desktop.settings.capabilities.text.store_capabilities")}
      description={t("retailer_desktop.residual.text.turn_on_only_what_your_shop_needs_solo_shops_run_on_core_alone")}
      actions={
        <button
          type="button"
          onClick={() => router.push("/settings")}
          className="inline-flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm hover:bg-muted"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to settings
        </button>
      }
    >
      <div className="mx-auto max-w-3xl space-y-4 px-4 pb-16 pt-2">
        <div className="rounded-xl border border-border bg-card p-4 text-sm text-muted-foreground">
          <div className="mb-1 flex items-center gap-2 font-medium text-foreground">
            <Info className="h-4 w-4" />
            Minimum path first
          </div>
          Core procurement stays on forever. Enable Team, Stock, POS, and other packs when you grow —
          the system blocks unsafe combinations and warns on recommended ones.
        </div>

        {banner && (
          <div
            className={`rounded-lg border px-3 py-2 text-sm ${
              banner.kind === "ok"
                ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                : banner.kind === "warn"
                  ? "border-amber-500/40 bg-amber-500/10 text-amber-800 dark:text-amber-200"
                  : "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300"
            }`}
          >
            <div className="flex items-start gap-2">
              {banner.kind === "ok" ? (
                <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
              ) : (
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              )}
              <div className="flex-1">{banner.text}</div>
            </div>
            {pendingWarn && (
              <div className="mt-3 flex flex-wrap gap-2">
                {pendingWarn.eval.status === "WARN" && (
                  <button
                    type="button"
                    className="rounded-md bg-foreground px-3 py-1.5 text-xs font-medium text-background"
                    onClick={() => void enablePack(pendingWarn.packId, { acceptSoft: true })}
                  >
                    Continue without recommended packs
                  </button>
                )}
                {pendingWarn.eval.enable_all && pendingWarn.eval.enable_all.length > 0 && (
                  <button
                    type="button"
                    className="rounded-md border border-border px-3 py-1.5 text-xs font-medium"
                    onClick={() =>
                      void enablePack(pendingWarn.packId, {
                        acceptSoft: true,
                        enableDeps: true,
                      })
                    }
                  >
                    Enable required packs too
                  </button>
                )}
              </div>
            )}
          </div>
        )}

        {loading && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading capabilities…
          </div>
        )}
        {error && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-700">
            {error}
            <button type="button" className="ml-3 underline" onClick={() => void load()}>
              Retry
            </button>
          </div>
        )}

        <div className="space-y-3">
          {(data?.packs ?? []).map((pack) => {
            const busy = busyPack === pack.id;
            return (
              <div
                key={pack.id}
                className="rounded-xl border border-border bg-card p-4 shadow-sm"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <Layers className="h-4 w-4 text-muted-foreground" />
                      <h3 className="font-semibold text-foreground">{pack.name}</h3>
                      <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                        {pack.id}
                      </span>
                      {pack.enabled && (
                        <span className="rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-semibold uppercase text-emerald-700 dark:text-emerald-300">
                          On
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">{pack.description}</p>
                    {(pack.hard_deps?.length || pack.soft_deps?.length) && (
                      <p className="mt-2 text-xs text-muted-foreground">
                        {pack.hard_deps && pack.hard_deps.length > 0 && (
                          <span>Requires: {pack.hard_deps.join(", ")}. </span>
                        )}
                        {pack.soft_deps && pack.soft_deps.length > 0 && (
                          <span>Recommended: {pack.soft_deps.join(", ")}.</span>
                        )}
                      </p>
                    )}
                  </div>
                  <div className="shrink-0">
                    {pack.always_on ? (
                      <span className="text-xs font-medium text-muted-foreground">{t("retailer_desktop.settings.capabilities.text.always_on")}</span>
                    ) : pack.enabled ? (
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => void disablePack(pack.id)}
                        className="rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-muted disabled:opacity-50"
                      >
                        {busy ? "…" : "Disable"}
                      </button>
                    ) : (
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => void enablePack(pack.id)}
                        className="rounded-lg bg-foreground px-3 py-1.5 text-sm font-medium text-background disabled:opacity-50"
                      >
                        {busy ? "…" : "Enable"}
                      </button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </PageChrome>
  );
}
