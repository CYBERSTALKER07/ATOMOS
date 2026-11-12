import SwiftUI

// MARK: - Section header

struct DriverSectionHeader: View {
    let title: String
    var subtitle: String? = nil
    var trailing: String? = nil
    var trailingTint: Color = LabTheme.fg

    var body: some View {
        HStack(alignment: .firstTextBaseline) {
            VStack(alignment: .leading, spacing: LabTheme.s4) {
                Text(title.uppercased())
                    .font(.system(.caption, design: .monospaced, weight: .black))
                    .foregroundStyle(LabTheme.fgSecondary)
                if let subtitle, !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.system(.caption2, design: .monospaced, weight: .medium))
                        .foregroundStyle(LabTheme.fgTertiary)
                }
            }
            Spacer()
            if let trailing, !trailing.isEmpty {
                Text(trailing.uppercased())
                    .font(.system(.caption2, design: .monospaced, weight: .black))
                    .foregroundStyle(trailingTint)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .accessibilityElement(children: .combine)
    }
}

// MARK: - Status badge

struct DriverStatusBadge: View {
    let text: String
    var tint: Color? = nil
    var large = false

    private var resolvedTint: Color {
        tint ?? LabTheme.statusTint(for: text)
    }

    var body: some View {
        Text(text.replacingOccurrences(of: "_", with: " ").uppercased())
            .font(.system(large ? .caption : .caption2, design: .monospaced, weight: .black))
            .foregroundStyle(resolvedTint)
            .padding(.horizontal, large ? 12 : 8)
            .padding(.vertical, large ? 6 : 4)
            .background(resolvedTint.opacity(0.14), in: Capsule())
            .overlay(Capsule().stroke(resolvedTint.opacity(0.2), lineWidth: 0.5))
    }
}

#Preview {
    VStack(spacing: 16) {
        DriverSectionHeader(title: "Today", subtitle: "Route summary", trailing: "JUN 15")
        HStack(spacing: 12) {
            KpiTile(label: "Pending", value: "4", icon: "clock")
            KpiTile(label: "Done", value: "12", icon: "checkmark", tint: LabTheme.success)
        }
        HStack(spacing: 8) {
            DriverStatusBadge(text: "ON_ROUTE")
            DriverStatusBadge(text: "AWAITING_SEAL", tint: LabTheme.warning)
        }
    }
    .padding()
    .background(LabTheme.bg)
}
