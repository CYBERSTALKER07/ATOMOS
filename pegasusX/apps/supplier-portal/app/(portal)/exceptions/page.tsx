"use client";

import Link from "next/link";
import type { Route } from "next";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierExceptionRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { ExceptionsList } from "@/components/exceptions/ExceptionsList";
import { PlaybookRunsPanel } from "@/components/exceptions/PlaybookRunsPanel";

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
      <section className="mb-6">
        <h2 className="mb-2 md-typescale-title-medium">Recommended playbooks</h2>
        <PlaybookRunsPanel />
      </section>
      <ExceptionsList exceptions={exceptions} />
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
        <Link href="/treasury/cash-reconciliations" className="text-[var(--color-md-primary)] underline">
          Cash discrepancies
        </Link>
        <Link href="/finance/credit-notes" className="text-[var(--color-md-primary)] underline">
          Credit notes
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
