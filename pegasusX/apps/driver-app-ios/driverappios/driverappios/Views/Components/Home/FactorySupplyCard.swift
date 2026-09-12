import SwiftUI

struct FactorySupplyCard: View {
    @Binding var showSupplyTransfers: Bool

    var body: some View {
        Button {
            Haptics.medium()
            showSupplyTransfers = true
        } label: {
            HStack(spacing: 14) {
                Image(systemName: "shippingbox.fill")
                    .font(.system(size: 22, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
                VStack(alignment: .leading, spacing: 3) {
                    Text("mobile_driver.ui.supply_transfers")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                    Text("\(TokenStore.shared.factoryName ?? "Factory depot") → warehouse legs")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(LabTheme.s16)
            .labCard()
        }
        .buttonStyle(.pressable)
    }
}
