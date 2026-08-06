"use client";

import { usePortalT } from "@/lib/i18n";
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
  onCreated: () => void;
}) {
  const [vehicleForm, setVehicleForm] = useState<VehicleFormState>(defaultVehicleForm);
  const [vehicleSubmitting, setVehicleSubmitting] = useState(false);
  const [vehicleMessage, setVehicleMessage] = useState<string | null>(null);

  async function submitVehicle(event: React.FormEvent<HTMLFormElement>) {
  const t = usePortalT();
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
      <h2 className="md-typescale-title-large">{t("supplier_portal.org_fleet.components.vehicle_form.text.vehicles")}</h2>
      <form className="grid gap-3 mt-4" onSubmit={submitVehicle}>
        <input
          className="md-input-outlined"
          placeholder={t("supplier_portal.org_fleet.components.vehicle_form.text.vehicle_label")}
          value={vehicleForm.label}
          onChange={(event) => setVehicleForm((current) => ({ ...current, label: event.target.value }))}
          disabled={vehicleSubmitting}
        />
        <input
          className="md-input-outlined"
          placeholder={t("supplier_portal.org_fleet.components.vehicle_form.text.license_plate")}
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
          <option value="WAREHOUSE">{t("supplier_portal.org_fleet.components.vehicle_form.text.warehouse_based_vehicle")}</option>
          <option value="FACTORY">{t("supplier_portal.org_fleet.components.vehicle_form.text.factory_based_vehicle")}</option>
        </select>
        <select
          className="md-input-outlined"
          value={vehicleForm.homeNodeID}
          onChange={(event) => setVehicleForm((current) => ({ ...current, homeNodeID: event.target.value }))}
          disabled={vehicleSubmitting}
        >
          <option value="">{t("supplier_portal.org_fleet.components.driver_form.text.select_home_node")}</option>
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
  );
}
