import SwiftUI

struct WarehouseList: View {
    let warehouses: [SupplierTopologyWarehouse]

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(warehouses) { warehouse in
                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                    Text(warehouse.name).font(.headline)
                    Text(warehouse.address.isEmpty ? "Coordinates on file" : warehouse.address)
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                    SupplierStatusBadge(text: warehouse.isOnShift ? "ON_SHIFT" : "OFF_SHIFT")
                }
            }
        }
    }
}
