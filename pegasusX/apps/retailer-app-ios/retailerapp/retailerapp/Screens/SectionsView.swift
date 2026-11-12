import SwiftUI

struct SectionsView: View {
    @State private var vm = SectionsViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            if let saveError = vm.saveError {
                Section {
                    Text(saveError)
                        .font(.caption)
                        .foregroundStyle(AppTheme.destructive)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Error: \(saveError)")
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
            Section("New section") {
                TextField("retailer_desktop.pos.text.name", text: $vm.name)
                    .accessibilityLabel("Section name")
                TextField("retailer_desktop.sections.text.aisle_tag", text: $vm.aisle)
                    .accessibilityLabel("Aisle tag")
                Button(vm.busy ? "…" : "Create") {
                    Task { await vm.create() }
                }
                .pegasusMinimumHitTarget()
                .disabled(vm.busy)
                .accessibilityLabel("Create section")
            }
            Section("portal.nav.sections") {
                if vm.items.isEmpty {
                    Text("mobile_retailer.ui.none_yet")
                        .foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(vm.items) { row in
                        Button {
                            Haptics.selection()
                            vm.selectedId = row.id
                        } label: {
                            HStack {
                                Text(row.name)
                                Spacer()
                                if vm.selectedId == row.id {
                                    Image(systemName: "checkmark")
                                }
                            }
                            .pegasusMinimumHitTarget()
                        }
                        .accessibilityLabel("\(row.name)\(vm.selectedId == row.id ? ", selected" : "")")
                    }
                }
            }
            if vm.selectedId != nil {
                Section("Map SKUs") {
                    TextField("mobile_retailer.ui.sku_list_comma_separated", text: $vm.skuText)
                        .accessibilityLabel("Comma separated SKU list")
                    Button("mobile_retailer.ui.save_skus") {
                        Task { await vm.saveSkus() }
                    }
                    .pegasusMinimumHitTarget()
                    .disabled(vm.busy)
                    .accessibilityLabel("Save SKUs to section")
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("portal.nav.sections")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.refresh() }
        .refreshable { await vm.refresh() }
    }
}
