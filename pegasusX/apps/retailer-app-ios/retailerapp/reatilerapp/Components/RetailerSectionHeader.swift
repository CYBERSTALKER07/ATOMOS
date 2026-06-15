import SwiftUI

struct RetailerSectionHeader: View {
    let title: String
    var subtitle: String? = nil
    var icon: String? = nil
    var count: Int? = nil

    var body: some View {
        HStack(spacing: AppTheme.spacingSM) {
            if let icon {
                Image(systemName: icon)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(AppTheme.textSecondary)
            }

            VStack(alignment: .leading, spacing: AppTheme.spacingXS) {
                Text(title)
                    .font(.system(.headline, design: .rounded))
                    .foregroundStyle(AppTheme.textPrimary)
                if let subtitle, !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            }

            if let count {
                Text("\(count)")
                    .font(.system(.caption2, design: .rounded, weight: .bold))
                    .foregroundStyle(.white)
                    .frame(width: 20, height: 20)
                    .background(AppTheme.accent)
                    .clipShape(.circle)
            }

            Spacer()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct RetailerStatusBadge: View {
    let text: String
    var tint: Color? = nil
    var showsLiveDot: Bool = false

    private var resolvedTint: Color {
        tint ?? AppTheme.statusTint(for: text)
    }

    var body: some View {
        HStack(spacing: 4) {
            if showsLiveDot {
                Circle()
                    .fill(resolvedTint)
                    .frame(width: 6, height: 6)
            }
            Text(text.replacingOccurrences(of: "_", with: " "))
                .font(.system(size: 11, weight: .bold, design: .rounded))
        }
        .foregroundStyle(resolvedTint)
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .background(resolvedTint.opacity(0.12), in: Capsule())
    }
}
