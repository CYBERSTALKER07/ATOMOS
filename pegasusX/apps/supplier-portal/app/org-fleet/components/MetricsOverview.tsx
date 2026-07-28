import { SupplierTopologyResponse } from "@pegasusx/types";
import { ReadyState } from "./utils";

function MetricCard(props: { label: string; value: number }) {
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
  topology,
  orgMembersCount,
  fleetEntitiesCount,
}: {
  topology: SupplierTopologyResponse;
  orgMembersCount: number;
  fleetEntitiesCount: number;
}) {
  const warehouses = topology.warehouses || [];
  const factories = topology.factories || [];

  return (
    <section className="grid gap-4 md:grid-cols-4 mb-6">
      <MetricCard label="Warehouses" value={warehouses.length} />
      <MetricCard label="Factories" value={factories.length} />
      <MetricCard label="Org members" value={orgMembersCount} />
      <MetricCard label="Fleet entities" value={fleetEntitiesCount} />
    </section>
  );
}
