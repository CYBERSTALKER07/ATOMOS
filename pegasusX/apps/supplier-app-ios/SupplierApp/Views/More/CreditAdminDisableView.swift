import SwiftUI

struct CreditAdminDisableView: View {
    @State private var vm = CreditAdminDisableViewModel()

    var body: some View {
        @Bindable var vm = vm
        Form {
            Picker("mobile_supplier.chrome.mode", selection: $vm.mode) {
                Text("mobile_supplier.chrome.relationship").tag("relationship")
                Text("mobile_supplier.chrome.program").tag("mobile_supplier.chrome.source_program")
            }
            .onChange(of: vm.mode) { _, _ in
                PegasusHaptics.selection()
            }
            TextField("mobile_supplier.chrome.supplier_id", text: $vm.supplierId)
            if !vm.isProgramMode {
                TextField("mobile_supplier.chrome.retailer_id", text: $vm.retailerId)
            }
            TextField("mobile_supplier.chrome.ticket_id", text: $vm.ticketId)
            TextField("mobile_supplier.chrome.reason", text: $vm.reason, axis: .vertical)
            Button("mobile_supplier.chrome.disable_permanently") {
                PegasusHaptics.light()
                Task { await vm.submit() }
            }
            .frame(minHeight: 44)
            .disabled(!vm.canSubmit)
            if let message = vm.message {
                Text(message)
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("mobile_supplier.chrome.credit_admin_disable")
    }
}
