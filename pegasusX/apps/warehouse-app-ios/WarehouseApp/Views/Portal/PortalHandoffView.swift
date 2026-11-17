import SwiftUI

struct PortalHandoffView: View {
    let feature: WarehousePortalFeature
    @Environment(\.openURL) private var openURL

    var body: some View {
        VStack(spacing: LabTheme.spacingLG) {
            Spacer()
            Image(systemName: "desktopcomputer")
                .font(.largeTitle)
                .imageScale(.large)
                .foregroundStyle(.primary)
                .accessibilityHidden(true)
            Text("mobile_warehouse.ui.manage_on_web_portal")
                .font(.title2.bold())
            Text(feature.handoffMessage)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 360)
            Button {
                PegasusHaptics.light()
                openURL(WarehousePortalLinks.url(for: feature))
            } label: {
                Text("mobile_warehouse.ui.open_warehouse_portal")
                    .frame(maxWidth: 360, minHeight: 44)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.borderedProminent)
            Spacer()
        }
        .padding()
        .navigationTitle(feature.title)
        .navigationBarTitleDisplayMode(.inline)
    }
}
