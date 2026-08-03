<<<<<<< HEAD
import { SupplierTopologyResponse } from "@pegasusx/types";
import { ReadyState } from "./utils";

function MetricCard(props: { label: string; value: number }) {
=======
export function MetricCard(props: { label: string; value: number }) {
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
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
<<<<<<< HEAD
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
=======
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
  return (
    <section className="grid gap-4 md:grid-cols-4 mb-6">
      <MetricCard label="Warehouses" value={warehouses} />
      <MetricCard label="Factories" value={factories} />
      <MetricCard label="Org members" value={orgMembers} />
      <MetricCard label="Fleet entities" value={fleetEntities} />
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
    </section>
  );
}
