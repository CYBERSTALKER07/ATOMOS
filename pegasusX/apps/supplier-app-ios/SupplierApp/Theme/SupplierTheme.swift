import SwiftUI

typealias SupplierTheme = PegasusMonochromeTheme

typealias SupplierAnim = PegasusAnim
typealias SupplierPressableButtonStyle = PegasusPressableButtonStyle

extension ButtonStyle where Self == PegasusPressableButtonStyle {
    static var pressable: PegasusPressableButtonStyle { PegasusPressableButtonStyle() }
}

struct SupplierCardModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding(SupplierTheme.spacingLG)
            .background(SupplierTheme.card)
            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: SupplierTheme.radiusLG, style: .continuous)
                    .stroke(SupplierTheme.cardBorder, lineWidth: 1)
            )
            .shadow(color: Color.black.opacity(0.03), radius: 6, x: 0, y: 2)
    }
}

extension View {
    func supplierCard() -> some View {
        modifier(SupplierCardModifier())
    }

    func supplierReadableWidth() -> some View {
        frame(maxWidth: SupplierTheme.readableMaxWidth)
            .frame(maxWidth: .infinity)
    }
}
