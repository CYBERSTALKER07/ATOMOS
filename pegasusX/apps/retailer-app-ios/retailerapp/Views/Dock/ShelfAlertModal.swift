import SwiftUI

struct ShelfAlertModal: View {
    let productId: String
    let currentStock: Int64
    let onDismiss: () -> Void

    var body: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.largeTitle)
                .foregroundStyle(AppTheme.destructive)
                .accessibilityHidden(true)
            
            Text("Restock Needed")
                .font(.title2)
                .bold()
                .foregroundStyle(AppTheme.destructive)
            
            Text("Product \(productId) has dropped to \(currentStock) items on the floor.")
                .font(.subheadline)
                .multilineTextAlignment(.center)
                .foregroundStyle(AppTheme.textSecondary)
                .monospacedDigit()
            
            Button(action: {
                Haptics.selection()
                onDismiss()
            }) {
                Text("Acknowledge")
                    .font(.headline)
                    .foregroundStyle(AppTheme.cardBackground)
                    .frame(maxWidth: .infinity)
                    .pegasusMinimumHitTarget(minHeight: AppTheme.minTouchTarget)
                    .background(AppTheme.destructive)
                    .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusSM, style: .continuous))
            }
            .accessibilityLabel("Acknowledge Restock Alert for Product \(productId)")
            .accessibilityHint("Dismisses the low stock alert")
            .padding(.top, AppTheme.spacingSM)
        }
        .padding(AppTheme.spacingXL)
        .background(AppTheme.secondaryBackground)
        .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusLG, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: AppTheme.radiusLG, style: .continuous)
                .stroke(AppTheme.destructive.opacity(0.3), lineWidth: 1)
        )
        .shadow(color: AppTheme.shadowColorLG, radius: 10, y: 4)
        .padding(AppTheme.spacingXL)
        .accessibilityElement(children: .combine)
    }
}
