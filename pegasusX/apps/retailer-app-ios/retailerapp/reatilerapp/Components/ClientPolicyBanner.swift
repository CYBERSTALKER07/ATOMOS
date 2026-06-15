import SwiftUI

struct ClientPolicyBanner: View {
    let message: String?

    var body: some View {
        if let message, !message.isEmpty {
            HStack(alignment: .top, spacing: AppTheme.spacingMD) {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundStyle(AppTheme.warning)
                Text(message)
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.textPrimary)
            }
            .padding(AppTheme.spacingLG)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(AppTheme.warningSoft)
            .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous))
        }
    }
}
