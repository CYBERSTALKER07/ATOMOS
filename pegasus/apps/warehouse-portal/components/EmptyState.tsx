import { motion } from "framer-motion";
import { ReactNode, useEffect, useState } from "react";

type EmptyStateVariant =
  | "loading"
  | "no-data"
  | "no-results"
  | "offline"
  | "restricted"
  | "error"
  | "reconnecting"
  | "stale";

interface EmptyStateProps {
  icon?: ReactNode;
  imageUrl?: string;
  headline: string;
  body?: string;
  action?: string;
  onAction?: () => void;
  variant?: EmptyStateVariant;
}

const VARIANT_ASSET_PATHS: Record<EmptyStateVariant, string> = {
  loading: "/illustrations/loading.svg",
  "no-data": "/illustrations/no-data.svg",
  "no-results": "/illustrations/no-results.svg",
  offline: "/illustrations/offline.svg",
  restricted: "/illustrations/restricted.svg",
  error: "/illustrations/error.svg",
  reconnecting: "/illustrations/reconnecting.svg",
  stale: "/illustrations/stale.svg",
};

function resolveVariant(headline: string, body?: string): EmptyStateVariant {
  const text = `${headline} ${body ?? ""}`.toLowerCase();
  if (/loading|fetching|syncing/.test(text)) return "loading";
  if (/reconnecting|restoring live|reconnect/.test(text)) return "reconnecting";
  if (/stale|outdated|degraded/.test(text)) return "stale";
  if (/offline|disconnected|network/.test(text)) return "offline";
  if (/permission|forbidden|access denied|restricted/.test(text))
    return "restricted";
  if (/error|failed|unable|unavailable/.test(text)) return "error";
  if (/search|filter|result/.test(text)) return "no-results";
  return "no-data";
}

function Illustration({
  variant,
  title,
}: {
  variant: EmptyStateVariant;
  title: string;
}) {
  const accent = {
    loading: "var(--desk-info)",
    "no-data": "var(--desk-accent)",
    "no-results": "var(--desk-info)",
    offline: "var(--desk-warning)",
    restricted: "var(--desk-danger)",
    error: "var(--desk-danger)",
    reconnecting: "var(--desk-accent)",
    stale: "var(--desk-text-secondary)",
  }[variant];

  const soft = {
    loading: "var(--desk-info-soft)",
    "no-data": "var(--desk-accent-soft)",
    "no-results": "var(--desk-info-soft)",
    offline: "var(--desk-warning-soft)",
    restricted: "var(--desk-danger-soft)",
    error: "var(--desk-danger-soft)",
    reconnecting: "var(--desk-accent-soft)",
    stale: "var(--desk-surface-subtle)",
  }[variant];

  return (
    <svg
      viewBox="0 0 240 160"
      role="img"
      aria-label={title}
      className="h-24 w-24 md:h-28 md:w-28"
    >
      {/* Container Base */}
      <rect
        x="30"
        y="40"
        width="180"
        height="100"
        rx="12"
        fill="var(--desk-surface-subtle)"
        stroke="var(--desk-border)"
        strokeWidth="2"
      />
      {/* Container Top-Lip (Depth) */}
      <rect
        x="30"
        y="30"
        width="180"
        height="10"
        rx="5"
        fill="var(--desk-surface)"
        stroke="var(--desk-border)"
        strokeWidth="2"
      />

      {/* Content Skeleton */}
      <rect
        x="50"
        y="55"
        width="140"
        height="12"
        rx="6"
        fill="var(--desk-border)"
      />
      <rect
        x="50"
        y="80"
        width="80"
        height="8"
        rx="4"
        fill="var(--desk-border-strong)"
      />
      <rect
        x="50"
        y="96"
        width="60"
        height="8"
        rx="4"
        fill="var(--desk-border)"
      />

      {/* Action Point */}
      <g transform="translate(160, 95)">
        <circle
          cx="20"
          cy="20"
          r="16"
          fill={soft}
          stroke={accent}
          strokeWidth="3"
        />
        {variant === "no-data" ? (
          <path
            d="M10 20h20M20 10v20"
            stroke={accent}
            strokeWidth="4"
            strokeLinecap="round"
          />
        ) : variant === "no-results" ? (
          <>
            <circle cx="18" cy="18" r="6" fill="none" stroke={accent} strokeWidth="3" />
            <path d="M22 22l6 6" stroke={accent} strokeWidth="3" strokeLinecap="round" />
          </>
        ) : variant === "offline" ? (
          <>
            <path d="M11 23c6-6 12-8 18-5" stroke={accent} strokeWidth="3" strokeLinecap="round" />
            <path d="M12 10l16 20" stroke={accent} strokeWidth="3" strokeLinecap="round" />
          </>
        ) : variant === "restricted" ? (
          <>
            <rect x="12" y="18" width="16" height="12" rx="4" fill="none" stroke={accent} strokeWidth="3" />
            <path d="M16 18v-4a4 4 0 0 1 8 0v4" stroke={accent} strokeWidth="3" strokeLinecap="round" />
          </>
        ) : variant === "error" ? (
          <>
            <path d="M20 9l10 18H10L20 9z" fill="none" stroke={accent} strokeWidth="3" strokeLinejoin="round" />
            <path d="M20 15v5" stroke={accent} strokeWidth="3" strokeLinecap="round" />
            <circle cx="20" cy="24" r="1.5" fill={accent} />
          </>
        ) : variant === "loading" ? (
          <>
            <path d="M20 11a9 9 0 1 1-6.5 2.8" fill="none" stroke={accent} strokeWidth="3" strokeLinecap="round" />
            <circle cx="12.5" cy="13.5" r="2" fill={accent} />
          </>
        ) : variant === "reconnecting" ? (
          <>
            <circle cx="13" cy="20" r="3" fill={accent} />
            <circle cx="27" cy="20" r="3" fill={accent} />
            <path d="M17 20h6" stroke={accent} strokeWidth="3" strokeLinecap="round" strokeDasharray="3 4" />
          </>
        ) : variant === "stale" ? (
          <>
            <circle cx="20" cy="20" r="9" fill="none" stroke={accent} strokeWidth="3" />
            <path d="M20 20V14M20 20l4 3" stroke={accent} strokeWidth="3" strokeLinecap="round" />
          </>
        ) : (
          <path
            d="M12 20h16M20 12l8 8-8 8"
            stroke={accent}
            strokeWidth="3"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        )}
      </g>
    </svg>
  );
}

