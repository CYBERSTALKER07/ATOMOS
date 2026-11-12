import SwiftUI

struct ConnectSupplierSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var vm = ConnectSupplierViewModel()

    let existingSuppliers: [Supplier]
    let onConnected: () async -> Void

    var body: some View {
        @Bindable var vm = vm

        List {
            Section {
                TextField("mobile_retailer.ui.supplier_name", text: $vm.query)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .pegasusMinimumHitTarget()
                    .accessibilityLabel("Supplier search query")
                    .onChange(of: vm.query) { _, newValue in
                        Task { await vm.search(newValue, existingSuppliers: existingSuppliers) }
                    }
            } footer: {
                Text("mobile_retailer.ui.search_the_supplier_network_and_add_vendors_to_your_procurement_")
            }

            if vm.isSearching {
                Section {
                    HStack {
                        Spacer()
                        ProgressView()
                        Spacer()
                    }
                    .accessibilityLabel("Searching suppliers")
                }
            } else if vm.hasMinQueryLength && vm.results.isEmpty {
                Section {
                    Text("mobile_retailer.ui.no_new_suppliers_match_that_search")
                        .foregroundStyle(AppTheme.textSecondary)
                }
            } else if !vm.results.isEmpty {
                Section("Results") {
                    ForEach(vm.results) { supplier in
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
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(supplier.name)\(supplier.displayCategory.map { ", \($0)" } ?? "")")

                            Spacer()

                            Button("mobile_retailer.ui.add") {
                                Haptics.medium()
                                Task {
                                    let didConnect = await vm.connect(supplier.id, onConnected: onConnected)
                                    if didConnect {
                                        dismiss()
                                    }
                                }
                            }
                            .buttonStyle(.borderedProminent)
                            .pegasusMinimumHitTarget()
                            .disabled(vm.connectingSupplierId == supplier.id)
                            .accessibilityLabel("Connect supplier \(supplier.name)")
                        }
                    }
                }
            }

            if let statusMessage = vm.statusMessage {
                Section {
                    Text(statusMessage)
                        .font(.footnote)
                        .foregroundStyle(AppTheme.textSecondary)
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("mobile_retailer.ui.connect_vendor")
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("common.action.close") {
                    Haptics.light()
                    dismiss()
                }
                .pegasusMinimumHitTarget()
                .accessibilityLabel("Close")
            }
        }
    }
}
