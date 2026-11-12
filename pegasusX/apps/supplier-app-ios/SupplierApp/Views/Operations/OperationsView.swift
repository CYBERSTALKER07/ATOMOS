import SwiftUI

struct OperationsView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var vm = OperationsViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.empathy == nil && vm.error == nil {
                SupplierLoadingView(
                    title: "mobile_supplier.loading.operations",
                    message: "mobile_supplier.loading.operations_message"
                )
            } else if let error = vm.error, vm.empathy == nil {
                SupplierErrorView(message: error) {
                    Task { await vm.loadEmpathy() }
                }
            } else {
                operationsForm
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.operations")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.loadEmpathy() }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(SupplierTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(Text("portal.page.orders.action.refresh"))
            }
        }
        .task { await vm.loadEmpathy() }
        .refreshable { await vm.loadEmpathy(silent: true) }
        .confirmationDialog(
            "mobile_supplier.chrome.issue_payment_bypass",
            isPresented: $vm.showBypassConfirm,
            titleVisibility: .visible
        ) {
            Button("mobile_supplier.ui.issue_token", role: .destructive) {
                Task { await vm.issueBypass() }
            }
            Button("common.action.cancel", role: .cancel) {}
        } message: {
            Text("mobile_supplier.ui.order_must_be_awaiting_payment_driver_receives_a_one_time_bypass")
        }
    }

    private var operationsForm: some View {
        @Bindable var vm = vm
        return Form {
            if let empathy = vm.empathy {
                Section {
                    SupplierSectionHeader(
                        title: "mobile_supplier.chrome.empathy_adoption",
                        subtitle: "mobile_supplier.chrome.prediction_lifecycle"
                    )
                }
                Section {
                    LabeledContent("mobile_supplier.chrome.predictions", value: "\(empathy.totalPredictions)")
                    LabeledContent("mobile_supplier.chrome.waiting", value: "\(empathy.predictionsWaiting)")
                    LabeledContent("mobile_supplier.chrome.fired", value: "\(empathy.predictionsFired)")
                    LabeledContent("mobile_supplier.chrome.dormant", value: "\(empathy.predictionsDormant)")
                    LabeledContent("mobile_supplier.chrome.rejected", value: "\(empathy.predictionsRejected)")
                }
            }

            OperatorBroadcastView(
                title: $vm.title,
                bodyText: $vm.bodyText,
                broadcastRole: $vm.broadcastRole,
                templateDate: $vm.templateDate,
                broadcasting: vm.broadcasting,
                sendBroadcast: {
                    Task { await vm.sendBroadcast(supplierId: tokenStore.supplierId) }
                }
            )

            ReplenishmentActionView(
                replenishing: vm.replenishing,
                runReplenishment: { vm.runReplenishment() }
            )

            PaymentBypassView(
                orderId: $vm.orderId,
                bypassReason: $vm.bypassReason,
                bypassToken: vm.bypassToken,
                bypassing: vm.bypassing,
                showBypassConfirm: $vm.showBypassConfirm
            )

            if let statusMessage = vm.statusMessage {
                Section {
                    Text(statusMessage)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }
}
