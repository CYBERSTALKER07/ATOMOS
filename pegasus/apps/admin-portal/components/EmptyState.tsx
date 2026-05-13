import { motion, useReducedMotion } from "framer-motion";
import { ReactNode } from "react";
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

function resolveAccent(variant: EmptyStateVariant) {
  switch (variant) {
    case "offline":
    case "no-predictions":
      return "var(--desk-info)";
    case "restricted":
      return "var(--desk-warning)";
    case "error":
      return "var(--desk-danger)";
    case "production":
    case "no-products":
      return "var(--desk-success)";
    default:
      return "var(--desk-accent)";
  }
}

function VariantGlyph({
  variant,
  accent,
}: {
  variant: EmptyStateVariant;
  accent: string;
}) {
  const stroke = {
    stroke: accent,
    strokeWidth: 2.8,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    fill: "none",
  };

  switch (variant) {
    case "no-results":
      return (
        <>
          <circle cx="18" cy="18" r="8" {...stroke} />
          <path d="M24 24L32 32" {...stroke} />
          <path d="M8 30L14 24" {...stroke} />
        </>
      );
    case "offline":
      return (
        <>
          <path d="M8 15C12 11 28 11 32 15" {...stroke} />
          <path d="M12 21C15 18 25 18 28 21" {...stroke} />
          <path d="M17 27C19 25 21 25 23 27" {...stroke} />
          <path d="M10 32L31 11" {...stroke} />
          <circle cx="20" cy="31" r="2" fill={accent} />
        </>
      );
    case "restricted":
      return (
        <>
          <path d="M14 19V15C14 11.7 16.7 9 20 9C23.3 9 26 11.7 26 15V19" {...stroke} />
          <rect x="12" y="19" width="16" height="14" rx="4" {...stroke} />
          <path d="M20 24V28" {...stroke} />
          <circle cx="20" cy="23" r="1.5" fill={accent} />
        </>
      );
    case "error":
      return (
        <>
          <path d="M20 8L32 31H8L20 8Z" {...stroke} />
          <path d="M20 17V23" {...stroke} />
          <circle cx="20" cy="27" r="1.7" fill={accent} />
        </>
      );
    case "production":
      return (
        <>
          <path d="M10 31V18L18 22V15L27 20V31H10Z" {...stroke} />
          <path d="M14 31V25" {...stroke} />
          <path d="M19 31V27" {...stroke} />
          <path d="M24 31V24" {...stroke} />
          <path d="M14 15V10H18V17" {...stroke} />
        </>
      );
    case "no-orders":
      return (
        <>
          <rect x="11" y="10" width="18" height="24" rx="4" {...stroke} />
          <path d="M16 10.5H24" {...stroke} />
          <path d="M15 18H25" {...stroke} />
          <path d="M15 23H25" {...stroke} />
          <path d="M15 28H22" {...stroke} />
        </>
      );
    case "no-products":
      return (
        <>
          <path d="M10 16L20 11L30 16L20 21L10 16Z" {...stroke} />
          <path d="M10 16V27L20 33L30 27V16" {...stroke} />
          <path d="M20 21V33" {...stroke} />
        </>
      );
    case "no-predictions":
      return (
        <>
          <path d="M10 31H31" {...stroke} />
          <path d="M10 31V11" {...stroke} />
          <path d="M13 26L18 20L23 22L30 14" {...stroke} />
          <circle cx="18" cy="20" r="1.7" fill={accent} />
          <circle cx="23" cy="22" r="1.7" fill={accent} />
          <circle cx="30" cy="14" r="1.7" fill={accent} />
        </>
      );
    case "no-suppliers":
      return (
        <>
          <circle cx="12" cy="18" r="4" {...stroke} />
          <circle cx="28" cy="18" r="4" {...stroke} />
          <circle cx="20" cy="28" r="4" {...stroke} />
          <path d="M15 20L17.5 25" {...stroke} />
          <path d="M25 20L22.5 25" {...stroke} />
          <path d="M16 18H24" {...stroke} />
        </>
      );
    case "no-data":
    default:
      return (
        <>
          <path d="M20 10V30" {...stroke} />
          <path d="M10 20H30" {...stroke} />
        </>
      );
  }
}

