import { motion } from "framer-motion";
import { ReactNode, useEffect, useState } from "react";

type EmptyStateVariant =
  | "no-data"
  | "no-results"
  | "offline"
  | "restricted"
  | "error"
  | "production"
  | "no-orders"
  | "no-products"
  | "no-predictions"
  | "no-suppliers";

interface EmptyStateProps {
  icon?: ReactNode;
  imageUrl?: string;
  headline: string;
  body?: string;
  action?: string;
  onAction?: () => void;
  variant?: EmptyStateVariant;
}

function resolveVariant(headline: string, body?: string): EmptyStateVariant {
  const text = `${headline} ${body ?? ""}`.toLowerCase();
  if (/offline|disconnected|network/.test(text)) return "offline";
  if (/permission|forbidden|access denied|restricted/.test(text))
    return "restricted";
  if (/error|failed|unable|unavailable/.test(text)) return "error";
  if (/search|filter|result/.test(text)) return "no-results";
  if (/order/.test(text)) return "no-orders";
  if (/product/.test(text)) return "no-products";
  if (/prediction|insight/.test(text)) return "no-predictions";
  if (/supplier/.test(text)) return "no-suppliers";
  if (/production|factory|line/.test(text)) return "production";
  return "no-data";
}

function Illustration({
  variant,
  title,
}: {
  variant: EmptyStateVariant;
  title: string;
}) {
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
          fill="var(--desk-surface)"
          stroke="var(--desk-accent)"
          strokeWidth="3"
        />
        {variant === "no-data" ? (
          <path
            d="M10 20h20M20 10v20"
            stroke="var(--desk-accent)"
            strokeWidth="4"
            strokeLinecap="round"
          />
        ) : (
          <path
            d="M12 20h16M20 12l8 8-8 8"
            stroke="var(--desk-accent)"
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
  const pngUrl = `/illustrations/${resolvedVariant}.png`;
  const svgUrl = `/illustrations/${resolvedVariant}.svg`;
  const [assetType, setAssetType] = useState<"png" | "svg" | "none">("png");

  useEffect(() => {
    setAssetType("png");
  }, [resolvedVariant]);

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
        className="w-48 h-48 flex items-center justify-center mb-6 overflow-hidden desk-illustration-frame"
      >
        {imageUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={imageUrl}
            alt={headline}
            className="w-full h-full object-contain p-4"
          />
        ) : icon ? (
          <div className="w-16 h-16" style={{ color: "var(--desk-accent)" }}>
            {icon}
          </div>
        ) : assetType === "png" ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={pngUrl}
            alt={headline}
            className="w-full h-full object-contain p-2"
            onError={() => setAssetType("svg")}
          />
        ) : assetType === "svg" ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={svgUrl}
            alt={headline}
            className="w-full h-full object-contain p-4"
            onError={() => setAssetType("none")}
          />
        ) : (
          <Illustration variant={resolvedVariant} title={headline} />
        )}
      </motion.div>
      <motion.h3
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.2 }}
        className="md-typescale-title-large font-light text-foreground mb-2"
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
