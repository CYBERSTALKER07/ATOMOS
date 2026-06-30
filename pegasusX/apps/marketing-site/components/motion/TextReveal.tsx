"use client";

import { type ReactNode } from "react";
import { useTextReveal } from "./useTextReveal";
import { useRef } from "react";

type TextRevealProps = {
  children: ReactNode;
  className?: string;
  as?: "div" | "section" | "h1" | "h2" | "p";
  split?: "lines" | "chars";
};

export function TextReveal({
  children,
  className = "",
  as: Tag = "div",
  split = "lines",
}: TextRevealProps) {
  const ref = useRef<HTMLElement>(null);
  useTextReveal(ref, { split });

  return (
    <Tag ref={ref as never} className={className}>
      {children}
    </Tag>
  );
}

export function RevealLine({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <span data-reveal-line className={`block ${className}`.trim()}>
      {children}
    </span>
  );
}
