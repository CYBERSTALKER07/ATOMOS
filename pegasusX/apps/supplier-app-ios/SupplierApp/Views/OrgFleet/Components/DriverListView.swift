import SwiftUI

struct DriverListView: View {
    let drivers: [FleetDriver]
    let topology: SupplierTopologyResponse?
    
    var body: some View {
        Group {
            if drivers.isEmpty {
                SupplierEmptyView(title: "No drivers", message: "Create a driver to start fleet onboarding.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(drivers) { driver in
                        VStack(alignment: .leading) {
                            Text(driver.name).font(.headline)
                            Text("\(nodeLabel(topology: topology, type: driver.homeNodeType, id: driver.homeNodeId)) · \(driver.phone)")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
            }
        }
    }
}
