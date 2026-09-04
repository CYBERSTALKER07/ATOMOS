import SwiftUI

struct ActiveCountBadge: View {
    let count: Int

    var body: some View {
        HStack(spacing: 4) {
            Image(systemName: "box.truck.fill")
                .font(.system(size: 12, weight: .semibold))
            Text(L10n.format("supplier_portal.admin.empathy.hierarchy.active_count", "\(count)"))
                .font(.system(.caption, design: .rounded, weight: .bold))
        }
        .foregroundStyle(AppTheme.textPrimary)
        .padding(.horizontal, AppTheme.spacingMD)
        .padding(.vertical, AppTheme.spacingSM)
        .background(.ultraThinMaterial, in: .capsule)
    }
}
