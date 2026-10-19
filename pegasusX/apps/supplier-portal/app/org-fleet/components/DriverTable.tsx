"use client";

import { usePortalT } from "@/lib/i18n";
import { describeHomeNode, describeVehicle, ReadyState } from "./utils";

export function DriverTable({ state }: { state: ReadyState }) {
  const t = usePortalT();
  return (
    <article className="md-card md-shape-md p-6 overflow-x-auto">
      <h2 className="md-typescale-title-large">{t("supplier_portal.org_fleet.components.driver_table.text.driver_roster")}</h2>
      {state.drivers.length === 0 ? (
        <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
          No drivers have been created yet.
        </p>
      ) : (
        <table className="w-full text-left mt-4">
          <thead>
            <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
              <th className="py-2 pr-4">{t("supplier_portal.analytics.knowledge_graph.text.name")}</th>
              <th className="py-2 pr-4">{t("supplier_portal.org_fleet.components.driver_table.text.home_node")}</th>
              <th className="py-2 pr-4">{t("supplier_portal.org_fleet.components.driver_table.text.vehicle")}</th>
              <th className="py-2 pr-4">{t("common.field.phone")}</th>
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
  );
}