function Illustration({
  variant,
  title,
  shouldReduceMotion,
}: {
  variant: EmptyStateVariant;
  title: string;
  shouldReduceMotion: boolean;
}) {
  const accent = resolveAccent(variant);
  const floatingTransition = shouldReduceMotion
    ? { duration: 0.01 }
    : {
        duration: 2.8,
        repeat: Infinity,
        repeatType: "reverse" as const,
        ease: "easeInOut" as const,
      };
  const pulseTransition = shouldReduceMotion
    ? { duration: 0.01 }
    : {
        duration: 2.2,
        repeat: Infinity,
        ease: "easeInOut" as const,
      };

  return (
    <svg
      viewBox="0 0 240 180"
      role="img"
      aria-label={title}
      className="h-28 w-28 md:h-32 md:w-32"
    >
      <motion.path
        d="M36 150C67 128 101 120 138 126C171 132 194 126 212 112"
        fill="none"
        stroke={accent}
        strokeOpacity="0.18"
        strokeWidth="3"
        strokeLinecap="round"
        initial={false}
        animate={
          shouldReduceMotion
            ? { opacity: 0.18, pathLength: 1 }
            : { opacity: [0.08, 0.22, 0.08], pathLength: [0.7, 1, 0.7] }
        }
        transition={{ duration: 3.2, repeat: Infinity, ease: "easeInOut" }}
      />
      <motion.path
        d="M40 58C72 36 108 31 149 39C177 44 196 41 212 28"
        fill="none"
        stroke={accent}
        strokeOpacity="0.12"
        strokeWidth="2"
        strokeLinecap="round"
        initial={false}
        animate={
          shouldReduceMotion
            ? { opacity: 0.12, pathLength: 1 }
            : { opacity: [0.05, 0.16, 0.05], pathLength: [0.55, 0.95, 0.55] }
        }
        transition={{ duration: 2.7, repeat: Infinity, ease: "easeInOut" }}
      />

      <motion.g
        initial={false}
        animate={shouldReduceMotion ? { y: 0 } : { y: [0, -4, 0] }}
        transition={floatingTransition}
      >
        <rect
          x="36"
          y="48"
          width="168"
          height="98"
          rx="22"
          fill="var(--desk-surface)"
          stroke="var(--desk-border)"
          strokeWidth="2"
        />
        <rect
          x="36"
          y="48"
          width="168"
          height="18"
          rx="22"
          fill="var(--desk-surface-subtle)"
        />
        <motion.rect
          x="54"
          y="78"
          width="92"
          height="10"
          rx="5"
          fill="var(--desk-border-strong)"
          initial={false}
          animate={shouldReduceMotion ? { opacity: 0.85 } : { opacity: [0.55, 1, 0.55] }}
          transition={pulseTransition}
        />
        <motion.rect
          x="54"
          y="97"
          width="118"
          height="8"
          rx="4"
          fill="var(--desk-border)"
          initial={false}
          animate={shouldReduceMotion ? { opacity: 0.75 } : { opacity: [0.35, 0.75, 0.35] }}
          transition={{ ...pulseTransition, delay: 0.12 }}
        />
        <motion.rect
          x="54"
          y="112"
          width="84"
          height="8"
          rx="4"
          fill="var(--desk-border)"
          initial={false}
          animate={shouldReduceMotion ? { opacity: 0.6 } : { opacity: [0.28, 0.65, 0.28] }}
          transition={{ ...pulseTransition, delay: 0.24 }}
        />
        <rect
          x="54"
          y="127"
          width="58"
          height="11"
          rx="5.5"
          fill="var(--desk-surface-subtle)"
          stroke="var(--desk-border)"
          strokeWidth="1.5"
        />
      </motion.g>

      <g transform="translate(138 18)">
        <motion.circle
          cx="42"
          cy="42"
          r="33"
          fill="var(--desk-surface-subtle)"
          stroke={accent}
          strokeWidth="3"
          initial={false}
          animate={
            shouldReduceMotion
              ? { scale: 1, rotate: 0 }
              : { scale: [1, 1.04, 1], rotate: [0, -2, 0] }
          }
          transition={{ duration: 2.6, repeat: Infinity, ease: "easeInOut" }}
        />
        <motion.circle
          cx="42"
          cy="42"
          r="41"
          fill="none"
          stroke={accent}
          strokeOpacity="0.16"
          strokeWidth="2"
          initial={false}
          animate={
            shouldReduceMotion
              ? { scale: 1, opacity: 0.16 }
              : { scale: [0.96, 1.04, 0.96], opacity: [0.08, 0.24, 0.08] }
          }
          transition={{ duration: 2.8, repeat: Infinity, ease: "easeInOut" }}
        />
        <g transform="translate(22 22)">
          <VariantGlyph variant={variant} accent={accent} />
        </g>
      </g>

      <g transform="translate(62 30)">
        {[0, 1, 2].map((index) => (
          <motion.circle
            key={index}
            cx={index * 12}
            cy="0"
            r="3"
            fill={accent}
            initial={false}
            animate={
              shouldReduceMotion
                ? { opacity: 0.7, scale: 1 }
                : { opacity: [0.35, 1, 0.35], scale: [1, 1.22, 1] }
            }
            transition={{ duration: 1.9, repeat: Infinity, delay: index * 0.14 }}
          />
        ))}
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
  const shouldReduceMotion = useReducedMotion() ?? false;

  return (
    <motion.div
      initial={shouldReduceMotion ? { opacity: 1 } : { opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={shouldReduceMotion ? { opacity: 1 } : { opacity: 0 }}
      transition={{ duration: shouldReduceMotion ? 0.01 : 0.22, ease: [0.2, 0, 0, 1] }}
      className="flex flex-col items-center justify-center p-8 md:p-16 h-full text-center"
    >
      <motion.div
        initial={shouldReduceMotion ? { opacity: 1, scale: 1 } : { scale: 0.94, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{
          opacity: { duration: shouldReduceMotion ? 0.01 : 0.32 },
          scale: shouldReduceMotion
            ? { duration: 0.01 }
            : { type: "spring", stiffness: 200, damping: 24 },
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
          ) : (
            <Illustration
              variant={resolvedVariant}
              title={headline}
              shouldReduceMotion={shouldReduceMotion}
            />
          )}
        </div>
      </motion.div>

      <motion.div
        initial={shouldReduceMotion ? { opacity: 1, y: 0 } : { opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: shouldReduceMotion ? 0 : 0.14, duration: shouldReduceMotion ? 0.01 : 0.24 }}
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
          initial={shouldReduceMotion ? { opacity: 1, scale: 1 } : { opacity: 0, scale: 0.94 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={
            shouldReduceMotion
              ? { duration: 0.01 }
              : { delay: 0.28, type: "spring", stiffness: 240, damping: 24 }
          }
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
