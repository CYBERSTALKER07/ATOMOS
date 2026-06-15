import SwiftUI

struct ClientPolicyBanner: View {
    let message: String?

    var body: some View {
        if let message, !message.isEmpty {
            HStack(alignment: .top, spacing: TermTheme.s8) {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundStyle(TermTheme.warn)
                Text(message)
                    .font(.subheadline)
                    .foregroundStyle(TermTheme.accent)
            }
            .padding(TermTheme.s16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(TermTheme.warn.opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM))
            .padding(.horizontal)
        }
    }
}
