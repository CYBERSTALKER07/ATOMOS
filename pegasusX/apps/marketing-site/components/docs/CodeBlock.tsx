"use client";

import { useState } from "react";

type CodeBlockProps = {
  code: string;
};

export function CodeBlock({ code }: CodeBlockProps) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="relative rounded-xl border border-[var(--mkt-border)] bg-black/40">
      <button
        type="button"
        onClick={copy}
        className="absolute right-3 top-3 rounded-md border border-[var(--mkt-border)] px-2 py-1 text-xs"
      >
        {copied ? "Copied" : "Copy"}
      </button>
      <pre className="overflow-x-auto p-4 pt-10 font-mono text-xs leading-relaxed text-[var(--mkt-text-secondary)]">
        <code>{code}</code>
      </pre>
    </div>
  );
}
