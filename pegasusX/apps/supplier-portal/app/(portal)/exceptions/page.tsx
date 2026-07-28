"use client";

import Link from "next/link";
import type { Route } from "next";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierExceptionRow } from "@pegasusx/types";
import StatusBadge from "@/components/StatusBadge";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function ExceptionsPage() {
  const [exceptions, setExceptions] = useState<SupplierExceptionRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierExceptions()
      .then((resp) => setExceptions(resp.exceptions))
      .catch((err) => setError(err instanceof Error ? err.message : "load_exceptions_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PageChrome
      title="Exceptions"
      description="Shop-closed, payment, and delivery escalation queues."
      icon="warning"
      loading={loading}
      error={error}
      empty={!loading && exceptions.length === 0}
      emptyMessage="No open exceptions. Escalations appear here when operators raise them."
    >
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {exceptions.map((row) => (
          <li key={row.order_id} className="p-4 md-typescale-body-medium">
            <div className="flex flex-wrap items-center gap-2">
              <span className="md-chip h-6 text-xs">{row.kind}</span>
              <span className="font-mono text-[var(--color-md-primary)]">{row.order_id}</span>
              <StatusBadge state={row.status} />
            </div>
            {row.note ? <p className="mt-2 text-[var(--color-md-outline)]">{row.note}</p> : null}
            <p className="mt-1 text-sm text-[var(--color-md-outline)]">
              Updated {new Date(row.updated_at).toLocaleString()}
            </p>
          </li>
        ))}
      </ul>
      <div className="flex flex-wrap gap-4 md-typescale-body-medium">
        <Link href={"/exceptions/claims" as Route} className="text-[var(--color-md-primary)] underline">
          Claims / chargebacks
        </Link>
        <Link href={"/exceptions/shop-closed" as Route} className="text-[var(--color-md-primary)] underline">
          Shop closed queue
        </Link>
        <Link href={"/exceptions/early-complete" as Route} className="text-[var(--color-md-primary)] underline">
          Early route complete
        </Link>
        <Link href="/manifest-exceptions" className="text-[var(--color-md-primary)] underline">
          Manifest gate exceptions
        </Link>
        <Link href={"/operations" as Route} className="text-[var(--color-md-primary)] underline">
          Operations
        </Link>
        <Link href="/orders" className="text-[var(--color-md-primary)] underline">
          Orders
        </Link>
      </div>
    </PageChrome>
  );
}
