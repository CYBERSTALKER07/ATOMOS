import SwiftUI

struct FiscalFailedView: View {
    var body: some View {
        VStack(spacing: 0) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 64))
                .foregroundStyle(LabTheme.destructive)
                .padding(.bottom, LabTheme.s16)
            Text("supplier_portal.compliance.text.fiscal_failed")
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)
            Text("mobile_driver.ui.retry_fiscal_receipt_or_call_supervisor_for_force_complete")
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
    }
}
