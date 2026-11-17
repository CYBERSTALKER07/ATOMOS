"use client";

import { useCallback, useEffect, useState } from "react";
import { Send, Copy, Check, ExternalLink, RefreshCw, Loader2 } from "lucide-react";
import { usePortalLocale } from "@/lib/i18n";
import {
  FAST_HOME_COPY,
  type TelegramLinkRequestPayload,
  type TelegramLinkStatusPayload,
} from "@/lib/fast-home";
import { getTelegramLinkStatus, requestTelegramLink } from "@/lib/api";

export function TelegramLinkCard() {
  const locale = usePortalLocale();
  const copy = FAST_HOME_COPY[locale] ?? FAST_HOME_COPY.uz;

  const [status, setStatus] = useState<TelegramLinkStatusPayload | null>(null);
  const [linkRequest, setLinkRequest] = useState<TelegramLinkRequestPayload | null>(null);
  const [loadingStatus, setLoadingStatus] = useState(false);
  const [requesting, setRequesting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedField, setCopiedField] = useState<"otp" | "link" | null>(null);

  const loadStatus = useCallback(async () => {
    setLoadingStatus(true);
    setError(null);
    try {
      const next = await getTelegramLinkStatus();
      setStatus(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load Telegram status");
    } finally {
      setLoadingStatus(false);
    }
  }, []);

  useEffect(() => {
    void loadStatus();
  }, [loadStatus]);

  const handleRequestLink = async () => {
    setRequesting(true);
    setError(null);
    try {
      const payload = await requestTelegramLink();
      setLinkRequest(payload);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to request Telegram OTP");
    } finally {
      setRequesting(false);
    }
  };

  const copyText = async (text: string, field: "otp" | "link") => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      window.setTimeout(() => {
        setCopiedField((prev) => (prev === field ? null : prev));
      }, 2000);
    } catch {
      // Clipboard may be restricted in non-HTTPS test context
    }
  };

  const deepLink =
    linkRequest?.bot_deep_link ||
    linkRequest?.deep_link ||
    status?.bot_deep_link ||
    status?.deep_link ||
    "";

  return (
    <div
      className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-5 shadow-[var(--shadow-sm)]"
      data-testid="telegram-link-card"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-xl bg-sky-500/15 text-sky-400 flex items-center justify-center shrink-0">
            <Send size={18} />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-base font-medium text-[var(--desk-text-primary)]">
                {copy.telegramTitle}
              </h3>
              <span
                className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                  status?.linked
                    ? "bg-emerald-500/15 text-emerald-300"
                    : "bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)]"
                }`}
              >
                {status?.linked ? copy.telegramLinked : copy.telegramNotLinked}
              </span>
            </div>
            <p className="mt-1 text-xs text-[var(--desk-text-secondary)]">
              {copy.telegramSubtitle}
            </p>
            {status?.linked && (status.username || status.telegram_chat_id) && (
              <p className="mt-1 text-xs font-mono text-[var(--desk-text-tertiary)]">
                {status.username ? `@${status.username}` : `Chat #${status.telegram_chat_id}`}
              </p>
            )}
          </div>
        </div>

        <button
          type="button"
          onClick={() => void loadStatus()}
          disabled={loadingStatus}
          className="portal-btn portal-btn--ghost rounded-xl px-3 py-1.5 text-xs"
        >
          <RefreshCw size={13} className={loadingStatus ? "animate-spin mr-1" : "mr-1"} />
          {copy.telegramRefreshStatus}
        </button>
      </div>

      {error && (
        <p className="mt-3 rounded-xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
          {error}
        </p>
      )}

      {linkRequest && (
        <div className="mt-4 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] p-4 space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-[11px] uppercase tracking-wider text-[var(--desk-text-tertiary)]">
                OTP
              </p>
              <p
                className="text-2xl font-mono font-semibold tracking-[0.22em] text-[var(--desk-text-primary)]"
                data-testid="telegram-otp-code"
              >
                {linkRequest.otp_code}
              </p>
              {linkRequest.expires_at && (
                <p className="mt-0.5 text-[11px] text-[var(--desk-text-tertiary)]">
                  {copy.telegramExpiresAt(linkRequest.expires_at)}
                </p>
              )}
            </div>

            <button
              type="button"
              onClick={() => void copyText(linkRequest.otp_code, "otp")}
              className="portal-btn portal-btn--ghost rounded-xl px-3 py-2 text-xs"
              data-testid="telegram-copy-otp"
            >
              {copiedField === "otp" ? (
                <>
                  <Check size={14} className="mr-1 text-emerald-400" />
                  {copy.telegramCopied}
                </>
              ) : (
                <>
                  <Copy size={14} className="mr-1" />
                  {copy.telegramCopyCode}
                </>
              )}
            </button>
          </div>

          {deepLink && (
            <div className="flex flex-wrap items-center gap-2 pt-2 border-t border-[var(--desk-border)]">
              <button
                type="button"
                onClick={() => void copyText(deepLink, "link")}
                className="portal-btn portal-btn--ghost rounded-xl px-3 py-1.5 text-xs"
                data-testid="telegram-copy-link"
              >
                {copiedField === "link" ? (
                  <>
                    <Check size={13} className="mr-1 text-emerald-400" />
                    {copy.telegramCopied}
                  </>
                ) : (
                  <>
                    <Copy size={13} className="mr-1" />
                    {copy.telegramCopyLink}
                  </>
                )}
              </button>
              <a
                href={deepLink}
                target="_blank"
                rel="noopener noreferrer"
                className="portal-btn portal-btn--primary rounded-xl px-3 py-1.5 text-xs"
              >
                <ExternalLink size={13} className="mr-1" />
                {copy.telegramOpenBot}
              </a>
            </div>
          )}
        </div>
      )}

      <div className="mt-4">
        <button
          type="button"
          onClick={() => void handleRequestLink()}
          disabled={requesting}
          className="portal-btn portal-btn--primary rounded-xl px-4 py-2 text-xs"
          data-testid="telegram-generate-otp"
        >
          {requesting && <Loader2 size={14} className="animate-spin mr-1.5" />}
          {copy.telegramGenerateCode}
        </button>
      </div>
    </div>
  );
}
