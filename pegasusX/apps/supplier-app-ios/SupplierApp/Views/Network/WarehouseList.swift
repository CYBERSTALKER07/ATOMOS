import SwiftUI

struct WarehouseList: View {
    let warehouses: [SupplierTopologyWarehouse]

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(warehouses) { warehouse in
                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                    Text(warehouse.name).font(.headline)
                    Text(warehouse.address.isEmpty ? L10n.format("mobile_supplier.chrome.coordinates_on_file") : warehouse.address)
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                    SupplierStatusBadge(text: warehouse.isOnShift ? "ON_SHIFT" : "OFF_SHIFT")
                }
            }
        }
    }
}
