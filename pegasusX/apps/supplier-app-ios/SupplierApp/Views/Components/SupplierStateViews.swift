import SwiftUI

typealias SupplierLoadingView = PegasusLoadingView
typealias SupplierErrorView = PegasusErrorView
typealias SupplierEmptyView = PegasusEmptyView

enum MoneyFormat {
    static func minor(_ amount: Int64, currency: String) -> String {
        let number = formatMinorUnits(amount, fractionDigits: 2, trimTrailingZeros: false)
        let code = currency.isEmpty ? packCurrency(MarketPackStore.pack) : currency
        return "\(number) \(code)"
    }
}
