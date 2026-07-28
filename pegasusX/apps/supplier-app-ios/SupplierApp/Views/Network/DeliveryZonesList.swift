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
                Text("H3 perimeter and warehouse coverage are configured via topology.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
    }
}
