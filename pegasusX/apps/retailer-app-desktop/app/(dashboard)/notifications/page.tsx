"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { AlertTriangle, CheckCheck, RefreshCw, WifiOff } from "lucide-react";
import { HandoffCard } from "@pegasusx/explain-ui";
import { useRouter } from "next/navigation";
import { PageChrome } from "@/components/PageChrome";
import EmptyState from "../../../components/EmptyState";
import { ListRowSkeleton } from "../../../components/Skeleton";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import { useRetailerNotifications } from "../../../lib/notifications";
import { useOptionalWebSocket } from "../../../lib/ws";

export default function NotificationsPage() {
  const router = useRouter();
  const {
    items,
    unreadCount,
    loading,
    isLoadingMore,
    hasMore,
    error,
    refresh,
    loadMore,
    markRead,
    markAllRead,
  } = useRetailerNotifications();
  const ws = useOptionalWebSocket();
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const loadMoreRef = useRef<HTMLDivElement | null>(null);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    setActionError(null);
    try {
      await refresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Refresh failed");
    } finally {
      setIsRefreshing(false);
    }
  }, [refresh]);

  const handleMarkAllRead = useCallback(async () => {
    setActionError(null);
    try {
      await markAllRead();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Mark all read failed");
    }
  }, [markAllRead]);

  const handleMarkRead = useCallback(async (notificationId: string) => {
    setActionError(null);
    try {
      await markRead(notificationId);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Mark read failed");
    }
  }, [markRead]);

  useRetailerSessionReconcile(() => {
    void handleRefresh();
  });

  useEffect(() => {
    const sentinel = loadMoreRef.current;
    if (!sentinel || !hasMore) {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          void loadMore();
        }
      },
      { rootMargin: "240px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasMore, items.length, loadMore]);

  const loadIssue = useMemo<"restricted" | "offline" | "error" | null>(() => {
    const message = actionError ?? error;
    if (!message) return null;
    if (/401|403|forbidden|restricted|access/i.test(message)) {
      return "restricted";
    }
    if (
      (typeof navigator !== "undefined" && !navigator.onLine) ||
      /network|failed to fetch|load failed|offline/i.test(message)
    ) {
      return "offline";
    }
    return "error";
  }, [actionError, error]);

  const syncBanner = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Notifications access restricted for this account.",
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: "Offline mode active. Showing latest cached inbox data.",
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Inbox sync degraded. Retry is available.",
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Live socket reconnecting. New alerts may be delayed.",
      };
    }
    if (isRefreshing) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: "Syncing notifications...",
      };
    }
    return null;
  }, [isRefreshing, loadIssue, ws]);

  const emptyStateConfig = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        headline: "Notifications access restricted",
        body: "Your account currently cannot load inbox alerts.",
        variant: "restricted" as const,
      };
    }
    if (loadIssue === "offline") {
      return {
        headline: "Notifications are offline",
        body: "Reconnect and retry to refresh alert history.",
        variant: "offline" as const,
      };
    }
    if (loadIssue === "error") {
      return {
        headline: "Notifications unavailable",
        body: "Inbox data could not be loaded right now.",
        variant: "error" as const,
      };
    }
    return {
      headline: "No notifications yet",
      body: "Order status changes, delivery alerts, and preorder updates will appear here.",
      variant: "no-data" as const,
    };
  }, [loadIssue]);

  const headerSubtitle = useMemo(() => {
    if (unreadCount === 0) {
      return "All alerts and order updates in one inbox.";
    }
    return `${unreadCount} unread notification${unreadCount === 1 ? "" : "s"} waiting.`;
  }, [unreadCount]);

  return (
    <div className="min-h-full p-6 md:p-8">
      <PageChrome
        icon="notifications"
        title="Notifications"
        description={headerSubtitle}
        loading={loading && items.length === 0}
        skeletonVariant="table"
        actions={
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void handleRefresh()}
              disabled={isRefreshing}
              aria-label="Refresh"
              className="portal-btn portal-btn--ghost desk-icon-btn"
            >
              <RefreshCw size={16} className={isRefreshing ? "animate-spin" : ""} />
            </button>
            <button
              type="button"
              disabled={unreadCount === 0}
              onClick={() => void handleMarkAllRead()}
              className="portal-btn portal-btn--primary flex items-center gap-2"
            >
              <CheckCheck size={16} />
              Mark All Read
            </button>
          </div>
        }
      >

      {syncBanner && (
        <motion.div
          initial={{ opacity: 0, y: -6 }}
          animate={{ opacity: 1, y: 0 }}
          className={`mb-6 flex items-center justify-between gap-3 rounded-2xl border px-4 py-3 ${
            syncBanner.kind === "refreshing"
              ? "border-[var(--desk-info)]/30 bg-[var(--desk-info)]/5 text-[var(--desk-info)]"
              : "border-[var(--desk-warning)]/30 bg-[var(--desk-warning)]/10 text-[var(--desk-warning)]"
          }`}
        >
          <div className="flex items-center gap-2">
            <syncBanner.icon
              size={16}
              className={syncBanner.kind === "refreshing" ? "animate-spin" : ""}
            />
            <span className="md-typescale-body-small font-light uppercase tracking-wide">
              {syncBanner.message}
            </span>
          </div>
          {syncBanner.kind !== "refreshing" && (
            <button
              onClick={() => void handleRefresh()}
              className="rounded-lg border border-current/30 px-3 py-1 text-[11px] font-light uppercase tracking-wide hover:bg-current/10"
            >
              Retry
            </button>
          )}
        </motion.div>
      )}

      <AnimatePresence mode="popLayout">
        {loading && items.length === 0 ? (
          <motion.div key="loading" className="grid gap-4">
            <ListRowSkeleton count={4} />
          </motion.div>
        ) : items.length === 0 ? (
          <motion.div
            key="empty"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="py-16"
          >
            <EmptyState
              headline={emptyStateConfig.headline}
              body={emptyStateConfig.body}
              variant={emptyStateConfig.variant}
              action="Refresh Inbox"
              onAction={() => void handleRefresh()}
            />
          </motion.div>
        ) : (
          <motion.div key="list" layout className="grid gap-4">
            {items.map((item) => (
              <motion.button
                key={item.id}
                layout
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.95 }}
                type="button"
                className={`bento-card w-full text-left transition-all hover:bg-surface-subtle ${item.readAt == null ? "ring-1 ring-[var(--desk-accent)]" : ""}`}
                onClick={() => {
                  if (item.readAt == null) {
                    void handleMarkRead(item.id);
                  }
                }}
              >
                <div className="flex items-start gap-4">
                  <div
                    className={`mt-1 h-2.5 w-2.5 rounded-full ${item.readAt == null ? "bg-accent" : "bg-outline"}`}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
                      <div className="min-w-0">
                        <h2 className="md-typescale-title-medium truncate text-foreground">
                          {item.title}
                        </h2>
                        <p className="md-typescale-body-medium mt-1 text-muted">
                          {item.body}
                        </p>
                        {item.handoffMetadata ? (
                          <div
                            className="mt-3 rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] p-3"
                            onClick={(e) => e.stopPropagation()}
                            role="presentation"
                          >
                            <HandoffCard
                              metadata={item.handoffMetadata}
                              onAction={(link) => router.push(link)}
                            />
                          </div>
                        ) : null}
                      </div>
                      <div className="flex items-center gap-2 text-xs uppercase tracking-widest text-muted">
                        <span>{formatRelativeTime(item.createdAt)}</span>
                        <span className="rounded-full border border-[var(--desk-border)] px-2 py-1">
                          {item.type.replaceAll("_", " ")}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </motion.button>
            ))}
            {hasMore && <div ref={loadMoreRef} className="h-8" aria-hidden="true" />}
            {isLoadingMore && (
              <div className="flex justify-center py-4">
                <RefreshCw size={18} className="animate-spin text-[var(--desk-accent)]" />
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
      </PageChrome>
    </div>
  );
}

function formatRelativeTime(value: string): string {
  const created = Date.parse(value);
  if (Number.isNaN(created)) {
    return "Now";
  }

  const diffMinutes = Math.max(0, Math.floor((Date.now() - created) / 60000));
  if (diffMinutes < 1) {
    return "Now";
  }
  if (diffMinutes < 60) {
    return `${diffMinutes}m ago`;
  }

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) {
    return `${diffHours}h ago`;
  }

  return `${Math.floor(diffHours / 24)}d ago`;
}
