import SwiftUI

struct FactoryList: View {
    let factories: [SupplierTopologyFactory]

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(factories) { factory in
                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                    Text(factory.name).font(.headline)
                    Text(factory.address.isEmpty ? "Coordinates on file" : factory.address)
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                    SupplierStatusBadge(text: factory.isActive ? "ACTIVE" : "INACTIVE")
                }
            }
        }
    }
}
