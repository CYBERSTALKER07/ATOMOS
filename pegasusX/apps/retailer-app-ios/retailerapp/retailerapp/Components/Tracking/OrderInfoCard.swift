import SwiftUI

struct OrderInfoCard: View {
    let order: TrackingOrder
    let onDismiss: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            // Header
            HStack {
                Circle()
                    .fill(order.isGreen ? AppTheme.success : AppTheme.accent)
                    .frame(width: 8, height: 8)
                Text(order.supplierName.isEmpty ? "Unknown Supplier" : order.supplierName)
                    .font(.system(.subheadline, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
                Spacer()
                HStack(spacing: 4) {
                    Circle().fill(order.isGreen ? AppTheme.success : AppTheme.textTertiary).frame(width: 6, height: 6)
                    Text(order.state.replacingOccurrences(of: "_", with: " "))
                        .font(.system(size: 11, weight: .bold, design: .rounded))
                        .foregroundStyle(order.isGreen ? AppTheme.success : AppTheme.textTertiary)
                }
                .padding(.horizontal, 8).padding(.vertical, 4)
                .background(AppTheme.surfaceElevated)
                .clipShape(.capsule)
            }

            // Items
            Text(order.items.map { "\($0.productName) ×\($0.quantity)" }.joined(separator: ", "))
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
                .lineLimit(2)

            // Total
            Text(order.displayTotal)
                .font(.system(.caption, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
        .onTapGesture {} // Prevent pass-through
        .gesture(DragGesture(minimumDistance: 20, coordinateSpace: .local)
            .onEnded { value in
                if value.translation.height > 50 { onDismiss() }
            })
    }
}
