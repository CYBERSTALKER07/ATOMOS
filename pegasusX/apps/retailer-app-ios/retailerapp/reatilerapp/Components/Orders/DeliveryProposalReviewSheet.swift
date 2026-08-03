import SwiftUI

struct DeliveryProposalReviewSheet: View {
    let order: Order
    let isPending: Bool
    let onAccept: () -> Void
    let onReject: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
            Text("Review Delivery Date")
                .font(.system(.title3, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)

            Text("Warehouse proposed a new delivery date for order #\(order.id.suffix(3)).")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)

            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                if let date = order.proposedDeliveryDate, !date.isEmpty {
                    Label("Proposed: \(date)", systemImage: "calendar")
                        .font(.system(.body, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.accent)
                }
                if let reason = order.deliveryProposalReason, !reason.isEmpty {
                    Text(reason)
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            }
            .padding(AppTheme.spacingMD)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(AppTheme.surfaceElevated)
            .clipShape(.rect(cornerRadius: AppTheme.radiusMD))

            HStack(spacing: AppTheme.spacingMD) {
                Button {
                    Haptics.medium()
                    onReject()
                } label: {
                    Text("Reject")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.destructive)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, AppTheme.spacingMD)
                        .background(AppTheme.destructiveSoft.opacity(0.5))
                        .clipShape(.capsule)
                }
                .disabled(isPending)

                Button {
                    Haptics.success()
                    onAccept()
                } label: {
                    Text("Accept Date")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.cardBackground)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, AppTheme.spacingMD)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                }
                .disabled(isPending)
            }

            Spacer(minLength: 0)
        }
        .padding(AppTheme.spacingXL)
    }
}