export default function EmptyState({
  icon,
  imageUrl,
  headline,
  body,
  action,
  onAction,
  variant,
}: EmptyStateProps) {
  const resolvedVariant = variant ?? resolveVariant(headline, body);
  const variantImageUrl = VARIANT_ASSET_PATHS[resolvedVariant];
  const assetSrc = imageUrl ?? variantImageUrl;
  const [assetLoadFailed, setAssetLoadFailed] = useState(false);

  useEffect(() => {
    setAssetLoadFailed(false);
  }, [assetSrc]);

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.95 }}
      transition={{ duration: 0.4, ease: "easeOut" }}
      className="flex flex-col items-center justify-center p-8 md:p-16 h-full text-center"
    >
      <motion.div
        initial={{ scale: 0.8, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ delay: 0.1, type: "spring", stiffness: 200, damping: 20 }}
        className={`desk-illustration-frame desk-illustration-frame--${resolvedVariant} mb-6 flex h-32 w-32 items-center justify-center overflow-hidden ring-1 ring-[var(--border)] shadow-sm`}
      >
        {icon ? (
          icon
        ) : assetSrc && !assetLoadFailed ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={assetSrc}
            alt={headline}
            className="h-full w-full object-contain"
            onError={() => setAssetLoadFailed(true)}
          />
        ) : (
          <Illustration variant={resolvedVariant} title={headline} />
        )}
      </motion.div>
      <motion.h3
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.2 }}
        className="md-typescale-title-large font-semibold text-foreground mb-2"
      >
        {headline}
      </motion.h3>
      {body && (
        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.3 }}
          className="md-typescale-body-medium text-muted max-w-sm"
        >
          {body}
        </motion.p>
      )}
      {action && onAction && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
          className="mt-6"
        >
          <button
            type="button"
            onClick={onAction}
            className="desk-btn-primary active-press"
          >
            {action}
          </button>
        </motion.div>
      )}
    </motion.div>
  );
}
