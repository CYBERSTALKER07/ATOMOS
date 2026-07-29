import SwiftUI

struct CollectCashView: View {
    let orderId: String
    let amount: Int
    @Binding var amountReceivedText: String
    let shortfallNote: String?
    
    var body: some View {
        VStack(spacing: 0) {
            Image(systemName: "banknote.fill")
                .font(.system(size: 64))
                .foregroundStyle(LabTheme.success)
                .padding(.bottom, LabTheme.s16)
            Text("Collect Cash")
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)
            Text(orderId)
                .font(.system(size: 15, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)
                .padding(.bottom, LabTheme.s16)
            Text("Expected \(amount.formattedAmount)")
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(LabTheme.fgTertiary)
                .padding(.bottom, LabTheme.s8)
            Text(receivedAmountDisplay.formattedAmount)
                .font(.system(size: 42, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)
            TextField("Amount received (tiyin)", text: $amountReceivedText)
                .keyboardType(.numberPad)
                .textFieldStyle(.roundedBorder)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            if let shortfallNote {
                Text(shortfallNote)
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }
            Text("Enter cash actually taken. Fiscal receipt uses this amount.")
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
    }
    
    private var receivedAmountDisplay: Int {
        Int(amountReceivedText.filter(\.isNumber)) ?? amount
    }
}
