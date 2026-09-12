import SwiftUI

struct FleetLiveMapView: View {
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                WarehouseSectionHeader(
                    title: "Live fleet map",
                    subtitle: "Sealed manifests with route geometry and driver markers"
                )
                FleetLiveMapSection(mapHeight: 520, showsExpand: false)
            }
            .labReadableWidth()
            .padding()
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.live_fleet")
        .navigationBarTitleDisplayMode(.inline)
    }
}
