import SwiftUI

struct KpiTile: View {
    let title: String
    let value: String
    let systemImage: String
    let tint: Color
    var supporting: String? = nil
    var chip: (text: String, tint: Color)? = nil
    var showChevron: Bool = true

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            HStack(alignment: .center) {
                ZStack {
                    RoundedRectangle(cornerRadius: SupplierTheme.radiusSM, style: .continuous)
                        .fill(tint.opacity(0.12))
                        .frame(width: 32, height: 32)
                    Image(systemName: systemImage)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(tint)
                }
                Spacer()
                if let chip {
                    Text(chip.text)
                        .font(.caption2.bold())
                        .padding(.horizontal, SupplierTheme.spacingSM)
                        .padding(.vertical, SupplierTheme.spacingXS)
                        .foregroundStyle(chip.tint)
                        .background(chip.tint.opacity(0.14), in: Capsule())
                } else if showChevron {
                    Image(systemName: "chevron.right")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(SupplierTheme.tertiaryLabel)
                }
            }
            Text(value)
                .font(.title2.bold())
                .minimumScaleFactor(0.8)
                .lineLimit(1)
            Text(title)
                .font(.caption.bold())
                .foregroundStyle(SupplierTheme.secondaryLabel)
            if let supporting, !supporting.isEmpty {
                Text(supporting)
                    .font(.caption2)
                    .foregroundStyle(SupplierTheme.tertiaryLabel)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .supplierCard()
        .contentShape(Rectangle())
    }
}
