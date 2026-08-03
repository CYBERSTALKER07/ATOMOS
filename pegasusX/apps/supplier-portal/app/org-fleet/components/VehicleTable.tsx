<<<<<<< HEAD
"use client";

import { SupplierTopologyResponse } from "@pegasusx/types";
import { ReadyState, describeHomeNode } from "./utils";

export function VehicleTable({
  vehicles,
  topology,
}: {
  vehicles: ReadyState["vehicles"];
  topology: SupplierTopologyResponse;
}) {
  return (
    <article className="md-card md-shape-md p-6 overflow-x-auto">
      <h2 className="md-typescale-title-large">Vehicle roster</h2>
      {vehicles.length === 0 ? (
=======
import { describeHomeNode, ReadyState } from "./utils";

export function VehicleTable({ state }: { state: ReadyState }) {
  return (
    <article className="md-card md-shape-md p-6 overflow-x-auto">
      <h2 className="md-typescale-title-large">Vehicle roster</h2>
      {state.vehicles.length === 0 ? (
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
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
<<<<<<< HEAD
            {vehicles.map((vehicle) => (
              <tr key={vehicle.vehicle_id} className="md-typescale-body-medium">
                <td className="py-2 pr-4">{vehicle.label || vehicle.vehicle_id}</td>
                <td className="py-2 pr-4">{vehicle.license_plate}</td>
                <td className="py-2 pr-4">
                  {describeHomeNode(vehicle.home_node_type, vehicle.home_node_id, topology)}
                </td>
=======
            {state.vehicles.map((vehicle) => (
              <tr key={vehicle.vehicle_id} className="md-typescale-body-medium">
                <td className="py-2 pr-4">{vehicle.label || vehicle.vehicle_id}</td>
                <td className="py-2 pr-4">{vehicle.license_plate}</td>
                <td className="py-2 pr-4">{describeHomeNode(vehicle.home_node_type, vehicle.home_node_id, state.topology)}</td>
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </article>
  );
}
