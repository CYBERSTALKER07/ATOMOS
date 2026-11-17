import SwiftUI

struct CreditProfileSection: View {
    let profile: CreditProfile?
    let isLoading: Bool
    let missing: Bool
    let error: String?

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack(spacing: 8) {
                Image(systemName: "creditcard.fill")
                    .foregroundStyle(AppTheme.accent)
                Text("retailer_desktop.credit_profile_card.text.supplier_credit")
                    .font(.system(.subheadline, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
                Spacer()
                if let profile {
                    Text(profile.status.uppercased())
                        .font(.system(.caption2, design: .rounded, weight: .bold))
                        .foregroundStyle(statusColor(profile.status))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(statusColor(profile.status).opacity(0.12))
                        .clipShape(Capsule())
                }
            }

            if isLoading {
                HStack(spacing: 8) {
                    ProgressView()
                        .controlSize(.small)
                    Text("mobile_retailer.ui.loading_credit")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            } else if missing {
                Text("mobile_retailer.ui.no_credit_line_on_file_for_this_supplier_relationship")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            } else if let error {
                Text(error)
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.error)
            } else if let profile {
                HStack(spacing: 12) {
                    KPICard(
                        title: "Limit",
                        value: RetailerMoney.formatWholeCurrency(profile.creditLimitMinor),
                        subtitle: "Credit limit"
                    )
                    KPICard(
                        title: "Balance due",
                        value: RetailerMoney.formatWholeCurrency(profile.currentBalanceMinor),
                        subtitle: "Outstanding"
                    )
                    KPICard(
                        title: "Available",
                        value: RetailerMoney.formatWholeCurrency(profile.availableCreditMinor),
                        subtitle: "Remaining"
                    )
                }

                HStack {
                    let util: String = {
                        guard profile.creditLimitMinor > 0 else { return "0.0" }
                        let tenths = (profile.currentBalanceMinor * 1_000) / profile.creditLimitMinor
                        return "\(tenths / 10).\(abs(tenths % 10))"
                    }()
                    Text(
                        profile.riskTier.map { "Utilization \(util)% · risk \($0)" }
                            ?? "Utilization \(util)%"
                    )
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    Spacer()
                    if let delinquency = profile.delinquencyCount, delinquency > 0 {
                        Text(L10n.format("mobile_retailer.ui.delinquency_delinquency_2", "\(delinquency)"))
                            .font(.system(.caption2, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.error)
                    }
                }
            }
        }
        .padding(AppTheme.spacingLG)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }

    private func statusColor(_ status: String) -> Color {
        switch status.uppercased() {
        case "FROZEN", "BLACKLISTED":
            return AppTheme.warning
        case "ACTIVE":
            return AppTheme.success
        default:
            return AppTheme.textTertiary
        }
    }
}
