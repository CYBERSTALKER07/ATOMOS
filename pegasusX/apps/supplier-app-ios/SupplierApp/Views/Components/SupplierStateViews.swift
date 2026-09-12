import SwiftUI

typealias SupplierLoadingView = PegasusLoadingView
typealias SupplierErrorView = PegasusErrorView
typealias SupplierEmptyView = PegasusEmptyView

enum MoneyFormat {
    static func minor(_ amount: Int64, currency: String) -> String {
        let major = Double(amount) / 100.0
        let code = currency.isEmpty ? packCurrency(MarketPackStore.pack) : currency
        return String(format: "%.2f %@", major, code)
    }
}
