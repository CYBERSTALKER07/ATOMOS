import SwiftUI

struct OrderRow: View {
    let order: SupplierOrder
    var showWarehouseMenu: Bool = false
    var onDelay: (() -> Void)?
    var onReject: (() -> Void)?

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
            HStack {
                Text(order.retailerId.isEmpty ? String(order.orderId.prefix(12)) : order.retailerId)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(1)
                Spacer()
                SupplierStatusBadge(text: order.status)
            }
            Text(MoneyFormat.minor(order.totalMinor, currency: order.currency))
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(order.orderId)
                .font(.caption2.monospaced())
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, SupplierTheme.spacingXS)
        .contextMenu {
            if showWarehouseMenu {
                if let onDelay {
                    Button("supplier_portal.orders.propose_delay_dialog.text.delay_delivery") { onDelay() }
                }
                if let onReject {
                    Button("mobile_supplier.ui.reject", role: .destructive) { onReject() }
                }
            }
        }
    }
}
