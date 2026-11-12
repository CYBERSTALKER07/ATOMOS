import SwiftUI

struct SeasonalOverridesList: View {
    let overrides: [SeasonalOverrideRow]

    var body: some View {
        if overrides.isEmpty {
            Text("mobile_supplier.ui.no_custom_seasonal_overrides_yet")
                .foregroundStyle(.secondary)
        } else {
            ForEach(overrides) { row in
                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                    Text((row.name?.isEmpty == false ? row.name : nil) ?? row.templateId)
                        .font(.headline)
                    Text(L10n.format("mobile_supplier.ui.startdate_enddate", "\(row.startDate)", "\(row.endDate)"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    if let m = row.multiplier {
                        Text("×\(m)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    Text(row.isActive ? L10n.format("mobile_supplier.chrome.active") : L10n.format("mobile_supplier.chrome.inactive"))
                        .font(.caption2)
                        .foregroundStyle(row.isActive ? SupplierTheme.success : SupplierTheme.secondaryLabel)
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel("\((row.name?.isEmpty == false ? row.name : nil) ?? row.templateId), \(row.startDate) to \(row.endDate), \(row.isActive ? "active" : "inactive")")
            }
        }
    }
}
