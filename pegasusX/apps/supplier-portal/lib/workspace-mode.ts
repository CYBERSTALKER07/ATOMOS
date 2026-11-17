import type { NavSection } from "@/lib/nav";

export type SupplierWorkspaceMode = "fast_desk" | "full_erp";

export const WORKSPACE_MODE_STORAGE_KEY = "pegasus_supplier_workspace_mode";

const FAST_DESK_ROLE_MARKERS = [
  "EXCEPTION_LEAD",
  "FINANCE_CLOSER",
  "FINANCE",
  "COLLECTIONS",
  "CONTROLLER",
  "OPS_LEAD",
  "ACCOUNTANT",
];

const FAST_DESK_HREFS = [
  "/exceptions",
  "/credit/collections",
  "/orders",
  "/treasury/cash-reconciliations",
  "/manifests",
  "/dashboard",
];

/** Infers default workspace mode from JWT claims so Exception Leads & Finance Closers open in Fast Desk. */
export function resolveDefaultWorkspaceMode(claims: Record<string, unknown> | null): SupplierWorkspaceMode {
  if (!claims) return "full_erp";
  const candidates = [
    claims.workspace_role,
    claims.job_role,
    claims.desk_role,
    claims.role,
    claims.workspace_mode,
  ]
    .filter((v): v is string => typeof v === "string")
    .map((v) => v.toUpperCase().trim());

  if (candidates.some((c) => c === "FAST_DESK")) return "fast_desk";
  if (candidates.some((c) => FAST_DESK_ROLE_MARKERS.some((m) => c.includes(m)))) {
    return "fast_desk";
  }
  return "full_erp";
}

/** Builds a streamlined Fast Desk sidebar view (Exceptions, Collections, Orders up front + full nav collapsed). */
export function buildNavForWorkspaceMode(nav: NavSection[], mode: SupplierWorkspaceMode): NavSection[] {
  if (mode === "full_erp") return nav;

  const allItems = nav.flatMap((section) => section.items);
  const byHref = new Map(allItems.map((item) => [item.href, item]));
  const fastItems = FAST_DESK_HREFS.map((href) => byHref.get(href)).filter(
    (item): item is NonNullable<typeof item> => Boolean(item),
  );
  const fastSet = new Set(fastItems.map((item) => item.href));

  const remainingSections = nav
    .map((section) => ({
      ...section,
      items: section.items.filter((item) => !fastSet.has(item.href)),
    }))
    .filter((section) => section.items.length > 0);

  return [
    {
      labelKey: "portal.nav.exceptions",
      items: fastItems,
    },
    ...remainingSections,
  ];
}
