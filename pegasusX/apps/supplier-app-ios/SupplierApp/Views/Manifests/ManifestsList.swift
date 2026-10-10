import SwiftUI

struct ManifestsList: View {
    let items: [SupplierManifestRow]

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(items) { row in
                NavigationLink {
                    ManifestDetailView(manifestId: row.manifestId)
                } label: {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.manifestId).font(.headline)
                        Text(L10n.format("mobile_supplier.ui.status_state", "\(row.status)", "\(row.state)")).font(.subheadline)
                        Text(L10n.format("mobile_supplier.ui.orderscount_orders_drivername", "\(row.ordersCount)", "\(row.driverName.isEmpty ? (row.driverId ?? "—") : row.driverName)"))
                            .font(.caption)
                        if let plate = row.vehiclePlate { Text(L10n.format("mobile_supplier.ui.vehicle_plate_2", "\(plate)")).font(.caption) }
                    }
                }
            }
        }
    }
}
