import SwiftUI

struct SeasonalOverridesList: View {
    let overrides: [SeasonalOverrideRow]

    var body: some View {
        if overrides.isEmpty {
            Text("No custom seasonal overrides yet.")
                .foregroundStyle(.secondary)
        } else {
            ForEach(overrides) { row in
                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                    Text(row.name?.isEmpty == false ? row.name! : row.templateId)
                        .font(.headline)
                    Text("\(row.startDate) → \(row.endDate)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text(row.isActive ? "Active" : "Inactive")
                        .font(.caption2)
                        .foregroundStyle(row.isActive ? SupplierTheme.success : SupplierTheme.secondaryLabel)
                }
            }
        }
    }
}
