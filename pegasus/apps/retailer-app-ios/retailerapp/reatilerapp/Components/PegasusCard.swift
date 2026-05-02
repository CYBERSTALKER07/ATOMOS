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

// MARK: - LabCard with header

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
            // Header
            HStack(spacing: AppTheme.spacingMD) {
                if let icon {
                    ZStack {
                        Circle()
                            .fill(AppTheme.accentSoft.opacity(0.5))
                            .frame(width: 36, height: 36)
                        Image(systemName: icon)
                            .font(.system(size: 15, weight: .semibold))
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

            // Separator
            Rectangle()
                .fill(AppTheme.separator.opacity(0.5))
                .frame(height: AppTheme.separatorHeight)
                .padding(.horizontal, AppTheme.spacingLG)

            // Content
            content
                .padding(AppTheme.spacingLG)
        }
    }
}

// MARK: - Gradient Header Card

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
            // Gradient Header
            HStack(spacing: AppTheme.spacingMD) {
                if let icon {
                    Image(systemName: icon)
                        .font(.system(size: 18, weight: .semibold))
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

            // Content
            content
                .padding(AppTheme.spacingLG)
        }
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
    }
}

#Preview {
    ScrollView {
        VStack(spacing: 16) {
            LabCard {
                Text("Simple card content")
                    .padding()
            }

            LabCardWithHeader(title: "Orders", subtitle: "3 active", icon: "shippingbox") {
                Text("Card content here")
            }

            GradientHeaderCard(title: "AI Insights", subtitle: "Powered by ML", icon: "sparkles") {
                Text("Gradient card content")
            }
        }
        .padding()
    }
}
