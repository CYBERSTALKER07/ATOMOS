import SwiftUI

struct ClientPolicyBanner: View {
    let message: String?
    var force: Bool = false
    var onUpdate: (() -> Void)? = nil
    var onDismiss: (() -> Void)? = nil

    var body: some View {
        if let message, !message.isEmpty {
            VStack(alignment: .leading, spacing: LabTheme.s8) {
                HStack(alignment: .top, spacing: LabTheme.s12) {
                    Image(systemName: force ? "exclamationmark.triangle.fill" : "arrow.down.app.fill")
                        .foregroundStyle(force ? LabTheme.warning : LabTheme.transit)
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(LabTheme.fg)
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
            .padding(LabTheme.s16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background((force ? LabTheme.warning : LabTheme.transit).opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: LabTheme.buttonRadius))
        }
    }
}
