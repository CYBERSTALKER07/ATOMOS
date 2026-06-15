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
                    .font(.system(size: 12, weight: .black, design: .monospaced))
                    .foregroundStyle(LabTheme.fgSecondary)
                if let subtitle, !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                }
            }
            Spacer()
            if let trailing, !trailing.isEmpty {
                Text(trailing.uppercased())
                    .font(.system(size: 11, weight: .black, design: .monospaced))
                    .foregroundStyle(trailingTint)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
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
            .font(.system(size: large ? 11 : 9, weight: .black, design: .monospaced))
            .foregroundStyle(resolvedTint)
            .padding(.horizontal, large ? 12 : 8)
            .padding(.vertical, large ? 6 : 4)
            .background(resolvedTint.opacity(0.14), in: Capsule())
            .overlay(Capsule().stroke(resolvedTint.opacity(0.2), lineWidth: 0.5))
    }
}

// MARK: - KPI tile

struct KpiTile: View {
    let label: String
    let value: String
    var icon: String? = nil
    var tint: Color = LabTheme.fg

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.s8) {
            if let icon {
                Image(systemName: icon)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            Text(label.uppercased())
                .font(.system(size: 10, weight: .black, design: .monospaced))
                .foregroundStyle(LabTheme.fgTertiary)
            Text(value)
                .font(.system(size: value.count > 10 ? 14 : 18, weight: .bold, design: .monospaced))
                .foregroundStyle(tint)
                .minimumScaleFactor(0.7)
                .lineLimit(1)
        }
        .padding(LabTheme.s16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
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
