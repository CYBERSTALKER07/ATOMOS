import Observation
import PhotosUI
import SwiftUI

@Observable
@MainActor
final class CatalogImagePickerViewModel {
    var photoItem: PhotosPickerItem?
    var error: String?

    func handleSelection(item: PhotosPickerItem?, onUpload: (Data) async -> Void) async {
        guard let item else { return }
        do {
            let loaded = try await item.loadTransferable(type: Data.self)
            error = nil
            await onUpload(try PhotoLoad.imageData(loaded))
        } catch {
            self.error = error.localizedDescription
        }
        photoItem = nil
    }
}

struct CatalogProductImageButton: View {
    let product: CatalogProduct
    let isSaving: Bool
    let onUpload: (Data) async -> Void

    @State private var vm = CatalogImagePickerViewModel()

    var body: some View {
        @Bindable var vm = vm
        VStack(alignment: .leading, spacing: 4) {
            PhotosPicker(selection: $vm.photoItem, matching: .images) {
                Text(LocalizedStringKey(label))
                    .font(.caption)
                    .frame(minHeight: 44)
            }
            .disabled(isSaving)
            if let error = vm.error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.destructive)
            }
        }
        .onChange(of: vm.photoItem) { _, item in
            guard item != nil else { return }
            Task { await vm.handleSelection(item: item, onUpload: onUpload) }
        }
    }

    private var label: String {
        if isSaving { return "mobile_supplier.chrome.uploading" }
        if let imageUrl = product.imageUrl, !imageUrl.isEmpty { return "mobile_supplier.chrome.change_image" }
        return "mobile_supplier.chrome.add_image"
    }
}
