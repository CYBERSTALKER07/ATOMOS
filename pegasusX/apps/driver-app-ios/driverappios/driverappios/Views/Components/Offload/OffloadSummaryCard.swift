import SwiftUI

struct OffloadSummaryCard: View {
    let retailerName: String
    let totalAmount: Double
    
    var body: some View {
        HStack {
            Text(retailerName)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(LabTheme.fg)
            Spacer()
            Text(totalAmount.formattedAmount)
                .font(.system(size: 15, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
        }
        .padding(.horizontal, LabTheme.s24)
        .padding(.bottom, LabTheme.s16)
    }
}
