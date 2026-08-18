import SwiftUI

struct PaymentConfigView: View {
    @State private var listings: [PSPListing] = []
    @State private var currency = ""
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { load() }
                }
            } else if listings.isEmpty {
                ContentUnavailableView("No Gateways", systemImage: "creditcard", description: Text("warehouse_portal.payment_config.text.no_payment_gateways_configured"))
            } else {
                List {
                    if !currency.isEmpty {
                        Section {
                            LabeledContent("Pack currency", value: currency)
                        }
                    }
                    Section("Pack payment catalog") {
                        ForEach(listings) { listing in
                            HStack {
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    Text(listing.code)
                                        .font(.headline)
                                    Text(listing.status)
                                        .font(.subheadline)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Image(systemName: listing.selectable ? "checkmark.circle.fill" : "xmark.circle")
                                    .foregroundStyle(listing.selectable ? .green : .secondary)
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("warehouse_portal.treasury.text.payment_config")
        .task { load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.paymentConfig()
                listings = resp.catalog.isEmpty
                    ? resp.gateways.map { PSPListing(code: $0.code, status: $0.status, selectable: $0.selectable) }
                    : resp.catalog
                currency = resp.currencyCode
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
