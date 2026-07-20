import SwiftUI

struct ClientPolicyBanner: View {
    let message: String?
    var force: Bool = false
    var onUpdate: (() -> Void)? = nil
    var onDismiss: (() -> Void)? = nil

    var body: some View {
        if let message, !message.isEmpty {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                HStack(alignment: .top, spacing: SupplierTheme.spacingMD) {
                    Image(systemName: force ? "exclamationmark.triangle.fill" : "arrow.down.app.fill")
                        .foregroundStyle(force ? SupplierTheme.warning : SupplierTheme.primary)
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(.primary)
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
            .padding(SupplierTheme.spacingMD)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                (force ? SupplierTheme.warning : SupplierTheme.primary).opacity(0.12)
            )
            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
        }
    }
}
