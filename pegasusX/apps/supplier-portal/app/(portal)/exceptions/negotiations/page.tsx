"use client";

import Link from "next/link";
import type { Route } from "next";
import { PageChrome } from "@/components/PageChrome";

/**
 * Quantity negotiation is product-disabled ecosystem-wide.
 * Delivery-time driver propose → supplier resolve is gated off (410 / empty list).
 * Not a substitute for claims, shop-closed, missing-items, or partial offload.
 */
export default function NegotiationsExceptionsPage() {
  return (
    <PageChrome
      icon="handshake"
      title="Quantity negotiations"
      description="Delivery-time line qty propose/resolve (driver → supplier)."
      empty
      emptyMessage="Quantity negotiation is disabled. Use shop-closed, claims, or missing-items for delivery exceptions."
    >
      <div className="max-w-xl space-y-4 md-typescale-body-medium text-[var(--color-md-on-surface)]">
        <p className="text-[var(--color-md-outline)]">
          This feature changes order line quantities during delivery (before settlement).
          It is intentionally off and does not replace other exception paths.
        </p>
        <ul className="list-disc pl-5 space-y-1 text-[var(--color-md-outline)]">
          <li>
            <Link href={"/exceptions/shop-closed" as Route} className="text-[var(--color-md-primary)] underline">
              Shop closed
            </Link>{" "}
            — retailer unavailable at stop
          </li>
          <li>
            <Link href={"/exceptions/claims" as Route} className="text-[var(--color-md-primary)] underline">
              Claims
            </Link>{" "}
            — post-delivery damage / shortage / OS&amp;D
          </li>
          <li>
            <Link href={"/exceptions" as Route} className="text-[var(--color-md-primary)] underline">
              All exceptions
            </Link>
          </li>
        </ul>
      </div>
    </PageChrome>
  );
}
