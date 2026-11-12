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
                        .foregroundStyle(force ? SupplierTheme.warning : Color.accentColor)
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(.primary)
                }
                if onUpdate != nil {
                    HStack {
                        Spacer()
                        if !force, let onDismiss {
                            Button("mobile_supplier.ui.later", action: onDismiss)
                                .buttonStyle(.borderless)
                        }
                        Button(LocalizedStringKey(force ? "mobile_supplier.chrome.update_now" : "mobile_supplier.chrome.update")) {
                            onUpdate?()
                        }
                        .buttonStyle(.borderedProminent)
                    }
                }
            }
            .padding(SupplierTheme.spacingMD)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                (force ? SupplierTheme.warning : Color.accentColor).opacity(0.12)
            )
            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
        }
    }
}

struct SupplierRefreshNotice: View {
    let message: String
    let dismiss: () -> Void

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: SupplierTheme.spacingMD) {
            Text(message)
                .font(.footnote)
                .foregroundStyle(.primary)
                .frame(maxWidth: .infinity, alignment: .leading)
            Button("mobile_supplier.chrome.dismiss", action: dismiss)
                .font(.footnote.weight(.semibold))
                .buttonStyle(.borderless)
        }
        .padding(.horizontal)
        .padding(.vertical, 8)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.yellow.opacity(0.25))
        .accessibilityElement(children: .combine)
    }
}
