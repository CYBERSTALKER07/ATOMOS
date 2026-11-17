import PhotosUI
import SwiftUI

struct CatalogCreateSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var vm = CatalogCreateViewModel()

    var onCreated: () -> Void

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            Form {
                Section("mobile_supplier.chrome.product") {
                    TextField("retailer_desktop.pos.text.name", text: $vm.name)
                    Picker("mobile_supplier.chrome.category", selection: $vm.categoryId) {
                        ForEach(vm.categories) { category in
                            Text(category.name).tag(category.categoryId)
                        }
                    }
                    TextField("mobile_supplier.ui.price_minor_units", text: $vm.priceMinor)
                        .keyboardType(.numberPad)
                    TextField("supplier_portal.catalog.components.catalog_table.text.unit_vu", text: $vm.unitVu)
                        .keyboardType(.decimalPad)
                    CatalogBarcodeField(value: $vm.barcode, enabled: !vm.creating)
                    let hasImage = vm.imageData != nil
                    PhotosPicker(selection: $vm.photoItem, matching: .images) {
                        Label(hasImage ? "mobile_supplier.chrome.image_selected" : "mobile_supplier.chrome.add_image", systemImage: "photo")
                            .frame(minHeight: 44)
                    }
                    .onChange(of: vm.photoItem) { _, item in
                        guard item != nil else { return }
                        Task { await vm.handlePhotoSelection(item) }
                    }
                }
                if let error = vm.error {
                    Section {
                        Text(error)
                            .foregroundStyle(SupplierTheme.destructive)
                            .font(.caption)
                    }
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_supplier.ui.add_product")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                        .disabled(vm.creating)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(LocalizedStringKey(vm.creating ? "mobile_supplier.chrome.creating" : "mobile_supplier.chrome.create")) {
                        PegasusHaptics.light()
                        Task {
                            if await vm.create() {
                                onCreated()
                                dismiss()
                            }
                        }
                    }
                    .disabled(vm.creating || vm.categories.isEmpty)
                }
            }
            .task { await vm.loadForm() }
        }
    }
}
