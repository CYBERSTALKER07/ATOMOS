<<<<<<< HEAD
"use client";

import { useState } from "react";
import { supplierFleetVehicleCreateKey } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import { SupplierTopologyResponse, SupplierFleetVehicleCreateRequest, HomeNodeType } from "@pegasusx/types";
import {
  defaultVehicleForm,
  VehicleFormState,
  supplierScopeId,
  toErrorMessage,
  StatusText,
  isErrorMessage,
  nodeOptionsFor,
} from "./utils";

export function VehicleForm({
  topology,
  onCreated,
}: {
  topology: SupplierTopologyResponse;
=======
import { useState } from "react";
import type { HomeNodeType, SupplierFleetVehicleCreateRequest } from "@pegasusx/types";
import { supplierFleetVehicleCreateKey } from "@pegasusx/api-client";
import type { ApiClient } from "@pegasusx/api-client";
import {
  VehicleFormState,
  defaultVehicleForm,
  StatusText,
  isErrorMessage,
  toErrorMessage,
  supplierScopeId,
  nodeOptionsFor,
  ReadyState,
} from "./utils";

export function VehicleForm({
  state,
  api,
  onCreated,
}: {
  state: ReadyState;
  api: ApiClient;
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
  onCreated: () => void;
}) {
  const [vehicleForm, setVehicleForm] = useState<VehicleFormState>(defaultVehicleForm);
  const [vehicleSubmitting, setVehicleSubmitting] = useState(false);
  const [vehicleMessage, setVehicleMessage] = useState<string | null>(null);

  async function submitVehicle(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setVehicleSubmitting(true);
    setVehicleMessage(null);
    const request: SupplierFleetVehicleCreateRequest = {
      label: vehicleForm.label || undefined,
      license_plate: vehicleForm.licensePlate,
      home_node_type: vehicleForm.homeNodeType,
      home_node_id: vehicleForm.homeNodeID,
    };
    try {
<<<<<<< HEAD
      const api = createSupplierApi();
=======
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
      await api.createSupplierFleetVehicle(
        request,
        supplierFleetVehicleCreateKey(supplierScopeId(), request.license_plate),
      );
      setVehicleForm(defaultVehicleForm);
      setVehicleMessage("Vehicle created.");
      onCreated();
    } catch (error) {
      setVehicleMessage(toErrorMessage(error));
    } finally {
      setVehicleSubmitting(false);
    }
  }

  return (
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
<<<<<<< HEAD
          {nodeOptionsFor(vehicleForm.homeNodeType, topology).map((option) => (
=======
          {nodeOptionsFor(vehicleForm.homeNodeType, state.topology).map((option) => (
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
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
  );
}
