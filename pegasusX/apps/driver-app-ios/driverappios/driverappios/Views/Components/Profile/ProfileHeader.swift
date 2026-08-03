import SwiftUI

struct ProfileHeader: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("DRIVER")
                .font(.system(size: 10, weight: .heavy, design: .monospaced))
                .foregroundStyle(LabTheme.fgTertiary)
                .tracking(1.2)

            Text("Profile")
                .font(.system(size: 28, weight: .bold))
                .foregroundStyle(LabTheme.fg)
        }
        .padding(.top, 60)
        .padding(.horizontal, LabTheme.s4)
    }
}
