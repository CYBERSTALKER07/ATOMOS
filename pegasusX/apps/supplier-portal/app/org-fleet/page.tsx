"use client";

import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ReadyState, toErrorMessage } from "./components/utils";
import { MetricsOverview } from "./components/MetricsOverview";
import { OrgMemberForm } from "./components/OrgMemberForm";
import { OrgMemberTable } from "./components/OrgMemberTable";
import { DriverForm } from "./components/DriverForm";
import { DriverTable } from "./components/DriverTable";
import { VehicleForm } from "./components/VehicleForm";
import { VehicleTable } from "./components/VehicleTable";

type LoadState =
  | { status: "loading" }
  | ReadyState
  | { status: "error"; message: string };

export default function OrgFleetPage() {
  const api = useMemo(() => createSupplierApi(), []);
  const [state, setState] = useState<LoadState>({ status: "loading" });

  const load = useCallback(async () => {
    setState((current) => current.status === "ready" ? current : { status: "loading" });
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

  const refresh = load;

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
            warehouses={state.topology.warehouses.length}
            factories={state.topology.factories.length}
            orgMembers={state.orgMembers.length}
            fleetEntities={state.drivers.length + state.vehicles.length}
          />

          <section className="grid gap-6 lg:grid-cols-2 mb-6">
            <OrgMemberForm state={state} api={api} onCreated={refresh} />
            <OrgMemberTable 
              state={state} 
              api={api} 
              onUpdated={(orgMembers) => setState({ ...state, orgMembers })} 
            />
          </section>

          <section className="grid gap-6 lg:grid-cols-2 mb-6">
            <DriverForm state={state} api={api} onCreated={refresh} />
            <DriverTable state={state} />
          </section>

          <section className="grid gap-6 lg:grid-cols-2">
            <VehicleForm state={state} api={api} onCreated={refresh} />
            <VehicleTable state={state} />
          </section>
        </>
      )}
    </PageChrome>
  );
}