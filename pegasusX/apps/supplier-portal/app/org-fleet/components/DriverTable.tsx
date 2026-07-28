"use client";

import { SupplierTopologyResponse } from "@pegasusx/types";
import { ReadyState, describeHomeNode, describeVehicle } from "./utils";

export function DriverTable({
  drivers,
  vehicles,
  topology,
}: {
  drivers: ReadyState["drivers"];
  vehicles: ReadyState["vehicles"];
  topology: SupplierTopologyResponse;
}) {
  return (
    <article className="md-card md-shape-md p-6 overflow-x-auto">
      <h2 className="md-typescale-title-large">Driver roster</h2>
      {drivers.length === 0 ? (
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
            {drivers.map((driver) => (
              <tr key={driver.driver_id} className="md-typescale-body-medium">
                <td className="py-2 pr-4">{driver.name}</td>
                <td className="py-2 pr-4">
                  {describeHomeNode(driver.home_node_type, driver.home_node_id, topology)}
                </td>
                <td className="py-2 pr-4">{describeVehicle(driver.vehicle_id, vehicles)}</td>
                <td className="py-2 pr-4">{driver.phone}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </article>
  );
}
