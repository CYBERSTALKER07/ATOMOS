import SwiftUI

struct LaneMetricRow: View {
    let label: String
    let value: String

    var body: some View {
        HStack {
            Text(LocalizedStringKey(label)).foregroundStyle(.secondary)
            Spacer()
            Text(LocalizedStringKey(value)).fontWeight(.medium)
        }
        .font(.subheadline)
        .accessibilityElement(children: .combine)
    }
}

struct LaneUtilizationRow: View {
    let pct: Double

    private var tint: Color {
        if pct > 85 { return SupplierTheme.destructive }
        if pct > 75 { return SupplierTheme.warning }
        return .accentColor
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text("mobile_supplier.ui.lane_utilization").foregroundStyle(.secondary)
                Spacer()
                Text(String(format: "%.0f%%", pct)).fontWeight(.medium).foregroundStyle(tint)
            }
            .font(.subheadline)
            ProgressView(value: min(100, max(0, pct)), total: 100)
                .tint(tint)
        }
        .accessibilityElement(children: .combine)
    }
}
