import SwiftUI

struct DeliveryZonesList: View {
    let warehouses: [SupplierTopologyWarehouse]

    var body: some View {
        ResponsiveGridContentWrapper {
            Section("Warehouse coverage") {
                ForEach(warehouses) { node in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(node.name.isEmpty ? "Unnamed warehouse" : node.name).font(.body)
                        Text(node.address.isEmpty ? "Coordinates on file" : node.address)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            Section {
                Text("mobile_supplier.ui.h3_perimeter_and_warehouse_coverage_are_configured_via_topology")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
    }
}
