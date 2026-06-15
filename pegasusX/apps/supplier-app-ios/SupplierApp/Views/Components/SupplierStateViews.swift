import SwiftUI

struct SupplierLoadingView: View {
    let title: String
    var message: String = "Fetching the latest supplier data."

    @State private var animating = false

    var body: some View {
        VStack(spacing: SupplierTheme.spacingLG) {
            ZStack {
                Circle()
                    .fill(SupplierTheme.tertiaryBackground)
                    .frame(width: 72, height: 72)
                    .scaleEffect(animating ? 1.04 : 0.96)
                ProgressView()
                    .controlSize(.regular)
            }

            VStack(spacing: SupplierTheme.spacingSM) {
                Text(title)
                    .font(.title3.bold())
                Text(message)
                    .font(.body)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(SupplierTheme.spacingXL)
        .onAppear {
            withAnimation(SupplierAnim.smooth.repeatForever(autoreverses: true)) {
                animating = true
            }
        }
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
                .buttonStyle(.borderedProminent)
        }
        .frame(maxWidth: .infinity, minHeight: 200)
    }
}

struct SupplierEmptyView: View {
    let title: String
    let message: String

    var body: some View {
        ContentUnavailableView(title, systemImage: "tray", description: Text(message))
            .frame(maxWidth: .infinity, minHeight: 200)
    }
}

enum MoneyFormat {
    static func minor(_ amount: Int64, currency: String) -> String {
        let major = Double(amount) / 100.0
        let code = currency.isEmpty ? "UZS" : currency
        return String(format: "%.2f %@", major, code)
    }
}
