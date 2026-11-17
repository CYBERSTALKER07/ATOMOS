import SwiftUI

struct FiscalFailedView: View {
    var subtitle: String? = nil

    var body: some View {
        VStack(spacing: 0) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(.largeTitle))
                .foregroundStyle(LabTheme.warning)
                .padding(.bottom, LabTheme.s16)
            Text(L10n.string("mobile_driver.ui.fiscal_retry_queued_title"))
                .font(.system(.title2, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .multilineTextAlignment(.center)
                .padding(.bottom, LabTheme.s8)
            Text(subtitle ?? L10n.string("mobile_driver.ui.fiscal_retry_queued_body"))
                .font(.system(.footnote, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
    }
}
