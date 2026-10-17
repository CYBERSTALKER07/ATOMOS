import SwiftUI

struct PortalHandoffView: View {
    let feature: WarehousePortalFeature
    @Environment(\.openURL) private var openURL

    var body: some View {
        VStack(spacing: LabTheme.spacingLG) {
            Spacer()
            Image(systemName: "desktopcomputer")
                .font(.system(size: 48))
                .foregroundStyle(.primary)
            Text("mobile_warehouse.ui.manage_on_web_portal")
                .font(.title2.bold())
            Text(feature.handoffMessage)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 360)
            Button("mobile_warehouse.ui.open_warehouse_portal") {
                openURL(WarehousePortalLinks.url(for: feature))
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
