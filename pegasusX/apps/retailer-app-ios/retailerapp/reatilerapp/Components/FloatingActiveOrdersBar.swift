import SwiftUI

// MARK: - Floating Active Orders Bar (Spotify-style)

struct FloatingActiveOrdersBar: View {
    let activeOrders: [Order]
    let onTap: () -> Void

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var pulse = false

    var body: some View {
        if !activeOrders.isEmpty {
            Button(action: onTap) {
                HStack(spacing: AppTheme.spacingMD) {
                    // Pulsing dot
                    ZStack {
                        Circle()
                            .fill(AppTheme.success.opacity(0.3))
                            .frame(width: 28, height: 28)
                            .scaleEffect(pulse ? 1.3 : 1)
                            .opacity(pulse ? 0 : 0.6)
                        Circle()
                            .fill(AppTheme.success)
                            .frame(width: 10, height: 10)
                    }

                    // Info
                    VStack(alignment: .leading, spacing: 1) {
                        Text("\(activeOrders.count) Active Order\(activeOrders.count == 1 ? "" : "s")")
                            .font(.system(.caption, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)

                        if let first = activeOrders.first {
                            Text(first.status.displayName + " · " + first.displayTotal)
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                                .lineLimit(1)
                        }
                    }

                    Spacer()

                    // ETA of first order
                    if let eta = activeOrders.first?.estimatedDelivery {
                        CountdownText(
                            targetISO: eta,
                            font: .system(.caption, design: .monospaced, weight: .bold),
                            color: AppTheme.textPrimary
                        )
                    }

                    Image(systemName: "chevron.up")
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.vertical, AppTheme.spacingMD)
                .background {
                    RoundedRectangle(cornerRadius: AppTheme.radiusPill)
                        .fill(AppTheme.cardBackground)
                        .shadow(color: .black.opacity(0.12), radius: 16, x: 0, y: -4)
                        .overlay {
                            RoundedRectangle(cornerRadius: AppTheme.radiusPill)
                                .strokeBorder(AppTheme.separator.opacity(0.15), lineWidth: 0.5)
                        }
                }
            }
            .pressable()
            .padding(.horizontal, AppTheme.spacingLG)
            .onAppear {
                guard !reduceMotion else { return }
                withAnimation(.easeInOut(duration: 1.5).repeatForever(autoreverses: false)) {
                    pulse = true
                }
            }
            .transition(.move(edge: .bottom).combined(with: .opacity))
        }
    }
}

#Preview {
    VStack {
        Spacer()
        FloatingActiveOrdersBar(
            activeOrders: Order.samples.filter { $0.status.isActive },
            onTap: {}
        )
    }
    .background(AppTheme.background)
}
