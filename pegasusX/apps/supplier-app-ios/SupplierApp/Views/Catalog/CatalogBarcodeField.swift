import SwiftUI

struct CatalogBarcodeField: View {
    @Binding var value: String
    var enabled: Bool = true

    @State private var showScanner = false
    @State private var scannerEnabled = true
    @State private var validationMessage: String?

    private var normalized: String? {
        EANBarcode.normalize(value)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            TextField("mobile_supplier.ui.ean_gtin_barcode", text: $value)
                .keyboardType(.numberPad)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .disabled(!enabled)
                .onChange(of: value) { _, _ in
                    validationMessage = nil
                }

            Button(showScanner ? "Hide camera" : "Scan barcode") {
                showScanner.toggle()
            }
            .font(.caption)
            .disabled(!enabled)

            if showScanner && enabled {
                EANBarcodeScannerView(
                    onBarcode: { scanned in
                        guard let code = EANBarcode.normalize(scanned) else {
                            validationMessage = "Invalid EAN/GTIN — check the label"
                            return
                        }
                        validationMessage = nil
                        value = code
                        scannerEnabled = false
                        Task {
                            try? await Task.sleep(for: .milliseconds(1500))
                            scannerEnabled = true
                        }
                    },
                    enabled: scannerEnabled
                )
            }

            if let validationMessage {
                Text(validationMessage)
                    .font(.caption)
                    .foregroundStyle(.red)
            } else if !value.isEmpty {
                Text(normalized != nil ? "Valid GTIN: \(normalized!)" : "Enter a valid EAN/GTIN checksum")
                    .font(.caption)
                    .foregroundStyle(normalized != nil ? Color.secondary : Color.red)
            }
        }
    }
}
