import SwiftUI

struct OffloadActionFooter: View {
    let orderId: String
    let hasRejections: Bool
    let isSubmitting: Bool
    let onShopClosed: ((String) -> Void)?
    let onCreditDelivery: ((String) -> Void)?
    let onReportMissing: ((String) -> Void)?
    let onConfirm: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            // MARK: - Shop Closed Button
            if let onShopClosed {
                Button {
                    onShopClosed(orderId)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "door.left.hand.closed")
                            .font(.system(size: 14, weight: .semibold))
                        Text("Shop Closed / No Answer")
                            .font(.system(size: 15, weight: .bold))
                    }
                    .foregroundStyle(Color.orange)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(Color.orange.opacity(0.12), in: .rect(cornerRadius: LabTheme.buttonRadius))
                    .overlay(
                        RoundedRectangle(cornerRadius: LabTheme.buttonRadius)
                            .stroke(Color.orange.opacity(0.3), lineWidth: 1)
                    )
                }
                .disabled(isSubmitting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Edge 32: Credit Delivery Button
            if let onCreditDelivery {
                Button {
                    onCreditDelivery(orderId)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "creditcard.fill")
                            .font(.system(size: 14, weight: .semibold))
                        Text("Deliver on Credit")
                            .font(.system(size: 15, weight: .bold))
                    }
                    .foregroundStyle(.blue)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(Color.blue.opacity(0.12), in: .rect(cornerRadius: LabTheme.buttonRadius))
                    .overlay(
                        RoundedRectangle(cornerRadius: LabTheme.buttonRadius)
                            .stroke(Color.blue.opacity(0.3), lineWidth: 1)
                    )
                }
                .disabled(isSubmitting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Edge 33: Missing Items Button
            if let onReportMissing, hasRejections {
                Button {
                    onReportMissing(orderId)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .font(.system(size: 14, weight: .semibold))
                        Text("Report Missing Items")
                            .font(.system(size: 15, weight: .bold))
                    }
                    .foregroundStyle(LabTheme.destructive)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(LabTheme.destructive.opacity(0.12), in: .rect(cornerRadius: LabTheme.buttonRadius))
                    .overlay(
                        RoundedRectangle(cornerRadius: LabTheme.buttonRadius)
                            .stroke(LabTheme.destructive.opacity(0.3), lineWidth: 1)
                    )
                }
                .disabled(isSubmitting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Confirm Offload Button
            Button {
                onConfirm()
            } label: {
                HStack(spacing: 8) {
                    if isSubmitting {
                        ProgressView().tint(LabTheme.buttonFg)
                    }
                    Text("Confirm Offload")
                        .font(.system(size: 15, weight: .bold))
                }
                .foregroundStyle(LabTheme.buttonFg)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 16)
                .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
            }
            .disabled(isSubmitting)
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s24)
        }
    }
}
