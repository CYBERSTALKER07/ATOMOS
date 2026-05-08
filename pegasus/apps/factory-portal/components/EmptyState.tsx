import { Button } from "@heroui/react";
import { motion } from "framer-motion";
import { ReactNode, useEffect, useState } from "react";

type EmptyStateVariant = "no-data" | "no-results" | "offline" | "restricted" | "error";

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
  if (/permission|forbidden|access denied|restricted/.test(text)) return "restricted";
  if (/error|failed|unable|unavailable/.test(text)) return "error";
  if (/search|filter|result/.test(text)) return "no-results";
  return "no-data";
}

function Illustration({ variant, title }: { variant: EmptyStateVariant; title: string }) {
  return (
    <svg viewBox="0 0 240 160" role="img" aria-label={title} className="h-24 w-24 md:h-28 md:w-28">
      <rect x="22" y="36" width="196" height="102" rx="18" fill="var(--desk-surface-subtle, #f8fafc)" />
      <rect x="34" y="48" width="172" height="14" rx="7" fill="var(--desk-border, #e5e7eb)" />
      <rect x="34" y="70" width="112" height="10" rx="5" fill="var(--desk-border-strong, #cbd5e1)" />
      <rect x="34" y="86" width="88" height="10" rx="5" fill="var(--desk-border, #e5e7eb)" />

      {variant === "no-results" && (
        <>
          <circle cx="170" cy="92" r="18" fill="none" stroke="var(--desk-accent, #ff7a1a)" strokeWidth="8" />
          <line x1="183" y1="105" x2="198" y2="120" stroke="var(--desk-accent, #ff7a1a)" strokeWidth="8" strokeLinecap="round" />
        </>
      )}

      {variant === "offline" && (
        <>
          <circle cx="170" cy="92" r="18" fill="var(--desk-warning, #d97706)" opacity="0.16" />
          <path d="M156 96c8-8 20-8 28 0" fill="none" stroke="var(--desk-warning, #d97706)" strokeWidth="6" strokeLinecap="round" />
          <line x1="156" y1="84" x2="184" y2="112" stroke="var(--desk-warning, #d97706)" strokeWidth="6" strokeLinecap="round" />
        </>
      )}

      {variant === "restricted" && (
        <>
          <path d="M170 74l18 8v14c0 13-8 22-18 27-10-5-18-14-18-27V82l18-8z" fill="var(--desk-info, #2563eb)" opacity="0.18" />
          <rect x="162" y="90" width="16" height="14" rx="3" fill="none" stroke="var(--desk-info, #2563eb)" strokeWidth="5" />
          <path d="M165 90v-4a5 5 0 0110 0v4" fill="none" stroke="var(--desk-info, #2563eb)" strokeWidth="5" strokeLinecap="round" />
        </>
      )}

      {variant === "error" && (
        <>
          <polygon points="170,72 196,118 144,118" fill="var(--desk-danger, #dc2626)" opacity="0.18" />
          <line x1="170" y1="86" x2="170" y2="102" stroke="var(--desk-danger, #dc2626)" strokeWidth="6" strokeLinecap="round" />
          <circle cx="170" cy="110" r="3.8" fill="var(--desk-danger, #dc2626)" />
        </>
      )}

      {variant === "no-data" && (
        <>
          <circle cx="170" cy="92" r="20" fill="var(--desk-accent, #ff7a1a)" opacity="0.16" />
          <rect x="158" y="84" width="24" height="16" rx="6" fill="none" stroke="var(--desk-accent, #ff7a1a)" strokeWidth="5" />
        </>
      )}
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
  const variantImageUrl = `/illustrations/${resolvedVariant}.svg`;
  const [assetLoadFailed, setAssetLoadFailed] = useState(false);

  useEffect(() => {
    setAssetLoadFailed(false);
  }, [variantImageUrl]);

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
        className="w-32 h-32 flex items-center justify-center mb-6 overflow-hidden shadow-sm ring-1 ring-[var(--border)] desk-illustration-frame"
      >
        {imageUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={imageUrl} alt={headline} className="w-full h-full object-cover" />
        ) : icon ? (
          icon
        ) : !assetLoadFailed ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={variantImageUrl}
            alt={headline}
            className="w-full h-full object-cover"
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
          <Button 
            variant="flat" 
            color="primary"
            onPress={onAction}
            className="font-medium hover-lift active-press"
          >
            {action}
          </Button>
        </motion.div>
      )}
    </motion.div>
  );
}
