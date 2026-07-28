"use client";

import { useState } from "react";
import { supplierFleetDriverCreateKey } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import { SupplierTopologyResponse, SupplierFleetDriverCreateRequest, HomeNodeType } from "@pegasusx/types";
import {
  defaultDriverForm,
  DriverFormState,
  supplierScopeId,
  toErrorMessage,
  StatusText,
  isErrorMessage,
  nodeOptionsFor,
  ReadyState,
} from "./utils";

export function DriverForm({
  topology,
  vehicles,
  onCreated,
}: {
  topology: SupplierTopologyResponse;
  vehicles: ReadyState["vehicles"];
  onCreated: () => void;
}) {
  const [driverForm, setDriverForm] = useState<DriverFormState>(defaultDriverForm);
  const [driverSubmitting, setDriverSubmitting] = useState(false);
  const [driverMessage, setDriverMessage] = useState<string | null>(null);

  const driverVehicleOptions = vehicles.filter(
    (vehicle) =>
      vehicle.home_node_type === driverForm.homeNodeType &&
      vehicle.home_node_id === driverForm.homeNodeID,
  );

  async function submitDriver(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
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
      const api = createSupplierApi();
      await api.createSupplierFleetDriver(
        request,
        supplierFleetDriverCreateKey(supplierScopeId(), request.phone),
      );
      setDriverForm(defaultDriverForm);
      setDriverMessage("Driver created.");
      onCreated();
    } catch (error) {
      setDriverMessage(toErrorMessage(error));
    } finally {
      setDriverSubmitting(false);
    }
  }

  return (
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
          {nodeOptionsFor(driverForm.homeNodeType, topology).map((option) => (
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
  );
}
