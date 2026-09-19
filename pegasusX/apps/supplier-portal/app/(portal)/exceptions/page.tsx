"use client";

import { usePortalT } from "@/lib/i18n";
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
  const t = usePortalT();
  const [exceptions, setExceptions] = useState<SupplierExceptionRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierExceptions()
      .then((resp) => setExceptions(resp.exceptions))
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_exceptions_failed")))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PageChrome
      title={t("portal.nav.exceptions")}
      description={t("supplier_portal.residual.text.shop_closed_payment_and_delivery_escalation_queues")}
      icon="warning"
      loading={loading}
      error={error}
      empty={!loading && exceptions.length === 0}
      emptyMessage={t("supplier_portal.residual.text.no_open_exceptions_escalations_appear_here_when_operators_raise_")}
    >
      <section className="mb-6">
        <h2 className="mb-2 md-typescale-title-medium">{t("supplier_portal.exceptions.text.recommended_playbooks")}</h2>
        <PlaybookRunsPanel />
      </section>
      <ExceptionsList
        exceptions={exceptions}
        onResolved={() => {
          api.getSupplierExceptions().then((resp) => setExceptions(resp.exceptions));
        }}
      />
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
