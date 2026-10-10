import SwiftUI

struct DeliveryCorrectionLineItemCard: View {
    let item: LineItem
    let currentReason: RejectionReason
    let onToggleStatus: () -> Void
    let onSetReason: (RejectionReason) -> Void
    
    private func reasonLabel(for reason: RejectionReason) -> String {
        reason.rawValue.replacingOccurrences(of: "_", with: " ").capitalized
    }
    
    var body: some View {
        let isRejected = item.status == .REJECTED_DAMAGED

        return VStack(alignment: .leading, spacing: 10) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(item.sku_id)
                        .font(.system(size: 15, weight: .bold))
                        .strikethrough(isRejected)
                        .foregroundStyle(isRejected ? LabTheme.destructive : LabTheme.fg)

                    Text(L10n.format("mobile_driver.ui.quantity_formattedamount", "\(item.quantity)", "\(item.unit_price.formattedAmount)"))
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                Text(isRejected ? "Rejected" : "Delivered")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(
                        isRejected ? LabTheme.destructive : LabTheme.fg,
                        in: Capsule()
                    )
            }

            Rectangle()
                .fill(LabTheme.separator)
                .frame(height: 0.5)

            HStack {
                Text("mobile_driver.ui.line_total")
                    .font(.caption)
                    .foregroundStyle(LabTheme.fgTertiary)
                Spacer()
                Text(item.lineTotal.formattedAmount)
                    .font(.system(size: 14, weight: .bold))
                    .strikethrough(isRejected)
                    .foregroundStyle(isRejected ? LabTheme.destructive.opacity(0.6) : LabTheme.fg)
            }

            if isRejected {
                HStack {
                    Text("supplier_portal.admin.control_center.field.reason")
                        .font(.caption)
                        .foregroundStyle(LabTheme.fgTertiary)
                    Spacer()
                    Menu {
                        ForEach(RejectionReason.allCases, id: \.self) { reason in
                            Button {
                                onSetReason(reason)
                            } label: {
                                if currentReason == reason {
                                    Label(reasonLabel(for: reason), systemImage: "checkmark")
                                } else {
                                    Text(reasonLabel(for: reason))
                                }
                            }
                        }
                    } label: {
                        HStack(spacing: 6) {
                            Text(reasonLabel(for: currentReason))
                            Image(systemName: "chevron.down")
                                .font(.system(size: 11, weight: .bold))
                        }
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(LabTheme.fg)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 8)
                        .background(LabTheme.fg.opacity(0.06), in: Capsule())
                    }
                }
            }

            RoundedRectangle(cornerRadius: 2)
                .fill(isRejected ? LabTheme.destructive : LabTheme.fg.opacity(0.15))
                .frame(height: 3)
        }
        .padding(LabTheme.s16)
        .labCard()
        .contentShape(Rectangle())
        .onTapGesture {
            onToggleStatus()
        }
    }
}
