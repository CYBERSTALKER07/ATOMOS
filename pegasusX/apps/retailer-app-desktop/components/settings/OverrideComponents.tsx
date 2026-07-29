import { useState, useEffect } from "react";
import { Loader2, CheckCircle2, ChevronDown, ChevronRight } from "lucide-react";
import { Chip } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";

function getBrowserStorage(): Storage | null {
  if (
    typeof window === "undefined" ||
    typeof window.localStorage?.getItem !== "function"
  ) {
    return null;
  }
  return window.localStorage;
}

export function Toggle({
  on,
  onToggle,
  disabled = false,
}: {
  on: boolean;
  onToggle: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      onClick={onToggle}
      disabled={disabled}
      className="w-11 h-6 rounded-full flex items-center p-1 cursor-pointer transition-all duration-200 disabled:opacity-50 shrink-0"
      style={{ background: on ? "var(--desk-accent)" : "var(--desk-border)" }}
    >
      {disabled ? (
        <Loader2 size={14} className="animate-spin mx-auto text-white" />
      ) : (
        <motion.div
          layout
          className="w-4 h-4 rounded-full bg-white shadow-sm"
          initial={false}
          animate={{ x: on ? 20 : 0 }}
          transition={{ type: "spring", stiffness: 500, damping: 30 }}
        />
      )}
    </button>
  );
}

export function OverrideRow({
  id,
  label,
  enabled,
  hasHistory,
  analyticsDate,
  icon: Icon,
  onToggle,
  saving,
}: {
  id: string;
  label: string;
  enabled: boolean;
  hasHistory?: boolean;
  analyticsDate?: string;
  icon: React.ElementType;
  onToggle: () => void;
  saving: boolean;
}) {
  return (
    <div className="flex items-center justify-between py-3 border-b border-[var(--desk-border)] last:border-0 gap-4">
      <div className="flex items-center gap-3 min-w-0">
        <div className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0 bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]">
          <Icon size={16} className="text-[var(--desk-text-tertiary)]" />
        </div>
        <div className="min-w-0">
          <span className="md-typescale-body-medium font-light text-[var(--desk-text-primary)] block truncate">
            {label}
          </span>
          <div className="flex items-center gap-2 mt-0.5">
            {hasHistory && (
              <span className="md-typescale-label-small text-[var(--desk-success)] flex items-center gap-1 font-light uppercase tracking-tighter">
                <CheckCircle2 size={10} /> Active History
              </span>
            )}
            {analyticsDate && (
              <span className="md-typescale-label-small text-[var(--desk-text-tertiary)] uppercase font-light tracking-tighter">
                Since {new Date(analyticsDate).toLocaleDateString()}
              </span>
            )}
          </div>
        </div>
      </div>
      <Toggle on={enabled} onToggle={onToggle} disabled={saving} />
    </div>
  );
}

export function OverrideSection<T extends { enabled: boolean }>({
  title,
  icon: Icon,
  items,
  getId,
  getLabel,
  getHasHistory,
  getAnalyticsDate,
  rowIcon,
  onToggle,
  savingId,
  storageKey,
}: {
  title: string;
  icon: React.ElementType;
  items: T[];
  getId: (item: T) => string;
  getLabel: (item: T) => string;
  getHasHistory?: (item: T) => boolean;
  getAnalyticsDate?: (item: T) => string | undefined;
  rowIcon: React.ElementType;
  onToggle: (id: string, enabled: boolean) => void;
  savingId: string | null;
  storageKey?: string;
}) {
  const [open, setOpen] = useState(() => {
    if (!storageKey) return items.length <= 6;
    const saved = getBrowserStorage()?.getItem(storageKey);
    if (saved === "1") return true;
    if (saved === "0") return false;
    return items.length <= 6;
  });

  useEffect(() => {
    if (!storageKey) return;
    getBrowserStorage()?.setItem(storageKey, open ? "1" : "0");
  }, [open, storageKey]);

  if (items.length === 0) return null;

  const enabledCount = items.filter((i) => i.enabled).length;

  return (
    <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-6 shadow-[var(--shadow-sm)]">
      <button
        onClick={() => setOpen((p) => !p)}
        className="flex items-center justify-between w-full cursor-pointer group"
      >
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] flex items-center justify-center">
            <Icon size={20} />
          </div>
          <span className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
            {title}
          </span>
          <Chip
            size="sm"
            variant="secondary"
            className="font-light text-[10px] ml-1"
          >
            {enabledCount}/{items.length} ACTIVE
          </Chip>
        </div>
        <div className="text-[var(--desk-text-tertiary)] group-hover:text-[var(--desk-text-primary)] transition-colors">
          {open ? <ChevronDown size={20} /> : <ChevronRight size={20} />}
        </div>
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            className="mt-6 overflow-hidden"
          >
            <div className="space-y-1">
              {items.map((item) => {
                const id = getId(item);
                return (
                  <OverrideRow
                    key={id}
                    id={id}
                    label={getLabel(item)}
                    enabled={item.enabled}
                    hasHistory={getHasHistory?.(item)}
                    analyticsDate={getAnalyticsDate?.(item)}
                    icon={rowIcon}
                    onToggle={() => onToggle(id, !item.enabled)}
                    saving={savingId === id}
                  />
                );
              })}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
