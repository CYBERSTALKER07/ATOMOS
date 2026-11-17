import SwiftUI

struct GradientHeaderCard<Content: View>: View {
    let title: String
    let subtitle: String?
    let icon: String?
    let gradient: LinearGradient
    let content: Content

    init(
        title: String,
        subtitle: String? = nil,
        icon: String? = nil,
        gradient: LinearGradient = AppTheme.heroGradient,
        @ViewBuilder content: () -> Content
    ) {
        self.title = title
        self.subtitle = subtitle
        self.icon = icon
        self.gradient = gradient
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: AppTheme.spacingMD) {
                if let icon {
                    Image(systemName: icon)
                        .font(.system(.headline, weight: .semibold))
                        .foregroundStyle(.white.opacity(0.9))
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.system(.headline, design: .rounded))
                        .foregroundStyle(.white)

                    if let subtitle {
                        Text(subtitle)
                            .font(.caption)
                            .foregroundStyle(.white.opacity(0.7))
                    }
                }

                Spacer()
            }
            .padding(AppTheme.spacingLG)
            .background(gradient)

            content
                .padding(AppTheme.spacingLG)
        }
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
    }
}
