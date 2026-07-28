import { describeHomeNode, ReadyState } from "./utils";

export function VehicleTable({ state }: { state: ReadyState }) {
  return (
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
  );
}
