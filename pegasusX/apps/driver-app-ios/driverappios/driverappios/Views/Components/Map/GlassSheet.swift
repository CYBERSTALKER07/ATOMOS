import SwiftUI

struct GlassSheet: View {
    @Environment(\.colorScheme) private var cs
    
    var body: some View {
        ZStack {
            UnevenRoundedRectangle(topLeadingRadius: LabTheme.cardRadius, topTrailingRadius: LabTheme.cardRadius, style: .continuous).fill(.ultraThinMaterial)
            UnevenRoundedRectangle(topLeadingRadius: LabTheme.cardRadius, topTrailingRadius: LabTheme.cardRadius, style: .continuous).fill(LabTheme.card.opacity(0.6))
            UnevenRoundedRectangle(topLeadingRadius: LabTheme.cardRadius, topTrailingRadius: LabTheme.cardRadius, style: .continuous).stroke(LabTheme.separator, lineWidth: 0.5)
        }
        .shadow(color: .black.opacity(cs == .dark ? 0.6 : 0.1), radius: 30, y: -8)
    }
}
