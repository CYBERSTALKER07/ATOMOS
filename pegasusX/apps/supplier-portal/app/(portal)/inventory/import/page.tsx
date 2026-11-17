"use client";

import Link from "next/link";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { usePortalT } from "@/lib/i18n";
import { PageChrome } from "@/components/PageChrome";

const DOMAIN_SHORTCUTS = [
  { domain: "products_prices", labelKey: "supplier_portal.migrations.pack_products" },
  { domain: "stock", labelKey: "supplier_portal.migrations.pack_stock" },
  { domain: "lots", labelKey: "supplier_portal.migrations.pack_lots" },
  { domain: "customers", labelKey: "supplier_portal.migrations.pack_retailers" },
  { domain: "nasiya_ar", labelKey: "supplier_portal.migrations.pack_nasiya" },
  { domain: "open_orders", labelKey: "supplier_portal.migrations.pack_orders_open" },
] as const;

export default function InventoryImportRedirectPage() {
  const t = usePortalT();
  const router = useRouter();

  useEffect(() => {
    const timer = setTimeout(() => {
      router.replace("/migrations?domain=stock");
    }, 300);
    return () => clearTimeout(timer);
  }, [router]);

  return (
    <PageChrome
      icon="inventory"
      title={t("supplier_portal.migrations.title")}
      description={t("supplier_portal.migrations.catalog_redirect_notice")}
    >
      <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-6 flex flex-col gap-4">
        <p className="text-sm text-[var(--muted)]">
          {t("supplier_portal.migrations.catalog_redirect_notice")}
        </p>
        <div className="flex flex-wrap gap-2">
          <Link href="/migrations?domain=stock" className="md-btn md-btn-filled px-4 py-2 text-sm">
            {t("supplier_portal.migrations.title")}
          </Link>
          {DOMAIN_SHORTCUTS.map((d) => (
            <Link
              key={d.domain}
              href={`/migrations?domain=${d.domain}`}
              className="md-btn md-btn-outlined px-3 py-2 text-xs"
            >
              {t(d.labelKey)}
            </Link>
          ))}
        </div>
      </div>
    </PageChrome>
  );
}
