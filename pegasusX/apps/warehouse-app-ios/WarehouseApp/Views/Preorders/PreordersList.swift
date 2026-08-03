import SwiftUI

struct PreordersList: View {
    let rows: [WarehousePreorderRow]
    let acting: Bool
    let onProposeDate: (WarehousePreorderRow) -> Void
    let onReject: (WarehousePreorderRow) -> Void

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(rows) { row in
                VStack(alignment: .leading, spacing: 6) {
                    Text(row.orderId).font(.headline)
                    Text("Status: \(row.status)").font(.caption)
                    if let date = row.requestedDeliveryDate {
                        Text("Delivery: \(date)").font(.caption2)
                    }
                    if let proposed = row.proposedDeliveryDate {
                        Text("Proposed: \(proposed)")
                            .font(.caption2)
                            .foregroundStyle(.tint)
                    }
                    if let reason = row.deliveryProposalReason, !reason.isEmpty {
                        Text("Reason: \(reason)").font(.caption2).foregroundStyle(.secondary)
                    }
                    if showsReviewBadge(row) {
                        Text("Awaiting retailer review")
                            .font(.caption2)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 2)
                            .background(.orange.opacity(0.15))
                            .clipShape(Capsule())
                    }
                    HStack {
                        Button("Propose date") {
                            onProposeDate(row)
                        }
                        .disabled(acting)
                        Button("Reject", role: .destructive) {
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
