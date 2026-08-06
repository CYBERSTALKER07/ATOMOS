"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { PageChrome } from "@/components/PageChrome";

/**
 * Quantity negotiation is product-disabled ecosystem-wide.
 * Delivery-time driver propose → supplier resolve is gated off (410 / empty list).
 * Not a substitute for claims, shop-closed, missing-items, or partial offload.
 */
export default function NegotiationsExceptionsPage() {
  const t = usePortalT();
  return (
    <PageChrome
      icon="handshake"
      title={t("supplier_portal.exceptions.negotiations.text.quantity_negotiations")}
      description={t("supplier_portal.residual.text.delivery_time_line_qty_propose_resolve_driver_supplier")}
      empty
      emptyMessage={t("supplier_portal.residual.text.quantity_negotiation_is_disabled_use_shop_closed_claims_or_missi")}
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
