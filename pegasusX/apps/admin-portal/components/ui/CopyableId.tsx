"use client";

import React, { useState } from "react";

interface CopyableIdProps {
  value?: string;
  id?: string;
  label?: string;
  truncateLength?: number;
  className?: string;
}

export function CopyableId({ value, id, label, truncateLength = 10, className = "" }: CopyableIdProps) {
  const [copied, setCopied] = useState(false);
  const targetVal = value ?? id ?? "";

  if (!targetVal) return <span className="text-gray-400 font-mono">—</span>;

  const display =
    truncateLength > 0 && targetVal.length > truncateLength
      ? `${targetVal.slice(0, truncateLength)}…`
      : targetVal;

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(targetVal);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard write failed
    }
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      title={`Click to copy ${label ? `${label}: ` : ""}${targetVal}`}
      className={`inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs font-mono transition-colors hover:bg-gray-100 dark:hover:bg-gray-800 ${className}`}
    >
      <span className="select-all">{display}</span>
      {copied ? (
        <span className="text-[10px] font-sans font-semibold text-emerald-600 dark:text-emerald-400">
          ✓ Copied
        </span>
      ) : (
        <svg
          className="h-3 w-3 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
          />
        </svg>
      )}
    </button>
  );
}
