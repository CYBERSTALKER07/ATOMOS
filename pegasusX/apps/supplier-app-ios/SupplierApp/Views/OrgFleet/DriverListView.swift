import SwiftUI

struct DriverListView: View {
    let drivers: [FleetDriver]
    let topology: SupplierTopologyResponse?
    
    var body: some View {
        Group {
            if drivers.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_drivers", message: "mobile_supplier.empty.drivers_onboarding")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(drivers) { driver in
                        VStack(alignment: .leading) {
                            Text(driver.name).font(.headline)
                            Text(L10n.format("mobile_supplier.ui.nodelabel_phone", "\(OrgFleetUtils.nodeLabel(type: driver.homeNodeType, id: driver.homeNodeId, topology: topology))", "\(driver.phone)"))
                                .font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
            }
        }
    }
}
