import SwiftUI

struct FiscalizingView: View {
    let amount: Int
    
    var body: some View {
        VStack(spacing: 0) {
            Image(systemName: "hourglass")
                .font(.system(.largeTitle))
                .foregroundStyle(LabTheme.warning)
                .padding(.bottom, LabTheme.s16)
            Text("supplier_portal.compliance.text.fiscalizing")
                .font(.system(.title2, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)
            Text(amount.formattedAmount)
                .font(.system(.largeTitle, design: .monospaced, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s16)
            ProgressView()
                .scaleEffect(1.2)
                .padding(.bottom, LabTheme.s8)
            Text("mobile_driver.ui.cash_captured_waiting_for_fiscal_receipt")
                .font(.system(.footnote, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
    }
}
