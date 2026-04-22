import SwiftUI

struct OrderCardView: View {
    let order: Order
    var onCancel: (() -> Void)?

    @State private var appeared = false

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            // Header row
            HStack(alignment: .top) {
                // Order icon
                ZStack {
                    Circle()
                        .fill(statusColor.opacity(0.12))
                        .frame(width: 42, height: 42)
                    Image(systemName: statusIcon)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(statusColor)
                }

                VStack(alignment: .leading, spacing: 3) {
                    Text("Order #\(order.id.suffix(3))")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)

                    Text("\(order.itemCount) items · \(order.displayTotal)")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Spacer()

                // Status Badge
                statusBadge
            }

            // Progress indicator for active orders
            if order.status.isActive {
                orderProgressBar
            }

            // Countdown for active orders
            if order.status.isActive, let eta = order.estimatedDelivery {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "clock")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(AppTheme.accent)

                    CountdownText(targetISO: eta, font: .system(.caption, design: .monospaced, weight: .bold), color: AppTheme.accent)

                    Spacer()
                }
                .padding(AppTheme.spacingSM)
                .background(AppTheme.accentSoft.opacity(0.3))
                .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
            }

            // Items preview
            VStack(alignment: .leading, spacing: 6) {
                ForEach(order.items.prefix(3)) { item in
                    HStack(spacing: AppTheme.spacingSM) {
                        Circle()
                            .fill(AppTheme.accentSoft.opacity(0.5))
                            .frame(width: 6, height: 6)

                        Text(item.productName)
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                            .lineLimit(1)

                        Spacer()

                        Text("×\(item.quantity)")
                            .font(.system(.caption2, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textTertiary)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(AppTheme.surfaceElevated)
                            .clipShape(.capsule)
                    }
                }
                if order.items.count > 3 {
                    Text("+\(order.items.count - 3) more items")
                        .font(.caption2)
                        .foregroundStyle(AppTheme.textTertiary)
                        .italic()
                }
            }

            // Footer
            if order.status == .pending {
                Rectangle()
                    .fill(AppTheme.separator.opacity(0.5))
                    .frame(height: AppTheme.separatorHeight)

                HStack {
                    Spacer()
                    Button {
                        Haptics.medium()
                        onCancel?()
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "xmark")
                                .font(.system(size: 10, weight: .bold))
                            Text("Cancel Order")
                                .font(.system(.caption, design: .rounded, weight: .semibold))
                        }
                        .foregroundStyle(AppTheme.destructive)
                        .padding(.horizontal, AppTheme.spacingMD)
                        .padding(.vertical, AppTheme.spacingSM)
                        .background(AppTheme.destructiveSoft.opacity(0.5))
                        .clipShape(.capsule)
                    }
                }
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
        .opacity(appeared ? 1 : 0)
        .offset(y: appeared ? 0 : 16)
        .onAppear {
            withAnimation(AnimationConstants.fluid) {
                appeared = true
            }
        }
    }

    // MARK: - Status Badge

    private var statusBadge: some View {
        HStack(spacing: 4) {
            if order.status.isActive {
                Circle()
                    .fill(statusColor)
                    .frame(width: 6, height: 6)
            }
            Text(order.status.displayName)
                .font(.system(size: 11, weight: .bold, design: .rounded))
        }
        .foregroundStyle(statusColor)
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .background(statusColor.opacity(0.1))
        .clipShape(.capsule)
    }

    // MARK: - Progress Bar

    private var orderProgressBar: some View {
        let progress: Double = switch order.status {
        case .pending: 0.15
        case .loaded: 0.4
        case .inTransit: 0.7
        case .arrived: 0.9
        default: 1.0
        }

        return GeometryReader { geo in
            ZStack(alignment: .leading) {
                Capsule()
                    .fill(AppTheme.separator.opacity(0.3))
                    .frame(height: 4)

                Capsule()
                    .fill(
                        LinearGradient(
                            colors: [statusColor.opacity(0.7), statusColor],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )
                    .frame(width: geo.size.width * progress, height: 4)
                    .animation(.easeOut(duration: 0.8), value: progress)
            }
        }
        .frame(height: 4)
    }

    // MARK: - Helpers

    private var statusColor: Color {
        switch order.status {
        case .pending: AppTheme.warning
        case .loaded: AppTheme.info
        case .dispatched: AppTheme.accent
        case .inTransit: AppTheme.accent
        case .arrived: AppTheme.success
        case .awaitingPayment: AppTheme.warning
        case .pendingCashCollection: AppTheme.warning
        case .completed: AppTheme.success
        case .cancelled: AppTheme.destructive
        }
    }

    private var statusIcon: String {
        switch order.status {
        case .pending: "clock"
        case .loaded: "shippingbox"
        case .dispatched: "truck"
        case .inTransit: "shippingbox.fill"
        case .arrived: "checkmark.circle"
        case .awaitingPayment: "creditcard"
        case .pendingCashCollection: "banknote"
        case .completed: "checkmark.circle.fill"
        case .cancelled: "xmark.circle"
        }
    }
}

#Preview {
    ScrollView {
        VStack(spacing: 16) {
            ForEach(Order.samples) { order in
                OrderCardView(order: order)
            }
        }
        .padding()
    }
}
