import SwiftUI

enum CRMDetailLoad {
    struct Outcome<T> {
        let value: T?
        let failure: Error?
    }

    /// A failed detail fetch keeps the detail already on screen and returns the error.
    static func selection<T, Failure: Error>(
        previous: T?,
        loaded: Result<T, Failure>
    ) -> Outcome<T> {
        switch loaded {
        case .success(let value):
            return Outcome(value: value, failure: nil)
        case .failure(let error):
            return Outcome(value: previous, failure: error)
        }
    }
}

struct CRMView: View {
    @State private var vm = CRMViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.retailers.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.crm")
            } else if let error = vm.error, vm.retailers.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.retailers.isEmpty {
                Text("mobile_supplier.chrome.no_retailers")
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List {
                    ForEach(vm.retailers) { row in
                        Button {
                            PegasusHaptics.selection()
                            Task { await vm.loadDetail(row.retailerId) }
                        } label: {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(row.retailerName.isEmpty ? row.retailerId : row.retailerName)
                                    .font(.headline)
                                Text(L10n.format("mobile_supplier.chrome.retailer_lifetime_orders", row.status, MissingNumber.label(row.lifetime), "\(row.orderCount)"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if !row.phone.isEmpty {
                                    Text(row.phone)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                if !row.email.isEmpty {
                                    Text(row.email)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                            .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                        .accessibilityElement(children: .combine)
                        .accessibilityAddTraits(.isButton)
                        .accessibilityLabel("\(row.retailerName.isEmpty ? row.retailerId : row.retailerName), status \(row.status), \(row.orderCount) orders")
                    }
                    if let selected = vm.selected {
                        Section("mobile_supplier.chrome.detail") {
                            Text(selected.retailerName)
                                .font(.headline)
                            if !selected.email.isEmpty {
                                Text(selected.email)
                                    .font(.caption)
                            }
                            Text(L10n.format("mobile_supplier.chrome.lifetime_minor", MissingNumber.label(selected.lifetime)))
                                .font(.caption)
                            ForEach(selected.orders) { order in
                                VStack(alignment: .leading, spacing: 2) {
                                    Text("\(order.orderId.prefix(8)) · \(order.state) · \(Self.formatCRMOrderAmount(order.amount))")
                                        .font(.caption)
                                    ForEach(order.lines) { line in
                                        Text("\(line.productName.isEmpty ? line.sku : line.productName) × \(line.qty) · \(Self.formatCRMOrderLineAmount(line.amountMinor))")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    }
                                }
                                .padding(.vertical, 2)
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("Order \(order.orderId.prefix(8)), \(order.state), \(Self.formatCRMOrderAmount(order.amount))")
                            }
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.crm")
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
    }

    static func formatCRMOrderAmount(_ amount: Int64?, currency: String = packCurrency(MarketPackStore.pack)) -> String {
        amount.map { MoneyFormat.minor($0, currency: currency) } ?? MissingNumber.label(nil as Int64?)
    }

    static func formatCRMOrderLineAmount(_ amountMinor: Int64?, currency: String = packCurrency(MarketPackStore.pack)) -> String {
        amountMinor.map { MoneyFormat.minor($0, currency: currency) } ?? MissingNumber.label(nil as Int64?)
    }
}
