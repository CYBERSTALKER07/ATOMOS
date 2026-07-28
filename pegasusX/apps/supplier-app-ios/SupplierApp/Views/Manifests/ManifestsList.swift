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
                        Text("\(row.status) · \(row.state)").font(.subheadline)
                        Text("\(row.ordersCount) orders · \(row.driverName.isEmpty ? (row.driverId ?? "—") : row.driverName)")
                            .font(.caption)
                        if let plate = row.vehiclePlate { Text("Vehicle \(plate)").font(.caption) }
                    }
                }
            }
        }
    }
}
