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
                        Text("Back")
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

            Text("Delivery Correction")
                .font(.system(size: 22, weight: .bold))
                .foregroundStyle(LabTheme.fg)

            Text(orderId)
                .font(.system(size: 13, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)

            if isPartial {
                VStack(alignment: .leading, spacing: 12) {
                    Text("Partial Order Split")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                    
                    Text("This order is split across multiple trucks. Press Start Transit when you are heading to this route to notify the other driver.")
                        .font(.system(size: 14))
                        .foregroundStyle(LabTheme.fgSecondary)
                    
                    Button {
                        showStartTransitAlert = true
                    } label: {
                        Text("Start Transit")
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
