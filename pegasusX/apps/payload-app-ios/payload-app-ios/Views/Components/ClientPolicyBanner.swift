import SwiftUI

struct ClientPolicyBanner: View {
    let message: String?
    var force: Bool = false
    var onUpdate: (() -> Void)? = nil
    var onDismiss: (() -> Void)? = nil

    var body: some View {
        if let message, !message.isEmpty {
            VStack(alignment: .leading, spacing: 8) {
                HStack(alignment: .top, spacing: 12) {
                    Image(systemName: force ? "exclamationmark.triangle.fill" : "arrow.down.app.fill")
                        .foregroundStyle(force ? Color.orange : Color.accentColor)
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(.primary)
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel("\(force ? "Required update" : "Update available"): \(message)")

                if onUpdate != nil {
                    HStack(spacing: 12) {
                        Spacer()
                        if !force, let onDismiss {
                            Button {
                                PegasusHaptics.light()
                                onDismiss()
                            } label: {
                                Text("mobile_payload.ui.later")
                                    .frame(minWidth: 44, minHeight: 44)
                            }
                            .buttonStyle(.borderless)
                        }
                        Button {
                            PegasusHaptics.medium()
                            onUpdate?()
                        } label: {
                            Text(force ? "Update now" : "Update")
                                .frame(minWidth: 44, minHeight: 44)
                        }
                        .buttonStyle(.borderedProminent)
                    }
                }
            }
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background((force ? Color.orange : Color.accentColor).opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        }
    }
}
