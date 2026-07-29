import SwiftUI

struct FiscalizingView: View {
    let amount: Int
    
    var body: some View {
        VStack(spacing: 0) {
            Image(systemName: "hourglass")
                .font(.system(size: 64))
                .foregroundStyle(LabTheme.warning)
                .padding(.bottom, LabTheme.s16)
            Text("Fiscalizing")
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)
            Text(amount.formattedAmount)
                .font(.system(size: 36, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s16)
            ProgressView()
                .scaleEffect(1.2)
                .padding(.bottom, LabTheme.s8)
            Text("Cash captured. Waiting for fiscal receipt…")
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
    }
}
