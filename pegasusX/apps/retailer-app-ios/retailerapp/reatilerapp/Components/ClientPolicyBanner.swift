import SwiftUI

struct ClientPolicyBanner: View {
    let message: String?
    var force: Bool = false
    var onUpdate: (() -> Void)? = nil
    var onDismiss: (() -> Void)? = nil

    var body: some View {
        if let message, !message.isEmpty {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack(alignment: .top, spacing: AppTheme.spacingMD) {
                    Image(systemName: force ? "exclamationmark.triangle.fill" : "arrow.down.app.fill")
                        .foregroundStyle(force ? AppTheme.warning : AppTheme.accent)
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.textPrimary)
                }
                if onUpdate != nil {
                    HStack {
                        Spacer()
                        if !force, let onDismiss {
                            Button("Later", action: onDismiss)
                                .buttonStyle(.borderless)
                        }
                        Button(force ? "Update now" : "Update") {
                            onUpdate?()
                        }
                        .buttonStyle(.borderedProminent)
                    }
                }
            }
            .padding(AppTheme.spacingLG)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(force ? AppTheme.warningSoft : AppTheme.accent.opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusCard, style: .continuous))
        }
    }
}
