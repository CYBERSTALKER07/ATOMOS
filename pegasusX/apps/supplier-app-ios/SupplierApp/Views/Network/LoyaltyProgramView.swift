import SwiftUI

struct LoyaltyProgramView: View {
    @State private var vm = LoyaltyProgramViewModel()

    var body: some View {
        @Bindable var vm = vm
        Form {
            if vm.loading && vm.program == nil {
                ProgressView("mobile_supplier.loading.loyalty")
            } else if let error = vm.error, vm.program == nil {
                Text(error).foregroundStyle(SupplierTheme.destructive)
            } else {
                Text("mobile_supplier.chrome.earn_on_paid_orders")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                Text(L10n.format("mobile_supplier.chrome.source_line", vm.program?.source ?? L10n.format("mobile_supplier.chrome.unconfigured")))
                    .font(.caption)
                TextField("mobile_supplier.chrome.earn_bps", text: $vm.earnBps)
                    .keyboardType(.numberPad)
                TextField("mobile_supplier.chrome.reason_required", text: $vm.reason)
                Button(LocalizedStringKey(vm.busy ? "mobile_supplier.chrome.saving" : "mobile_supplier.chrome.save_program")) {
                    PegasusHaptics.light()
                    Task { await vm.save() }
                }
                .frame(minHeight: 44)
                .disabled(vm.busy)
                if let status = vm.status {
                    Text(status).font(.footnote)
                }
                ForEach(vm.program?.tiers ?? [], id: \.name) { tier in
                    Text(L10n.format("mobile_supplier.chrome.tier_from_points", tier.name, "\(tier.minPoints)"))
                        .font(.caption)
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("portal.nav.loyalty")
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
    }
}
