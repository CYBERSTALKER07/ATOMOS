import SwiftUI

struct ClientPolicyBanner: View {
    let message: String?

    var body: some View {
        if let message, !message.isEmpty {
            HStack(alignment: .top, spacing: LabTheme.spacingSM) {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundStyle(.orange)
                Text(message)
                    .font(.subheadline)
                    .foregroundStyle(.primary)
            }
            .padding(LabTheme.spacingMD)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color.orange.opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD))
            .padding(.horizontal)
        }
    }
}
