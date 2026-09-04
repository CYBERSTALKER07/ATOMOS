"use client";

import { usePortalT } from "@/lib/i18n";
export function MetricCard(props: { label: string; value: number }) {
  const t = usePortalT();
  return (
    <article className="md-card md-shape-md p-4">
      <p className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
        {props.label}
      </p>
      <p className="md-typescale-title-large mt-2">{props.value}</p>
    </article>
  );
}

export function MetricsOverview({
  warehouses,
  factories,
  orgMembers,
  fleetEntities,
}: {
  warehouses: number;
  factories: number;
  orgMembers: number;
  fleetEntities: number;
}) {
  const t = usePortalT();
  return (
    <section className="grid gap-4 md:grid-cols-4 mb-6">
      <MetricCard label={t("portal.nav.warehouses")} value={warehouses} />
      <MetricCard label={t("portal.nav.factories")} value={factories} />
      <MetricCard label={t("supplier_portal.org_fleet.components.org_member_form.text.org_members")} value={orgMembers} />
      <MetricCard label={t("supplier_portal.residual.text.fleet_entities")} value={fleetEntities} />
    </section>
  );
}
