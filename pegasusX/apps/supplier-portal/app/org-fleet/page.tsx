"use client";

import { ApiClient, ApiError } from "@pegasusx/api-client";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import {
  supplierFleetDriverCreateKey,
  supplierFleetVehicleCreateKey,
  supplierOrgMemberCreateKey,
  supplierOrgMemberDeactivateKey,
  supplierOrgMemberUpdateKey,
} from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import type {
  HomeNodeType,
  Role,
  SupplierFleetDriverCreateRequest,
  SupplierFleetVehicleCreateRequest,
  SupplierOrgMemberCreateRequest,
  SupplierTopologyResponse,
} from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { PortalSection } from "@/components/portal";
import { useCallback, useEffect, useMemo, useState } from "react";

function supplierScopeId(): string {
  if (typeof window === "undefined") {
    return "supplier";
  }
  return window.localStorage.getItem("supplier_id")?.trim() || "supplier";
}

type ReadyState = {
  status: "ready";
  topology: SupplierTopologyResponse;
  orgMembers: Awaited<ReturnType<ApiClient["getSupplierOrgMembers"]>>["items"];
  drivers: Awaited<ReturnType<ApiClient["getSupplierFleetDrivers"]>>["items"];
  vehicles: Awaited<ReturnType<ApiClient["getSupplierFleetVehicles"]>>["items"];
};

type LoadState =
  | { status: "loading" }
  | ReadyState
  | { status: "error"; message: string };

type OrgFormState = {
  name: string;
  email: string;
  phone: string;
  password: string;
  role: Role;
  nodeType: HomeNodeType;
  nodeID: string;
};

type DriverFormState = {
  name: string;
  phone: string;
  pin: string;
  homeNodeType: HomeNodeType;
  homeNodeID: string;
  vehicleID: string;
};

type VehicleFormState = {
  label: string;
  licensePlate: string;
  homeNodeType: HomeNodeType;
  homeNodeID: string;
};

const orgRoleOptions: Array<{ value: Role; label: string }> = [
  { value: "ADMIN", label: "Supplier operator" },
  { value: "WAREHOUSE_ADMIN", label: "Warehouse admin" },
  { value: "FACTORY_ADMIN", label: "Factory admin" },
  { value: "PAYLOAD", label: "Payload staff" },
];

const defaultOrgForm: OrgFormState = {
  name: "",
  email: "",
  phone: "",
  password: "",
  role: "WAREHOUSE_ADMIN",
  nodeType: "WAREHOUSE",
  nodeID: "",
};

const defaultDriverForm: DriverFormState = {
  name: "",
  phone: "",
  pin: "",
  homeNodeType: "WAREHOUSE",
  homeNodeID: "",
  vehicleID: "",
};

const defaultVehicleForm: VehicleFormState = {
  label: "",
  licensePlate: "",
  homeNodeType: "WAREHOUSE",
  homeNodeID: "",
};

