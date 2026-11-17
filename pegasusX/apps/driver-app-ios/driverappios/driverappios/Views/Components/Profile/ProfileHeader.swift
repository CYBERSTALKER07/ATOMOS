import SwiftUI

struct ProfileHeader: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("DRIVER")
                .font(.system(.caption2, design: .monospaced, weight: .heavy))
                .foregroundStyle(LabTheme.fgTertiary)
                .tracking(1.2)

            Text("portal.nav.profile")
                .font(.system(.title, weight: .bold))
                .foregroundStyle(LabTheme.fg)
        }
        .padding(.top, 60)
        .padding(.horizontal, LabTheme.s4)
    }
}
