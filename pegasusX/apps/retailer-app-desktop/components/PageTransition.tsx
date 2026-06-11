"use client";

import { motion, AnimatePresence, useReducedMotion } from "framer-motion";
import { duration, easing, reducedMotionTransition } from "@pegasusx/motion-tokens";
import { ReactNode } from "react";
import { usePathname } from "next/navigation";

interface PageTransitionProps {
  children: ReactNode;
  className?: string;
}

export default function PageTransition({
  children,
  className = "",
}: PageTransitionProps) {
  const pathname = usePathname();
  const shouldReduceMotion = useReducedMotion();

  // Calm, high-fidelity transition: subtle fade + layout shift
  return (
    <AnimatePresence mode="popLayout">
      <motion.div
        key={pathname}
        initial={shouldReduceMotion ? { opacity: 1 } : { opacity: 0, y: 4 }}
        animate={{ opacity: 1, y: 0 }}
        exit={shouldReduceMotion ? { opacity: 1 } : { opacity: 0, y: -4 }}
        transition={reducedMotionTransition(!!shouldReduceMotion, {
          duration: duration.short4,
          ease: easing.standard,
        })}
        className={`w-full min-h-full ${className}`}
      >
        {children}
      </motion.div>
    </AnimatePresence>
  );
}
