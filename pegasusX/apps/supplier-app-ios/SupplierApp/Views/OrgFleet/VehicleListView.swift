import SwiftUI

struct VehicleListView: View {
    let vehicles: [FleetVehicle]
    let topology: SupplierTopologyResponse?
    
    var body: some View {
        Group {
            if vehicles.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_vehicles", message: "mobile_supplier.empty.vehicles_create")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vehicles) { vehicle in
                        VStack(alignment: .leading) {
                            Text(vehicle.label ?? vehicle.licensePlate).font(.headline)
                            Text(L10n.format("mobile_supplier.ui.licenseplate_nodelabel", "\(vehicle.licensePlate)", "\(OrgFleetUtils.nodeLabel(type: vehicle.homeNodeType, id: vehicle.homeNodeId, topology: topology))"))
                                .font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
            }
        }
    }
}
