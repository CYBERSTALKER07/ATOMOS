import SwiftUI

struct TacticalStatusBadge: View {
    let status: String
    var overrideTint: Color? = nil
    var showPulse: Bool = true

    private var tint: Color {
        if let overrideTint { return overrideTint }
        switch status.uppercased() {
        case "COMPLETED", "DONE", "ACTIVE", "APPROVED", "SEALED", "DISPATCHED", "RECEIVED", "PAID", "OK", "SUCCESS", "ON_ROUTE":
            return TacticalTheme.statusSuccess
        case "PENDING", "AWAITING_REVIEW", "AWAITING_PAYMENT", "LOADING", "IN_TRANSIT", "OPEN", "SUBMITTED", "STAGED", "WARN":
            return TacticalTheme.statusWarning
        case "CANCELLED", "REJECTED", "FAILED", "EXCEPTION", "CRITICAL", "OVERDUE", "FAIL", "HOLD":
            return TacticalTheme.statusDanger
        case "LIVE", "TELEMETRY":
            return TacticalTheme.accentCyan
        default:
            return TacticalTheme.textSecondary
        }
    }

    var body: some View {
        HStack(spacing: 5) {
            if showPulse {
                Circle()
                    .fill(tint)
                    .frame(width: 6, height: 6)
                    .overlay(
                        Circle()
                            .stroke(tint.opacity(0.4), lineWidth: 2)
                    )
            }

            Text(status.replacingOccurrences(of: "_", with: " ").uppercased())
                .font(.system(size: 10, weight: .bold, design: .monospaced))
                .tracking(0.8)
                .foregroundStyle(tint)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(tint.opacity(0.12))
        .clipShape(Capsule())
        .overlay(
            Capsule()
                .stroke(tint.opacity(0.35), lineWidth: 1)
        )
    }
}
