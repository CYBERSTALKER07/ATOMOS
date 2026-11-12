import SwiftUI

struct LocalSkusView: View {
    @State private var vm = LocalSkusViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            if let loadError = vm.loadError {
                Section {
                    Text(loadError)
                        .font(.caption)
                        .foregroundStyle(AppTheme.destructive)
                    Button("common.action.retry") {
                        Haptics.light()
                        Task { await vm.load() }
                    }
                    .pegasusMinimumHitTarget()
                    .accessibilityLabel("Retry loading local SKUs")
                }
            } else if let banner = vm.banner {
                Section {
                    Text(banner)
                        .font(.caption)
                        .foregroundStyle(AppTheme.accent)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(banner)
                }
            }

            Section("Add local SKU") {
                TextField("retailer_desktop.pos.text.name", text: $vm.name)
                    .accessibilityLabel("SKU name")
                TextField("retailer_desktop.stock.local_skus.text.barcode", text: $vm.barcode)
                    .accessibilityLabel("Barcode")
                TextField("supplier_portal.catalog.components.catalog_table.text.price_minor", text: $vm.priceInput)
                    .keyboardType(.numberPad)
                    .accessibilityLabel("Price in minor units")
                Button(vm.busy ? "…" : "Create") {
                    Task { await vm.create() }
                }
                .pegasusMinimumHitTarget()
                .disabled(!vm.canCreate)
                .accessibilityLabel("Create local SKU")
            }

            if vm.summaryReady && vm.loadError == nil {
                Section("Catalog") {
                    if vm.rows.isEmpty {
                        Text("mobile_retailer.ui.no_local_skus")
                            .foregroundStyle(AppTheme.textTertiary)
                    } else {
                        ForEach(vm.rows) { row in
                            HStack(alignment: .center, spacing: AppTheme.spacingMD) {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(row.name)
                                        .font(.headline)
                                    Text("\(row.id) · \(row.priceMinor) \(row.currency) · \(row.active ? "active" : "inactive")")
                                        .font(.caption)
                                        .foregroundStyle(AppTheme.textSecondary)
                                }
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("\(row.name), \(row.priceMinor) \(row.currency), \(row.active ? "active" : "inactive")")

                                Spacer()

                                Button(row.active ? "Disable" : "Enable") {
                                    Task { await vm.toggle(row) }
                                }
                                .buttonStyle(.bordered)
                                .pegasusMinimumHitTarget()
                                .disabled(vm.busy)
                                .accessibilityLabel("\(row.active ? "Disable" : "Enable") \(row.name)")
                            }
                            .padding(.vertical, 2)
                        }
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("portal.nav.local_skus")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
