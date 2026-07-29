import SwiftUI

struct FiscalFailedView: View {
    var body: some View {
        VStack(spacing: 0) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 64))
                .foregroundStyle(LabTheme.destructive)
                .padding(.bottom, LabTheme.s16)
            Text("Fiscal Failed")
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)
            Text("Retry fiscal receipt or call supervisor for force-complete.")
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
    }
}
