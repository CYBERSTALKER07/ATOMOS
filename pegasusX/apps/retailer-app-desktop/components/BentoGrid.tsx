"use client";

import { type ReactNode } from "react";
import { motion, AnimatePresence } from "framer-motion";

interface BentoGridProps {
  children: ReactNode;
  className?: string;
  staggerChildren?: number;
}

export function BentoGrid({
  children,
  className = "",
  staggerChildren = 0.04,
}: BentoGridProps) {
  return (
    <motion.div
      layout
      initial="hidden"
      animate="show"
      variants={{
        hidden: {},
        show: {
          transition: {
            staggerChildren,
          },
        },
      }}
      className={`grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 ${className}`}
    >
      <AnimatePresence mode="popLayout">{children}</AnimatePresence>
    </motion.div>
  );
}

type BentoSize = "stat" | "anchor" | "list" | "control" | "wide" | "full";

interface BentoCardProps {
  children: ReactNode;
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
  interactive?: boolean;
  delay?: number;
}

export function BentoCard({
  children,
  size,
  span = 1,
  rowSpan = false,
  className = "",
  interactive = true,
  delay = 0,
}: BentoCardProps) {
  const spanClass =
    span === 4
      ? "lg:col-span-4"
      : span === 3
        ? "lg:col-span-3"
        : span === 2
          ? "lg:col-span-2"
          : "lg:col-span-1";
  const rowSpanClass = rowSpan ? "row-span-2" : "";

  return (
    <motion.div
      layout
      variants={{
        hidden: { opacity: 0, y: 10 },
        show: {
          opacity: 1,
          y: 0,
          transition: {
            type: "spring",
            stiffness: 400,
            damping: 40,
            delay: delay / 1000,
          },
        },
        exit: { opacity: 0, scale: 0.95 },
      }}
      className={`desk-card ${interactive ? "active-press cursor-pointer hover:shadow-md transition-shadow duration-200" : ""} ${spanClass} ${rowSpanClass} ${className} relative overflow-hidden`}
      style={{
        background: "var(--desk-surface)",
        border: "1px solid var(--desk-border)",
        borderRadius: "var(--radius-lg)",
        padding: "var(--desk-card-padding)",
      }}
    >
      <div className="relative h-full w-full z-10">{children}</div>
    </motion.div>
  );
}

interface BentoSkeletonProps {
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
}

export function BentoSkeleton({
  size,
  span = 1,
  rowSpan = false,
  className = "",
}: BentoSkeletonProps) {
  const spanClass =
    span === 4
      ? "lg:col-span-4"
      : span === 3
        ? "lg:col-span-3"
        : span === 2
          ? "lg:col-span-2"
          : "lg:col-span-1";
  const rowSpanClass = rowSpan ? "row-span-2" : "";

  return (
    <div
      className={`animate-pulse ${spanClass} ${rowSpanClass} ${className}`}
      style={{
        background: "var(--desk-surface-subtle)",
        border: "1px solid var(--desk-border)",
        borderRadius: "var(--radius-lg)",
        height: "100%",
        minHeight: "120px",
      }}
    />
  );
}
