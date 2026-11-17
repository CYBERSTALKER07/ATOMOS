import SwiftUI

struct CollectCashView: View {
    let orderId: String
    let amount: Int
    @Binding var amountReceivedText: String
    let shortfallNote: String?
    
    var body: some View {
        VStack(spacing: 0) {
            Image(systemName: "banknote.fill")
                .font(.system(.largeTitle))
                .foregroundStyle(LabTheme.success)
                .padding(.bottom, LabTheme.s16)
            Text("mobile_driver.ui.collect_cash_2")
                .font(.system(.title2, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)
            Text(orderId)
                .font(.system(.subheadline, design: .monospaced, weight: .semibold))
                .foregroundStyle(LabTheme.fgSecondary)
                .padding(.bottom, LabTheme.s16)
            Text(L10n.format("mobile_driver.ui.expected_formattedamount", "\(amount.formattedAmount)"))
                .font(.system(.subheadline, weight: .semibold))
                .foregroundStyle(LabTheme.fgTertiary)
                .padding(.bottom, LabTheme.s8)
            Text(receivedAmountDisplay.formattedAmount)
                .font(.system(.largeTitle, design: .monospaced, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)
            TextField("mobile_driver.ui.amount_received_tiyin", text: $amountReceivedText)
                .keyboardType(.numberPad)
                .textFieldStyle(.roundedBorder)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            if let shortfallNote {
                Text(shortfallNote)
                    .font(.system(.caption, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }
            Text("mobile_driver.ui.enter_cash_actually_taken_fiscal_receipt_uses_this_amount")
                .font(.system(.footnote, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
    }
    
    private var receivedAmountDisplay: Int {
        Int(amountReceivedText.filter(\.isNumber)) ?? amount
    }
}
