"use client";

import React from "react";
import { motion, HTMLMotionProps } from "framer-motion";

export interface GlassmorphismPanelProps extends HTMLMotionProps<"div"> {
  children: React.ReactNode;
  className?: string;
}

export function GlassmorphismPanel({
  children,
  className = "",
  ...props
}: GlassmorphismPanelProps) {
  return (
    <motion.div
      className={`bg-black/40 backdrop-blur-md border border-white/10 rounded-xl p-4 shadow-2xl ${className}`}
      {...props}
    >
      {children}
    </motion.div>
  );
}
