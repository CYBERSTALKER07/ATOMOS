import SwiftUI

struct LabCard<Content: View>: View {
    let content: Content

    init(@ViewBuilder content: () -> Content) {
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            content
        }
        .background {
            RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                .fill(AppTheme.cardBackground)
                .overlay {
                    RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous)
                        .stroke(AppTheme.separator.opacity(0.15), lineWidth: 1)
                }
        }
        .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous))
    }
}

struct LabCardWithHeader<Content: View>: View {
    let title: String
    let subtitle: String?
    let icon: String?
    let content: Content

    init(
        title: String,
        subtitle: String? = nil,
        icon: String? = nil,
        @ViewBuilder content: () -> Content
    ) {
        self.title = title
        self.subtitle = subtitle
        self.icon = icon
        self.content = content()
    }

    var body: some View {
        LabCard {
            HStack(spacing: AppTheme.spacingMD) {
                if let icon {
                    ZStack {
                        Circle()
                            .fill(AppTheme.accentSoft.opacity(0.5))
                            .frame(width: 36, height: 36)
                        Image(systemName: icon)
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(AppTheme.accent)
                    }
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.system(.headline, design: .rounded))
                        .foregroundStyle(AppTheme.textPrimary)

                    if let subtitle {
                        Text(subtitle)
                            .font(.caption)
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                }

                Spacer()
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingMD)

            Rectangle()
                .fill(AppTheme.separator.opacity(0.5))
                .frame(height: AppTheme.separatorHeight)
                .padding(.horizontal, AppTheme.spacingLG)

            content
                .padding(AppTheme.spacingLG)
        }
    }
}

#Preview {
    ScrollView {
        VStack(spacing: 16) {
            LabCard {
                Text("mobile_retailer.ui.simple_card_content")
                    .padding()
            }

            LabCardWithHeader(title: "Orders", subtitle: "3 active", icon: "shippingbox") {
                Text("mobile_retailer.ui.card_content_here")
            }

            GradientHeaderCard(title: "AI Insights", subtitle: "Powered by ML", icon: "sparkles") {
                Text("mobile_retailer.ui.gradient_card_content")
            }
        }
        .padding()
    }
}
