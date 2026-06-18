import SwiftUI

typealias SupplierTheme = PegasusMonochromeTheme

typealias SupplierAnim = PegasusAnim

struct SupplierCardModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding(SupplierTheme.spacingLG)
            .background(SupplierTheme.card)
            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG, style: .continuous))
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
