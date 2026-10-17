import SwiftUI

struct RescueWarningBanner: View {
    let message: String?
    let onDismiss: () -> Void

    var body: some View {
        if let message, !message.isEmpty {
            HStack(alignment: .top, spacing: AppTheme.spacingMD) {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundStyle(AppTheme.warning)
                
                Text(message)
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.textPrimary)
                
                Spacer()
                
                Button(action: onDismiss) {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(AppTheme.textSecondary)
                        .font(.title3)
                }
            }
            .padding(AppTheme.spacingLG)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(AppTheme.warningSoft)
            .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous))
            .padding(.horizontal, AppTheme.spacingMD)
            .padding(.top, AppTheme.spacingMD)
        }
    }
}
