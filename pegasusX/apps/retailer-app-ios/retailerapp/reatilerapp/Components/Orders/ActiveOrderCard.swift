import SwiftUI

struct ActiveOrderCard: View {
    let order: Order
    let onDetails: () -> Void
    let onQR: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack(alignment: .top) {
                ZStack {
                    Circle().fill(AppTheme.success.opacity(0.1)).frame(width: 40, height: 40)
                    Image(systemName: "shippingbox.fill").font(.system(size: 15, weight: .semibold)).foregroundStyle(AppTheme.success)
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text("Order #\(order.id.suffix(3))")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("\(order.itemCount) items · \(order.displayTotal)")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Spacer()

                RetailerStatusBadge(
                    text: order.status.displayName,
                    tint: AppTheme.statusTint(for: order.status.displayName),
                    showsLiveDot: true
                )
            }

            // Order Status Timeline
            OrderStatusTimeline(currentStep: order.status.timelineStepIndex)

            if let eta = order.estimatedDelivery {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "clock").font(.system(size: 12, weight: .semibold)).foregroundStyle(AppTheme.textSecondary)
                    CountdownText(targetISO: eta, font: .system(.caption, design: .monospaced, weight: .bold), color: AppTheme.textPrimary)
                    Spacer()
                }
                .padding(AppTheme.spacingSM)
                .background(AppTheme.surfaceElevated)
                .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
            }

            Rectangle().fill(AppTheme.separator.opacity(0.2)).frame(height: AppTheme.separatorHeight)

            HStack(spacing: AppTheme.spacingMD) {
                Button {
                    Haptics.light()
                    onDetails()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "doc.text").font(.system(size: 12, weight: .semibold))
                        Text("Details").font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textPrimary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                }

                if order.status.hasDeliveryToken {
                    Button {
                        Haptics.light()
                        onQR()
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "qrcode").font(.system(size: 12, weight: .semibold))
                            Text("Show QR").font(.system(.caption, design: .rounded, weight: .semibold))
                        }
                        .foregroundStyle(.white)
                        .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                    }
                } else {
                    HStack(spacing: 4) {
                        Image(systemName: "qrcode").font(.system(size: 12, weight: .semibold))
                        Text("Awaiting Dispatch").font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                }
                Spacer()
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
    }
}
