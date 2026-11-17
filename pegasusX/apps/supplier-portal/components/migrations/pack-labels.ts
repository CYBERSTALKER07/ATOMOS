import type { MigrationPack } from "@pegasusx/types";

export const ALL_MIGRATION_PACKS: readonly MigrationPack[] = [
  "warehouses",
  "products_prices",
  "customers",
  "stock",
  "lots",
  "historical_orders",
  "nasiya_ar",
  "open_orders",
];

export const ENCODING_OPTIONS = [
  "utf-8",
  "windows-1251",
  "cp1251",
  "cp866",
] as const;

export function packLabelKey(pack: MigrationPack | string): string {
  switch (pack) {
    case "warehouses":
      return "supplier_portal.migrations.warehouse_label";
    case "products_prices":
      return "supplier_portal.migrations.pack_products";
    case "customers":
      return "supplier_portal.migrations.pack_retailers";
    case "stock":
      return "supplier_portal.migrations.pack_stock";
    case "lots":
      return "supplier_portal.migrations.pack_lots";
    case "nasiya_ar":
      return "supplier_portal.migrations.pack_nasiya";
    case "open_orders":
      return "supplier_portal.migrations.pack_orders_open";
    case "historical_orders":
      return "supplier_portal.migrations.pack_orders_history";
    default:
      return "supplier_portal.migrations.pack_quarantine";
  }
}

export function domainQueryToPack(domain: string | null): MigrationPack | null {
  switch ((domain || "").trim().toLowerCase()) {
    case "products":
    case "products_prices":
      return "products_prices";
    case "customers":
    case "retailers":
      return "customers";
    case "stock":
    case "inventory":
      return "stock";
    case "lots":
      return "lots";
    case "nasiya":
    case "nasiya_ar":
      return "nasiya_ar";
    case "open_orders":
    case "orders_open":
      return "open_orders";
    case "historical_orders":
    case "orders_history":
      return "historical_orders";
    case "warehouses":
      return "warehouses";
    default:
      return null;
  }
}
