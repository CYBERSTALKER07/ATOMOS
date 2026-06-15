import SwiftUI

struct ClientPolicyBanner: View {
    let message: String?

    var body: some View {
        if let message, !message.isEmpty {
            HStack(alignment: .top, spacing: SupplierTheme.spacingMD) {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundStyle(SupplierTheme.warning)
                Text(message)
                    .font(.subheadline)
                    .foregroundStyle(.primary)
            }
            .padding(SupplierTheme.spacingMD)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(SupplierTheme.warning.opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
        }
    }
}
