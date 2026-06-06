"use client";

import Link from "next/link";
import { PortalSurface } from "../_components/PortalSurface";

export default function CatalogPage() {
  return (
    <PortalSurface
      title="Catalog"
      description="SKU catalog is supplier-scoped and aligned with inventory and pricing."
    >
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        Live SKU rows are served from{" "}
        <Link href="/inventory" className="text-[var(--color-md-primary)] underline">
          inventory
        </Link>
        . Markup authority is on{" "}
        <Link href="/pricing" className="text-[var(--color-md-primary)] underline">
          pricing
        </Link>
        .
      </p>
    </PortalSurface>
  );
}
