import SwiftUI

struct ConnectSupplierSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var query = ""
    @State private var results: [Supplier] = []
    @State private var isSearching = false
    @State private var statusMessage: String?

    let existingSuppliers: [Supplier]
    let onConnected: () async -> Void

    private let api = APIClient.shared

    var body: some View {
        NavigationStack {
            List {
                Section {
                    TextField("mobile_retailer.ui.supplier_name", text: $query)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .onChange(of: query) { _, newValue in
                            Task { await search(newValue) }
                        }
                } footer: {
                    Text("mobile_retailer.ui.search_the_supplier_network_and_add_vendors_to_your_procurement_")
                }

                if isSearching {
                    Section {
                        HStack {
                            Spacer()
                            ProgressView()
                            Spacer()
                        }
                    }
                } else if query.trimmingCharacters(in: .whitespacesAndNewlines).count >= 2, results.isEmpty {
                    Section {
                        Text("mobile_retailer.ui.no_new_suppliers_match_that_search")
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                } else {
                    Section("Results") {
                        ForEach(results) { supplier in
                            HStack {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(supplier.name)
                                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                    if let category = supplier.displayCategory {
                                        Text(category)
                                            .font(.system(.caption, design: .rounded))
                                            .foregroundStyle(AppTheme.textTertiary)
                                    }
                                }
                                Spacer()
                                Button("mobile_retailer.ui.add") {
                                    Task { await connect(supplier.id) }
                                }
                                .buttonStyle(.borderedProminent)
                            }
                        }
                    }
                }

                if let statusMessage {
                    Section {
                        Text(statusMessage)
                            .font(.footnote)
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                }
            }
            .navigationTitle("mobile_retailer.ui.connect_vendor")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.close") { dismiss() }
                }
            }
        }
    }

    private func search(_ raw: String) async {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 2 else {
            results = []
            return
        }
        isSearching = true
        statusMessage = nil
        defer { isSearching = false }
        do {
            let found = try await RetailerSupplierDiscoveryService.searchSuppliers(api: api, query: trimmed)
            results = RetailerSupplierDiscoveryService.filterNewCandidates(found, existing: existingSuppliers)
        } catch {
            statusMessage = error.localizedDescription
            results = []
        }
    }

    private func connect(_ supplierId: String) async {
        statusMessage = nil
        do {
            try await RetailerSupplierDiscoveryService.addSupplier(api: api, supplierId: supplierId)
            await onConnected()
            dismiss()
        } catch {
            statusMessage = error.localizedDescription
        }
    }
}
