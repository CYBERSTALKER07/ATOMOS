import SwiftUI

struct SupplierLoadingView: View {
    let title: String

    var body: some View {
        VStack(spacing: SupplierTheme.spacingLG) {
            ProgressView()
            Text(title)
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, minHeight: 200)
    }
}

struct SupplierErrorView: View {
    let message: String
    let retry: () -> Void

    var body: some View {
        ContentUnavailableView {
            Label("Unable to load", systemImage: "exclamationmark.triangle")
        } description: {
            Text(message)
        } actions: {
            Button("Retry", action: retry)
        }
    }
}

struct SupplierEmptyView: View {
    let title: String
    let message: String

    var body: some View {
        ContentUnavailableView(title, systemImage: "tray", description: Text(message))
    }
}

enum MoneyFormat {
    static func minor(_ amount: Int64, currency: String) -> String {
        let major = Double(amount) / 100.0
        let code = currency.isEmpty ? "UZS" : currency
        return String(format: "%.2f %@", major, code)
    }
}
