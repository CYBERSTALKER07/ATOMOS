import { motion } from "framer-motion";
import { ReactNode, useEffect, useState } from "react";
import Icon from "./Icon";

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
  icon?: ReactNode | string;
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
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="flex flex-col items-center justify-center p-8 md:p-16 h-full text-center"
    >
      <motion.div
        initial={{ scale: 0.9, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{
          opacity: { duration: 0.4 },
          scale: { type: "spring", stiffness: 200, damping: 25 },
        }}
        className="relative w-56 h-56 flex items-center justify-center mb-8"
      >
        <div className="relative z-10 w-full h-full flex items-center justify-center overflow-hidden desk-illustration-frame rounded-3xl">
          {imageUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={imageUrl}
              alt={headline}
              className="w-full h-full object-contain p-6"
            />
          ) : icon ? (
            typeof icon === "string" ? (
              <div
                className="flex items-center justify-center w-20 h-20"
                style={{ color: "var(--desk-accent)" }}
              >
                <Icon name={icon} size={48} />
              </div>
            ) : (
              <div
                className="w-20 h-20"
                style={{ color: "var(--desk-accent)" }}
              >
                {icon}
              </div>
            )
          ) : assetType === "png" ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={pngUrl}
              alt={headline}
              className="w-full h-full object-contain p-4"
              onError={() => setAssetType("svg")}
            />
          ) : assetType === "svg" ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={svgUrl}
              alt={headline}
              className="w-full h-full object-contain p-6"
              onError={() => setAssetType("none")}
            />
          ) : (
            <Illustration variant={resolvedVariant} title={headline} />
          )}
        </div>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.2 }}
        className="max-w-md space-y-3"
      >
        <h3 className="md-typescale-headline-small font-bold tracking-tight text-foreground">
          {headline}
        </h3>
        {body && (
          <p className="md-typescale-body-large text-muted-foreground leading-relaxed">
            {body}
          </p>
        )}
      </motion.div>

      {action && onAction && (
        <motion.div
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.4, type: "spring" }}
          className="mt-8"
        >
          <button type="button" onClick={onAction} className="desk-btn-primary">
            {action}
          </button>
        </motion.div>
      )}
    </motion.div>
  );
}
