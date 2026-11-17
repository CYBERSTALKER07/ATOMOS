import SwiftUI

struct CheckoutPaymentPickerSheet: View {
    let paymentOptions: [CheckoutPaymentOption]
    let selectedPaymentId: String
    let onSelect: (CheckoutPaymentOption) -> Void

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(paymentOptions, id: \.id) { option in
                Button {
                    onSelect(option)
                } label: {
                    HStack(spacing: AppTheme.spacingMD) {
                        Image(systemName: paymentIcon(option))
                            .font(.callout.weight(.medium))
                            .foregroundStyle(AppTheme.accent)
                            .frame(width: 24)

                        Text(option.label)
                            .font(.system(.body, design: .rounded))
                            .foregroundStyle(AppTheme.textPrimary)

                        Spacer()

                        if option.id == selectedPaymentId {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundStyle(AppTheme.accent)
                        }
                    }
                    .pegasusMinimumHitTarget()
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel("\(option.label)\(option.id == selectedPaymentId ? ", selected" : "")")
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("mobile_retailer.ui.payment_method")
        .navigationBarTitleDisplayMode(.inline)
    }

    private func paymentIcon(_ option: CheckoutPaymentOption) -> String {
        if option.isToken { return "creditcard.fill" }
        switch option.id {
        case "GlobalPay": return "wallet.pass"
        case "Cash": return "banknote"
        default: return "creditcard"
        }
    }
}
