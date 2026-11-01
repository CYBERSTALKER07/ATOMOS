import SwiftUI

struct ShelfAlertModal: View {
    let productId: String
    let currentStock: Int64
    let onDismiss: () -> Void

    var body: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 44))
                .foregroundColor(AppTheme.destructive)
                .accessibilityHidden(true)
            
            Text("Restock Needed")
                .font(.title2)
                .bold()
                .foregroundColor(AppTheme.destructive)
            
            Text("Product \(productId) has dropped to \(currentStock) items on the floor.")
                .font(.subheadline)
                .multilineTextAlignment(.center)
                .foregroundColor(AppTheme.textSecondary)
                .monospacedDigit()
            
            Button(action: {
                Haptics.selection()
                onDismiss()
            }) {
                Text("Acknowledge")
                    .font(.headline)
                    .foregroundColor(AppTheme.cardBackground)
                    .frame(maxWidth: .infinity)
                    .pegasusMinimumHitTarget(minHeight: AppTheme.minTouchTarget)
                    .background(AppTheme.destructive)
                    .cornerRadius(AppTheme.radiusSM)
            }
            .accessibilityLabel("Acknowledge Restock Alert for Product \(productId)")
            .accessibilityHint("Dismisses the low stock alert")
            .padding(.top, AppTheme.spacingSM)
        }
        .padding(AppTheme.spacingXL)
        .background(AppTheme.secondaryBackground)
        .cornerRadius(AppTheme.radiusLG)
        .overlay(
            RoundedRectangle(cornerRadius: AppTheme.radiusLG)
                .stroke(AppTheme.destructive.opacity(0.3), lineWidth: 1)
        )
        .shadow(color: AppTheme.shadowColorLG, radius: 10, y: 4)
        .padding(AppTheme.spacingXL)
        .accessibilityElement(children: .contain)
    }
}
