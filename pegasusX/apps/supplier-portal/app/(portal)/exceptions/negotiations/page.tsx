"use client";

import Link from "next/link";
import type { Route } from "next";
import { PageChrome } from '@/components/PageChrome';

/** Quantity negotiation is disabled across the ecosystem. */
export default function NegotiationsExceptionsPage() {
  return (
    <PageChrome
      icon="warning"
      title="Quantity negotiations"
      description="This feature is not available."
      loading={false}
      error={null}
      empty={false}
    >
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        Quantity negotiation has been disabled. Use the exceptions queue for shop-closed and delivery escalations.
      </p>
      <Link href={"/exceptions" as Route} className="mt-4 inline-block text-[var(--color-md-primary)] underline">
        Back to exceptions
      </Link>
    </PageChrome>
  );
}
