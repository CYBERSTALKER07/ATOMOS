"use client";

import { useMemo } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { AlertTriangle, Bell, CheckCheck, RefreshCw } from "lucide-react";
import { Button } from "@heroui/react";
import { useRetailerNotifications } from "../../../lib/notifications";

export default function NotificationsPage() {
  const { items, unreadCount, loading, error, refresh, markRead, markAllRead } =
    useRetailerNotifications();

  const headerSubtitle = useMemo(() => {
    if (unreadCount === 0) {
      return "All alerts and order updates in one inbox.";
    }
    return `${unreadCount} unread notification${unreadCount === 1 ? "" : "s"} waiting.`;
  }, [unreadCount]);

  return (
    <div className="min-h-full p-6 md:p-8">
      <header className="mb-8 flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <Bell size={18} style={{ color: "var(--desk-accent)" }} />
            <h1 className="md-typescale-headline-large">Notifications</h1>
          </div>
          <p
            className="md-typescale-body-medium mt-1"
            style={{ color: "var(--desk-text-secondary)" }}
          >
            {headerSubtitle}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="flat"
            onPress={() => void refresh()}
            isIconOnly
            aria-label="Refresh"
          >
            <RefreshCw size={16} />
          </Button>
          <Button
            variant="solid"
            color="primary"
            isDisabled={unreadCount === 0}
            onPress={() => void markAllRead()}
            className="flex items-center gap-2"
          >
            <CheckCheck size={16} />
            Mark All Read
          </Button>
        </div>
      </header>

      <AnimatePresence mode="popLayout">
        {loading ? (
          <motion.div key="loading" className="grid gap-4">
            {[0, 1, 2, 3].map((item) => (
              <motion.div key={item} className="bento-card h-24 opacity-50" />
            ))}
          </motion.div>
        ) : error && items.length === 0 ? (
          <motion.div
            key="error"
            className="bento-card flex items-center gap-3"
          >
            <AlertTriangle size={18} style={{ color: "var(--desk-warning)" }} />
            <span className="md-typescale-body-medium text-muted">
              Could not load notifications.
            </span>
            <Button
              size="sm"
              variant="flat"
              onPress={() => void refresh()}
              className="ml-auto"
            >
              Retry
            </Button>
          </motion.div>
        ) : items.length === 0 ? (
          <motion.div
            key="empty"
            className="flex flex-col items-center justify-center gap-3 py-16 text-center"
          >
            <Bell size={48} style={{ color: "var(--desk-text-tertiary)" }} />
            <div>
              <h2 className="md-typescale-title-large font-semibold text-foreground">
                No notifications yet
              </h2>
              <p className="md-typescale-body-medium mt-1 text-muted">
                Order status changes, delivery alerts, and preorder updates will
                appear here.
              </p>
            </div>
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
                    void markRead(item.id);
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
          </motion.div>
        )}
      </AnimatePresence>
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
