"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { MetricsOverview } from "./components/MetricsOverview";
import { OrgMemberForm } from "./components/OrgMemberForm";
import { OrgMemberTable } from "./components/OrgMemberTable";
import { DriverForm } from "./components/DriverForm";
import { DriverTable } from "./components/DriverTable";
import { VehicleForm } from "./components/VehicleForm";
import { VehicleTable } from "./components/VehicleTable";
import { ReadyState, toErrorMessage } from "./components/utils";

type LoadState =
  | { status: "loading" }
  | ReadyState
  | { status: "error"; message: string };

export default function OrgFleetPage() {
  const api = useMemo(() => createSupplierApi(), []);
  const [state, setState] = useState<LoadState>({ status: "loading" });

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const [topology, orgMembers, drivers, vehicles] = await Promise.all([
        api.getSupplierTopology(),
        api.getSupplierOrgMembers(),
        api.getSupplierFleetDrivers(),
        api.getSupplierFleetVehicles(),
      ]);
      setState({
        status: "ready",
        topology,
        orgMembers: orgMembers.items,
        drivers: drivers.items,
        vehicles: vehicles.items,
      });
    } catch (error) {
      setState({ status: "error", message: toErrorMessage(error) });
    }
  }, [api]);

  useSupplierSessionReconcile(() => {
    void load();
  });

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <PageChrome
      icon="person-add"
      title="Org, staff, and fleet onboarding"
      description="Seed node admins, payload staff, drivers, and vehicles from the supplier control plane so downstream factory, warehouse, payload, and driver work is not blocked by missing entity contracts."
      loading={state.status === "loading"}
      skeletonVariant="form"
      error={state.status === "error" ? state.message : null}
    >
      {state.status === "ready" && (
        <>
          <MetricsOverview
            topology={state.topology}
            orgMembersCount={state.orgMembers.length}
            fleetEntitiesCount={state.drivers.length + state.vehicles.length}
          />

          <section className="grid gap-6 lg:grid-cols-2 mb-6">
            <OrgMemberForm topology={state.topology} onCreated={() => void load()} />
            <OrgMemberTable
              orgMembers={state.orgMembers}
              topology={state.topology}
              onUpdated={() => void load()}
            />
          </section>

          <section className="grid gap-6 lg:grid-cols-2 mb-6">
            <DriverForm
              topology={state.topology}
              vehicles={state.vehicles}
              onCreated={() => void load()}
            />
            <DriverTable
              drivers={state.drivers}
              vehicles={state.vehicles}
              topology={state.topology}
            />
          </section>

          <section className="grid gap-6 lg:grid-cols-2">
            <VehicleForm topology={state.topology} onCreated={() => void load()} />
            <VehicleTable vehicles={state.vehicles} topology={state.topology} />
          </section>
        </>
      )}
    </PageChrome>
  );
}