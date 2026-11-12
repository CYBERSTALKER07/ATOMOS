import SwiftUI

struct ProductScannerSheet: View {
    @Bindable var viewModel: HomeViewModel
    @Binding var isPresented: Bool

    var body: some View {
        NavigationStack {
            VStack(spacing: 16) {
                Text("mobile_payload.ui.scan_product_ean")
                    .font(.system(.subheadline, design: .monospaced, weight: .black))
                EANBarcodeScannerView(onBarcode: { code in
                    PegasusHaptics.light()
                    Task {
                        await viewModel.onBarcodeScanned(code)
                        isPresented = false
                    }
                })
                Spacer()
            }
            .padding()
            .navigationTitle("mobile_payload.ui.product_scan")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.close") {
                        PegasusHaptics.light()
                        isPresented = false
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            }
        }
    }
}
