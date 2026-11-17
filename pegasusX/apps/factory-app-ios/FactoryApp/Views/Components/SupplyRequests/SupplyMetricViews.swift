import SwiftUI

struct SupplyMetric: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.subheadline.bold())
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(
            LabTheme.tertiaryBackground,
            in: RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous)
        )
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }
}

struct SupplyTag: View {
    let text: String
    let emphasized: Bool

    var body: some View {
        Text(text.replacingOccurrences(of: "_", with: " "))
            .font(.footnote.bold())
            .padding(.horizontal, LabTheme.spacingSM)
            .padding(.vertical, LabTheme.spacingXS)
            .background(emphasized ? LabTheme.fill : LabTheme.tertiaryBackground, in: Capsule())
            .accessibilityLabel(text.replacingOccurrences(of: "_", with: " "))
    }
}

func supplyRequestLabel(_ request: SupplyRequest) -> String {
    if request.warehouseId.isEmpty { return "Warehouse" }
    return "Warehouse \(request.warehouseId.prefix(8))"
}

func supplyVolumeLabel(_ value: Double) -> String {
    value.rounded(.towardZero) == value ? "\(Int(value)) VU" : String(format: "%.1f VU", value)
}

func supplyShortDate(_ value: String?) -> String {
    guard let value, !value.isEmpty else { return "Unscheduled" }
    return String(value.prefix { $0 != "T" })
}