export default function OrgFleetPage() {
  const api = useMemo(() => createSupplierApi(), []);
  const [state, setState] = useState<LoadState>({ status: "loading" });
  const [orgForm, setOrgForm] = useState<OrgFormState>(defaultOrgForm);
  const [driverForm, setDriverForm] = useState<DriverFormState>(defaultDriverForm);
  const [vehicleForm, setVehicleForm] = useState<VehicleFormState>(defaultVehicleForm);
  const [orgMessage, setOrgMessage] = useState<string | null>(null);
  const [driverMessage, setDriverMessage] = useState<string | null>(null);
  const [vehicleMessage, setVehicleMessage] = useState<string | null>(null);
  const [orgSubmitting, setOrgSubmitting] = useState(false);
  const [driverSubmitting, setDriverSubmitting] = useState(false);
  const [vehicleSubmitting, setVehicleSubmitting] = useState(false);
  const [editingMemberId, setEditingMemberId] = useState<string | null>(null);
  const [memberActionId, setMemberActionId] = useState<string | null>(null);

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

  const warehouses = state.status === "ready" ? state.topology.warehouses : [];
  const factories = state.status === "ready" ? state.topology.factories : [];
  const activeNodeOptions = state.status === "ready"
    ? nodeOptionsFor(orgEffectiveNodeType(orgForm), state.topology)
    : [];
  const driverVehicleOptions = state.status === "ready"
    ? state.vehicles.filter(
        (vehicle) =>
          vehicle.home_node_type === driverForm.homeNodeType &&
          vehicle.home_node_id === driverForm.homeNodeID,
      )
    : [];

  async function submitOrgMember(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (state.status !== "ready") {
      return;
    }
    setOrgSubmitting(true);
    setOrgMessage(null);
    const request = buildOrgMemberRequest(orgForm);
    try {
      const response = await api.createSupplierOrgMember(
        request,
        supplierOrgMemberCreateKey(supplierScopeId(), request.phone),
      );
      setState({ ...state, orgMembers: response.items });
      setOrgForm(defaultOrgForm);
      setOrgMessage("Org member created.");
    } catch (error) {
      setOrgMessage(toErrorMessage(error));
    } finally {
      setOrgSubmitting(false);
    }
  }

  async function deactivateOrgMember(userId: string) {
    if (state.status !== "ready") {
      return;
    }
    setMemberActionId(userId);
    setOrgMessage(null);
    try {
      const response = await api.deactivateSupplierOrgMember(
        userId,
        supplierOrgMemberDeactivateKey(supplierScopeId(), userId),
      );
      setState({ ...state, orgMembers: response.items });
      setOrgMessage("Org member deactivated.");
    } catch (error) {
      setOrgMessage(toErrorMessage(error));
    } finally {
      setMemberActionId(null);
    }
  }

  async function saveOrgMemberEdit(userId: string, role: Role, nodeType: HomeNodeType, nodeID: string) {
    if (state.status !== "ready") {
      return;
    }
    setMemberActionId(userId);
    setOrgMessage(null);
    const request: import("@pegasusx/types").SupplierOrgMemberUpdateRequest = {
      supplier_role: role,
    };
    if (role === "WAREHOUSE_ADMIN") {
      request.assigned_warehouse_id = nodeID;
    } else if (role === "FACTORY_ADMIN") {
      request.assigned_factory_id = nodeID;
    } else if (role === "PAYLOAD") {
      if (nodeType === "WAREHOUSE") {
        request.assigned_warehouse_id = nodeID;
      } else {
        request.assigned_factory_id = nodeID;
      }
    }
    try {
      const revision = `${role}:${nodeType}:${nodeID}`;
      const response = await api.updateSupplierOrgMember(
        userId,
        request,
        supplierOrgMemberUpdateKey(supplierScopeId(), userId, revision),
      );
      setState({ ...state, orgMembers: response.items });
      setEditingMemberId(null);
      setOrgMessage("Org member updated.");
    } catch (error) {
      setOrgMessage(toErrorMessage(error));
    } finally {
      setMemberActionId(null);
    }
  }

  async function submitDriver(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (state.status !== "ready") {
      return;
    }
    setDriverSubmitting(true);
    setDriverMessage(null);
    const request: SupplierFleetDriverCreateRequest = {
      name: driverForm.name,
      phone: driverForm.phone,
      pin: driverForm.pin,
      home_node_type: driverForm.homeNodeType,
      home_node_id: driverForm.homeNodeID,
      vehicle_id: driverForm.vehicleID || undefined,
    };
    try {
      const response = await api.createSupplierFleetDriver(
        request,
        supplierFleetDriverCreateKey(supplierScopeId(), request.phone),
      );
      setState({ ...state, drivers: response.items });
      setDriverForm(defaultDriverForm);
      setDriverMessage("Driver created.");
    } catch (error) {
      setDriverMessage(toErrorMessage(error));
    } finally {
      setDriverSubmitting(false);
    }
  }

  async function submitVehicle(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (state.status !== "ready") {
      return;
    }
    setVehicleSubmitting(true);
    setVehicleMessage(null);
    const request: SupplierFleetVehicleCreateRequest = {
      label: vehicleForm.label || undefined,
      license_plate: vehicleForm.licensePlate,
      home_node_type: vehicleForm.homeNodeType,
      home_node_id: vehicleForm.homeNodeID,
    };
    try {
      const response = await api.createSupplierFleetVehicle(
        request,
        supplierFleetVehicleCreateKey(supplierScopeId(), request.license_plate),
      );
      setState({ ...state, vehicles: response.items });
      setVehicleForm(defaultVehicleForm);
      setVehicleMessage("Vehicle created.");
    } catch (error) {
      setVehicleMessage(toErrorMessage(error));
    } finally {
      setVehicleSubmitting(false);
    }
  }

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
          <section className="grid gap-4 md:grid-cols-4 mb-6">
            <MetricCard label="Warehouses" value={warehouses.length} />
            <MetricCard label="Factories" value={factories.length} />
            <MetricCard label="Org members" value={state.orgMembers.length} />
            <MetricCard label="Fleet entities" value={state.drivers.length + state.vehicles.length} />
          </section>

          <section className="grid gap-6 lg:grid-cols-2 mb-6">
            <article className="md-card md-shape-md p-6">
              <h2 className="md-typescale-title-large">Org members</h2>
              <p className="md-typescale-body-medium mt-2" style={{ color: "var(--color-md-outline)" }}>
                Create supplier operators, node admins, and payload staff with explicit node assignments.
              </p>
              <form className="grid gap-3 mt-4" onSubmit={submitOrgMember}>
                <input
                  className="md-input-outlined"
                  placeholder="Full name"
                  value={orgForm.name}
                  onChange={(event) => setOrgForm((current) => ({ ...current, name: event.target.value }))}
                  disabled={orgSubmitting}
                />
                <input
                  className="md-input-outlined"
                  placeholder="Email"
                  value={orgForm.email}
                  onChange={(event) => setOrgForm((current) => ({ ...current, email: event.target.value }))}
                  disabled={orgSubmitting}
                />
                <input
                  className="md-input-outlined"
                  placeholder="Phone"
                  value={orgForm.phone}
                  onChange={(event) => setOrgForm((current) => ({ ...current, phone: event.target.value }))}
                  disabled={orgSubmitting}
                />
                <input
                  className="md-input-outlined"
                  placeholder="Temporary password"
                  type="password"
                  value={orgForm.password}
                  onChange={(event) => setOrgForm((current) => ({ ...current, password: event.target.value }))}
                  disabled={orgSubmitting}
                />
                <select
                  className="md-input-outlined"
                  value={orgForm.role}
                  onChange={(event) =>
                    setOrgForm((current) => ({
                      ...current,
                      role: event.target.value as Role,
                      nodeType: event.target.value === "FACTORY_ADMIN" ? "FACTORY" : current.nodeType,
                      nodeID: "",
                    }))
                  }
                  disabled={orgSubmitting}
                >
                  {orgRoleOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>

                {orgForm.role === "PAYLOAD" && (
                  <select
                    className="md-input-outlined"
                    value={orgForm.nodeType}
                    onChange={(event) =>
                      setOrgForm((current) => ({
                        ...current,
                        nodeType: event.target.value as HomeNodeType,
                        nodeID: "",
                      }))
                    }
                    disabled={orgSubmitting}
                  >
                    <option value="WAREHOUSE">Warehouse payload staff</option>
                    <option value="FACTORY">Factory payload staff</option>
                  </select>
                )}

                {orgForm.role !== "ADMIN" && (
                  <select
                    className="md-input-outlined"
                    value={orgForm.nodeID}
                    onChange={(event) => setOrgForm((current) => ({ ...current, nodeID: event.target.value }))}
                    disabled={orgSubmitting}
                  >
                    <option value="">Select node</option>
                    {activeNodeOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                )}

                <button className="md-btn md-btn-filled" type="submit" disabled={orgSubmitting}>
                  {orgSubmitting ? "Creating member..." : "Create org member"}
                </button>
                {orgMessage && <StatusText message={orgMessage} isError={isErrorMessage(orgMessage)} />}
              </form>
            </article>

            <article className="md-card md-shape-md p-6 overflow-x-auto">
              <h2 className="md-typescale-title-large">Current org roster</h2>
              {state.orgMembers.length === 0 ? (
                <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                  No org members have been created yet.
                </p>
              ) : (
                <table className="w-full text-left mt-4">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">Name</th>
                      <th className="py-2 pr-4">Role</th>
                      <th className="py-2 pr-4">Node</th>
                      <th className="py-2 pr-4">Phone</th>
                      <th className="py-2 pr-4">Status</th>
                      <th className="py-2 pr-4">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.orgMembers.map((member) => (
                      <tr key={member.user_id} className="md-typescale-body-medium">
                        <td className="py-2 pr-4">{member.name}</td>
                        <td className="py-2 pr-4">
                          {editingMemberId === member.user_id ? (
                            <select
                              className="md-input-outlined"
                              defaultValue={member.supplier_role}
                              id={`role-${member.user_id}`}
                            >
                              {orgRoleOptions.map((option) => (
                                <option key={option.value} value={option.value}>
                                  {option.label}
                                </option>
                              ))}
                            </select>
                          ) : (
                            formatRole(member.supplier_role)
                          )}
                        </td>
                        <td className="py-2 pr-4">{describeMemberNode(member, state.topology)}</td>
                        <td className="py-2 pr-4">{member.phone}</td>
                        <td className="py-2 pr-4">{member.is_active ? "Active" : "Inactive"}</td>
                        <td className="py-2 pr-4">
                          <div className="flex flex-wrap gap-2">
                            {editingMemberId === member.user_id ? (
                              <>
                                <button
                                  type="button"
                                  className="md-btn md-btn-tonal md-typescale-label-medium"
                                  disabled={memberActionId === member.user_id}
                                  onClick={() => {
                                    const role = document.getElementById(`role-${member.user_id}`) as HTMLSelectElement;
                                    void saveOrgMemberEdit(
                                      member.user_id,
                                      role.value as Role,
                                      member.assigned_factory_id ? "FACTORY" : "WAREHOUSE",
                                      member.assigned_warehouse_id ?? member.assigned_factory_id ?? "",
                                    );
                                  }}
                                >
                                  Save
                                </button>
                                <button
                                  type="button"
                                  className="md-btn md-btn-outlined md-typescale-label-medium"
                                  onClick={() => setEditingMemberId(null)}
                                >
                                  Cancel
                                </button>
                              </>
                            ) : (
                              <>
                                <button
                                  type="button"
                                  className="md-btn md-btn-outlined md-typescale-label-medium"
                                  disabled={!member.is_active || memberActionId === member.user_id}
                                  onClick={() => setEditingMemberId(member.user_id)}
                                >
                                  Edit role
                                </button>
                                <button
                                  type="button"
                                  className="md-btn md-btn-outlined md-typescale-label-medium"
                                  disabled={!member.is_active || memberActionId === member.user_id}
                                  onClick={() => void deactivateOrgMember(member.user_id)}
                                >
                                  Deactivate
                                </button>
                              </>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </article>
          </section>

          <section className="grid gap-6 lg:grid-cols-2 mb-6">
            <article className="md-card md-shape-md p-6">
              <h2 className="md-typescale-title-large">Drivers</h2>
              <form className="grid gap-3 mt-4" onSubmit={submitDriver}>
                <input
                  className="md-input-outlined"
                  placeholder="Driver name"
                  value={driverForm.name}
                  onChange={(event) => setDriverForm((current) => ({ ...current, name: event.target.value }))}
                  disabled={driverSubmitting}
                />
                <input
                  className="md-input-outlined"
                  placeholder="Phone"
                  value={driverForm.phone}
                  onChange={(event) => setDriverForm((current) => ({ ...current, phone: event.target.value }))}
                  disabled={driverSubmitting}
                />
                <input
                  className="md-input-outlined"
                  placeholder="PIN"
                  type="password"
                  value={driverForm.pin}
                  onChange={(event) => setDriverForm((current) => ({ ...current, pin: event.target.value }))}
                  disabled={driverSubmitting}
                />
                <select
                  className="md-input-outlined"
                  value={driverForm.homeNodeType}
                  onChange={(event) =>
                    setDriverForm((current) => ({
                      ...current,
                      homeNodeType: event.target.value as HomeNodeType,
                      homeNodeID: "",
                      vehicleID: "",
                    }))
                  }
                  disabled={driverSubmitting}
                >
                  <option value="WAREHOUSE">Warehouse-based driver</option>
                  <option value="FACTORY">Factory-based driver</option>
                </select>
                <select
                  className="md-input-outlined"
                  value={driverForm.homeNodeID}
                  onChange={(event) =>
                    setDriverForm((current) => ({ ...current, homeNodeID: event.target.value, vehicleID: "" }))
                  }
                  disabled={driverSubmitting}
                >
                  <option value="">Select home node</option>
                  {nodeOptionsFor(driverForm.homeNodeType, state.topology).map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
                <select
                  className="md-input-outlined"
                  value={driverForm.vehicleID}
                  onChange={(event) => setDriverForm((current) => ({ ...current, vehicleID: event.target.value }))}
                  disabled={driverSubmitting || driverForm.homeNodeID === ""}
                >
                  <option value="">Assign vehicle later</option>
                  {driverVehicleOptions.map((vehicle) => (
                    <option key={vehicle.vehicle_id} value={vehicle.vehicle_id}>
                      {vehicle.label ? `${vehicle.label} · ${vehicle.license_plate}` : vehicle.license_plate}
                    </option>
                  ))}
                </select>
                <button className="md-btn md-btn-filled" type="submit" disabled={driverSubmitting}>
                  {driverSubmitting ? "Creating driver..." : "Create driver"}
                </button>
                {driverMessage && <StatusText message={driverMessage} isError={isErrorMessage(driverMessage)} />}
              </form>
            </article>

            <article className="md-card md-shape-md p-6 overflow-x-auto">
              <h2 className="md-typescale-title-large">Driver roster</h2>
              {state.drivers.length === 0 ? (
                <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                  No drivers have been created yet.
                </p>
              ) : (
                <table className="w-full text-left mt-4">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">Name</th>
                      <th className="py-2 pr-4">Home node</th>
                      <th className="py-2 pr-4">Vehicle</th>
                      <th className="py-2 pr-4">Phone</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.drivers.map((driver) => (
                      <tr key={driver.driver_id} className="md-typescale-body-medium">
                        <td className="py-2 pr-4">{driver.name}</td>
                        <td className="py-2 pr-4">{describeHomeNode(driver.home_node_type, driver.home_node_id, state.topology)}</td>
                        <td className="py-2 pr-4">{describeVehicle(driver.vehicle_id, state.vehicles)}</td>
                        <td className="py-2 pr-4">{driver.phone}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </article>
          </section>

          <section className="grid gap-6 lg:grid-cols-2">
            <article className="md-card md-shape-md p-6">
              <h2 className="md-typescale-title-large">Vehicles</h2>
              <form className="grid gap-3 mt-4" onSubmit={submitVehicle}>
                <input
                  className="md-input-outlined"
                  placeholder="Vehicle label"
                  value={vehicleForm.label}
                  onChange={(event) => setVehicleForm((current) => ({ ...current, label: event.target.value }))}
                  disabled={vehicleSubmitting}
                />
                <input
                  className="md-input-outlined"
                  placeholder="License plate"
                  value={vehicleForm.licensePlate}
                  onChange={(event) => setVehicleForm((current) => ({ ...current, licensePlate: event.target.value.toUpperCase() }))}
                  disabled={vehicleSubmitting}
                />
                <select
                  className="md-input-outlined"
                  value={vehicleForm.homeNodeType}
                  onChange={(event) =>
                    setVehicleForm((current) => ({
                      ...current,
                      homeNodeType: event.target.value as HomeNodeType,
                      homeNodeID: "",
                    }))
                  }
                  disabled={vehicleSubmitting}
                >
                  <option value="WAREHOUSE">Warehouse-based vehicle</option>
                  <option value="FACTORY">Factory-based vehicle</option>
                </select>
                <select
                  className="md-input-outlined"
                  value={vehicleForm.homeNodeID}
                  onChange={(event) => setVehicleForm((current) => ({ ...current, homeNodeID: event.target.value }))}
                  disabled={vehicleSubmitting}
                >
                  <option value="">Select home node</option>
                  {nodeOptionsFor(vehicleForm.homeNodeType, state.topology).map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
                <button className="md-btn md-btn-filled" type="submit" disabled={vehicleSubmitting}>
                  {vehicleSubmitting ? "Creating vehicle..." : "Create vehicle"}
                </button>
                {vehicleMessage && <StatusText message={vehicleMessage} isError={isErrorMessage(vehicleMessage)} />}
              </form>
            </article>

            <article className="md-card md-shape-md p-6 overflow-x-auto">
              <h2 className="md-typescale-title-large">Vehicle roster</h2>
              {state.vehicles.length === 0 ? (
                <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                  No vehicles have been created yet.
                </p>
              ) : (
                <table className="w-full text-left mt-4">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">Vehicle</th>
                      <th className="py-2 pr-4">License plate</th>
                      <th className="py-2 pr-4">Home node</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.vehicles.map((vehicle) => (
                      <tr key={vehicle.vehicle_id} className="md-typescale-body-medium">
                        <td className="py-2 pr-4">{vehicle.label || vehicle.vehicle_id}</td>
                        <td className="py-2 pr-4">{vehicle.license_plate}</td>
                        <td className="py-2 pr-4">{describeHomeNode(vehicle.home_node_type, vehicle.home_node_id, state.topology)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </article>
          </section>
        </>
      )}
    </PageChrome>
  );
}

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

function StatusText(props: { message: string; isError: boolean }) {
  return (
    <p
      className="md-typescale-body-medium"
      role={props.isError ? "alert" : undefined}
      style={{ color: props.isError ? "var(--color-md-error)" : "var(--color-md-outline)" }}
    >
      {props.message}
    </p>
  );
}

function buildOrgMemberRequest(form: OrgFormState): SupplierOrgMemberCreateRequest {
  const request: SupplierOrgMemberCreateRequest = {
    name: form.name,
    email: form.email || undefined,
    phone: form.phone,
    password: form.password,
    supplier_role: form.role,
  };
  if (form.role === "WAREHOUSE_ADMIN") {
    request.assigned_warehouse_id = form.nodeID;
  }
  if (form.role === "FACTORY_ADMIN") {
    request.assigned_factory_id = form.nodeID;
  }
  if (form.role === "PAYLOAD") {
    if (form.nodeType === "WAREHOUSE") {
      request.assigned_warehouse_id = form.nodeID;
    } else {
      request.assigned_factory_id = form.nodeID;
    }
  }
  return request;
}

function orgEffectiveNodeType(form: OrgFormState): HomeNodeType {
  if (form.role === "FACTORY_ADMIN") {
    return "FACTORY";
  }
  return form.nodeType;
}

function nodeOptionsFor(nodeType: HomeNodeType, topology: SupplierTopologyResponse) {
  if (nodeType === "FACTORY") {
    return topology.factories.map((factory) => ({ value: factory.factory_id, label: factory.name }));
  }
  return topology.warehouses.map((warehouse) => ({ value: warehouse.warehouse_id, label: warehouse.name }));
}

function describeMemberNode(
  member: ReadyState["orgMembers"][number],
  topology: SupplierTopologyResponse,
) {
  if (member.assigned_warehouse_id) {
    return describeHomeNode("WAREHOUSE", member.assigned_warehouse_id, topology);
  }
  if (member.assigned_factory_id) {
    return describeHomeNode("FACTORY", member.assigned_factory_id, topology);
  }
  return "Supplier-wide";
}

function describeHomeNode(nodeType: HomeNodeType, nodeID: string, topology: SupplierTopologyResponse) {
  if (nodeType === "FACTORY") {
    const factory = topology.factories.find((item) => item.factory_id === nodeID);
    return factory ? `Factory · ${factory.name}` : nodeID;
  }
  const warehouse = topology.warehouses.find((item) => item.warehouse_id === nodeID);
  return warehouse ? `Warehouse · ${warehouse.name}` : nodeID;
}

function describeVehicle(vehicleID: string | undefined, vehicles: ReadyState["vehicles"]) {
  if (!vehicleID) {
    return "Assigned later";
  }
  const vehicle = vehicles.find((item) => item.vehicle_id === vehicleID);
  if (!vehicle) {
    return vehicleID;
  }
  return vehicle.label ? `${vehicle.label} · ${vehicle.license_plate}` : vehicle.license_plate;
}

function formatRole(role: Role) {
  switch (role) {
    case "WAREHOUSE_ADMIN":
      return "Warehouse admin";
    case "FACTORY_ADMIN":
      return "Factory admin";
    case "PAYLOAD":
      return "Payload staff";
    case "ADMIN":
      return "Supplier operator";
    default:
      return role;
  }
}

function toErrorMessage(error: unknown) {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error && error.message.trim() !== "") {
    return error.message;
  }
  return "Request failed.";
}

function isErrorMessage(message: string) {
  const lower = message.toLowerCase();
  return lower.includes("failed") || lower.includes("invalid") || lower.includes("exists");
}

function newIdempotencyKey() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `pegasusx-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}