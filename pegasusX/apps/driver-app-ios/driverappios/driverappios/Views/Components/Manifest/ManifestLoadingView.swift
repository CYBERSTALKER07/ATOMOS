import SwiftUI

struct ManifestLoadingView: View {
    var body: some View {
        DriverLoadingView(
            title: "Loading routes",
            message: "Checking manifest state, sequence, and delivery assignments."
        )
        .frame(maxWidth: .infinity)
        .padding(.horizontal, LabTheme.s16)
        .padding(.vertical, 60)
    }
}
