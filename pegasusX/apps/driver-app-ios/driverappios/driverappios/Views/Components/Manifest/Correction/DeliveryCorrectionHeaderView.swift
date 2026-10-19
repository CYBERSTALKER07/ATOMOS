import SwiftUI

struct DeliveryCorrectionHeaderView: View {
    let orderId: String
    let isPartial: Bool
    let hasRejections: Bool
    let rejectedCount: Int
    let onClose: () -> Void
    @Binding var showStartTransitAlert: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Button {
                    Haptics.light()
                    onClose()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "chevron.left")
                        Text("common.action.back")
                    }
                    .font(.body.weight(.semibold))
                    .foregroundStyle(LabTheme.fg)
                }

                Spacer()

                StatusPill(
                    label: hasRejections ? "\(rejectedCount) REJECTED" : "ALL CLEAR",
                    color: hasRejections ? LabTheme.warning : LabTheme.success
                )
            }

            Text("mobile_driver.ui.delivery_correction")
                .font(.system(size: 22, weight: .bold))
                .foregroundStyle(LabTheme.fg)

            Text(orderId)
                .font(.system(size: 13, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)

            if isPartial {
                VStack(alignment: .leading, spacing: 12) {
                    Text("mobile_driver.ui.partial_order_split")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                    
                    Text("mobile_driver.ui.this_order_is_split_across_multiple_trucks_press_start_transit_w")
                        .font(.system(size: 14))
                        .foregroundStyle(LabTheme.fgSecondary)
                    
                    Button {
                        showStartTransitAlert = true
                    } label: {
                        Text("mobile_driver.ui.start_transit_2")
                            .font(.system(size: 15, weight: .bold))
                            .foregroundStyle(.white)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 12)
                            .background(LabTheme.success, in: .rect(cornerRadius: LabTheme.buttonRadius))
                    }
                    .buttonStyle(.pressable)
                }
                .padding(LabTheme.s16)
                .labCard()
                .padding(.top, 8)
            }
        }
    }
}
