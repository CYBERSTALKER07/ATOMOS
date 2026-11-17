import Observation
import SwiftUI

@Observable
@MainActor
final class CatalogBarcodeFieldViewModel {
    var showScanner = false
    var scannerEnabled = true
    var validationMessage: String?

    func handleScanned(_ scanned: String) async -> String? {
        guard let code = EANBarcode.normalize(scanned) else {
            validationMessage = L10n.format("mobile_supplier.chrome.invalid_gtin_label")
            PegasusHaptics.warning()
            return nil
        }
        validationMessage = nil
        scannerEnabled = false
        PegasusHaptics.success()
        try? await Task.sleep(for: .milliseconds(1500))
        scannerEnabled = true
        return code
    }
}

struct CatalogBarcodeField: View {
    @Binding var value: String
    var enabled: Bool = true

    @State private var viewModel = CatalogBarcodeFieldViewModel()

    private var normalized: String? {
        EANBarcode.normalize(value)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            TextField("mobile_supplier.ui.ean_gtin_barcode", text: $value)
                .font(.body.monospaced())
                .keyboardType(.numberPad)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .disabled(!enabled)
                .onChange(of: value) { _, _ in
                    viewModel.validationMessage = nil
                }

            Button(LocalizedStringKey(viewModel.showScanner ? "mobile_supplier.chrome.hide_camera" : "mobile_supplier.chrome.scan_barcode")) {
                PegasusHaptics.light()
                viewModel.showScanner.toggle()
            }
            .font(.caption)
            .frame(minHeight: 44)
            .contentShape(Rectangle())
            .disabled(!enabled)

            if viewModel.showScanner && enabled {
                EANBarcodeScannerView(
                    onBarcode: { scanned in
                        Task {
                            if let code = await viewModel.handleScanned(scanned) {
                                value = code
                            }
                        }
                    },
                    enabled: viewModel.scannerEnabled
                )
            }

            if let validationMessage = viewModel.validationMessage {
                Text(validationMessage)
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.destructive)
            } else if !value.isEmpty {
                if let normalized {
                    Text(L10n.format("mobile_supplier.chrome.valid_gtin", normalized))
                        .font(.caption)
                        .foregroundStyle(Color.secondary)
                } else {
                    Text(L10n.format("mobile_supplier.chrome.enter_valid_gtin"))
                        .font(.caption)
                        .foregroundStyle(SupplierTheme.destructive)
                }
            }
        }
    }
}
