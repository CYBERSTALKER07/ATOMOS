import SwiftUI

struct OffloadSummaryCard: View {
    let retailerName: String
    let totalAmount: Int
    
    var body: some View {
        HStack {
            Text(retailerName)
                .font(.system(.subheadline, weight: .semibold))
                .foregroundStyle(LabTheme.fg)
            Spacer()
            Text(totalAmount.formattedAmount)
                .font(.system(.subheadline, design: .monospaced, weight: .bold))
                .foregroundStyle(LabTheme.fg)
        }
        .padding(.horizontal, LabTheme.s24)
        .padding(.bottom, LabTheme.s16)
    }
}
