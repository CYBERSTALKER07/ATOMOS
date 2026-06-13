import SwiftUI

struct PortalHandoffView: View {
    let feature: SupplierPortalFeature
    @Environment(\.openURL) private var openURL

    var body: some View {
        VStack(spacing: SupplierTheme.spacingLG) {
            Spacer()
            Image(systemName: "desktopcomputer")
                .font(.system(size: 48))
                .foregroundStyle(.primary)
            Text("Manage on web portal")
                .font(.title2.bold())
            Text(feature.handoffMessage)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 360)
            Button("Open supplier portal") {
                openURL(SupplierPortalLinks.url(for: feature))
            }
            .buttonStyle(.borderedProminent)
            .frame(maxWidth: 360)
            Spacer()
        }
        .padding()
        .navigationTitle(feature.title)
        .navigationBarTitleDisplayMode(.inline)
    }
}
