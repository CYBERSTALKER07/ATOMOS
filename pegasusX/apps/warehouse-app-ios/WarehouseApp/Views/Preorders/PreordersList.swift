import SwiftUI

struct PreordersList: View {
    let rows: [WarehousePreorderRow]
    let acting: Bool
    let onPropose: (WarehousePreorderRow) -> Void
    let onReject: (WarehousePreorderRow) -> Void

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(rows) { row in
                VStack(alignment: .leading, spacing: 6) {
                    Text(row.orderId).font(.headline)
                    Text(L10n.format("mobile_warehouse.ui.status_status", "\(row.status)")).font(.caption)
                    if let date = row.requestedDeliveryDate {
                        Text(L10n.format("mobile_warehouse.ui.delivery_date_2", "\(date)")).font(.caption2)
                    }
                    if let proposed = row.proposedDeliveryDate {
                        Text(L10n.format("mobile_warehouse.ui.proposed_proposed_2", "\(proposed)"))
                            .font(.caption2)
                            .foregroundStyle(.tint)
                    }
                    if let reason = row.deliveryProposalReason, !reason.isEmpty {
                        Text(L10n.format("mobile_warehouse.ui.reason_reason_2", "\(reason)")).font(.caption2).foregroundStyle(.secondary)
                    }
                    if showsReviewBadge(row) {
                        Text("mobile_warehouse.ui.awaiting_retailer_review")
                            .font(.caption2)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 2)
                            .background(.orange.opacity(0.15))
                            .clipShape(Capsule())
                    }
                    HStack {
                        Button("mobile_warehouse.ui.propose_date") {
                            onPropose(row)
                        }
                        .disabled(acting)
                        Button("mobile_warehouse.ui.reject", role: .destructive) {
                            onReject(row)
                        }
                        .disabled(acting)
                    }
                    .font(.subheadline)
                }
                .padding(.vertical, 4)
            }
        }
    }

    private func showsReviewBadge(_ row: WarehousePreorderRow) -> Bool {
        row.confirmationStatus == "PENDING_WAREHOUSE" || row.preorderBadge == "REVIEW_DELIVERY"
    }
}
