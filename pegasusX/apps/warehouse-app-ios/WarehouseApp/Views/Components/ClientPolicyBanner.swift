import SwiftUI

struct ClientPolicyBanner: View {
    let message: String?
    var force: Bool = false
    var onUpdate: (() -> Void)? = nil
    var onDismiss: (() -> Void)? = nil

    var body: some View {
        if let message, !message.isEmpty {
            VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                HStack(alignment: .top, spacing: LabTheme.spacingSM) {
                    Image(systemName: force ? "exclamationmark.triangle.fill" : "arrow.down.app.fill")
                        .foregroundStyle(force ? Color.orange : Color.accentColor)
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(.primary)
                }
                if onUpdate != nil {
                    HStack {
                        Spacer()
                        if !force, let onDismiss {
                            Button("mobile_warehouse.ui.later", action: onDismiss)
                                .buttonStyle(.borderless)
                        }
                        Button(force ? "Update now" : "Update") {
                            onUpdate?()
                        }
                        .buttonStyle(.borderedProminent)
                    }
                }
            }
            .padding(LabTheme.spacingMD)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background((force ? Color.orange : Color.accentColor).opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD))
            .padding(.horizontal)
        }
    }
}
