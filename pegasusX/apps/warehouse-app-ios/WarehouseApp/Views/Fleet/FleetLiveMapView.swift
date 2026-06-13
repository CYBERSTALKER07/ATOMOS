import SwiftUI

struct FleetLiveMapView: View {
    var body: some View {
        ScrollView {
            FleetLiveMapSection(mapHeight: 520, showsExpand: false)
                .padding()
        }
        .background(LabTheme.background)
        .navigationTitle("Live fleet")
        .navigationBarTitleDisplayMode(.inline)
    }
}
