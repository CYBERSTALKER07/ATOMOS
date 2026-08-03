import SwiftUI

struct ManifestEmptyView: View {
    var body: some View {
        DriverEmptyView(
            icon: "road.lanes",
            title: "No upcoming rides",
            message: "Pull to refresh or check back later."
        )
        .frame(maxWidth: .infinity)
        .padding(.horizontal, LabTheme.s16)
        .padding(.vertical, 60)
    }
}
