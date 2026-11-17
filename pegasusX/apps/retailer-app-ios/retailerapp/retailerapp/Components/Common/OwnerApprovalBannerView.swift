import SwiftUI

struct OwnerApprovalBannerView: View {
    let detail: OwnerApprovalRequiredDetail
    let isSubmitting: Bool
    let onPayCash: () -> Void
    let onPayCard: () -> Void
    let onDismiss: () -> Void

    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack(alignment: .center) {
                    Label {
                        Text(String(localized: "mobile_retailer.approval.sent_banner_title"))
                            .font(.system(.subheadline, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                    } icon: {
                        Image(systemName: "person.badge.shield.checkmark.fill")
                            .foregroundStyle(AppTheme.warning)
                    }
                    Spacer()
                    Button("Dismiss", action: onDismiss)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                        .disabled(isSubmitting)
                }

                Text(detail.message)
                    .font(.caption)
                    .foregroundStyle(AppTheme.textSecondary)

                HStack(spacing: AppTheme.spacingMD) {
                    Text(detail.formattedAmount)
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text(detail.readableRuleLabel)
                        .font(.caption)
                        .foregroundStyle(AppTheme.textTertiary)
                }

                if detail.formattedThreshold != nil || detail.formattedRemainingCap != nil {
                    let parts = [
                        detail.formattedThreshold.map { "Limit: \($0)" },
                        detail.formattedRemainingCap.map { "Remaining: \($0)" },
                    ].compactMap { $0 }
                    Text(parts.joined(separator: " · "))
                        .font(.caption2)
                        .foregroundStyle(AppTheme.textTertiary)
                }

                if detail.allowsCashFallback || detail.allowsCardFallback {
                    HStack(spacing: AppTheme.spacingSM) {
                        if detail.allowsCashFallback {
                            LabButton(
                                String(localized: "mobile_retailer.approval.pay_cash_instead"),
                                icon: "banknote",
                                fullWidth: true
                            ) {
                                onPayCash()
                            }
                            .disabled(isSubmitting)
                        }
                        if detail.allowsCardFallback {
                            Button(action: onPayCard) {
                                HStack(spacing: 6) {
                                    Image(systemName: "creditcard")
                                    Text(String(localized: "mobile_retailer.approval.pay_card_instead"))
                                }
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .background(AppTheme.surfaceElevated)
                                .foregroundStyle(AppTheme.textPrimary)
                                .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusMD, style: .continuous))
                            }
                            .disabled(isSubmitting)
                        }
                    }
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }
}
