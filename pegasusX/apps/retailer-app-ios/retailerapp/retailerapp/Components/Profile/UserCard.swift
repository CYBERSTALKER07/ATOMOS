import SwiftUI

struct UserCard: View {
    var displayName: String
    var displayCompany: String
    var userEmail: String?
    var profilePhone: String

    var body: some View {
        VStack(spacing: 0) {
            // Gradient header
            ZStack(alignment: .bottomLeading) {
                AppTheme.heroGradient
                    .frame(height: 80)

                HStack(spacing: AppTheme.spacingLG) {
                    ZStack {
                        Circle()
                            .fill(.white)
                            .frame(width: 68, height: 68)
                            .shadow(color: AppTheme.accent.opacity(0.2), radius: 8, y: 4)
                        Text(String(displayName.prefix(1)))
                            .font(.system(.title, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.accent)
                    }
                    .offset(y: 34)

                    Spacer()
                }
                .padding(.horizontal, AppTheme.spacingXL)
            }

            // Info
            VStack(alignment: .leading, spacing: AppTheme.spacingXS) {
                Text(displayName)
                    .font(.system(.title3, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                Text(displayCompany)
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)

                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "envelope")
                        .font(.system(size: 12))
                    Text(userEmail ?? "—")
                        .font(.system(.caption, design: .rounded))
                }
                .foregroundStyle(AppTheme.textTertiary)

                if !profilePhone.isEmpty {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "phone")
                            .font(.system(size: 12))
                        Text(profilePhone)
                            .font(.system(.caption, design: .rounded))
                    }
                    .foregroundStyle(AppTheme.textTertiary)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, AppTheme.spacingXL)
            .padding(.top, AppTheme.spacingHuge)
            .padding(.bottom, AppTheme.spacingLG)
        }
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
    }
}
