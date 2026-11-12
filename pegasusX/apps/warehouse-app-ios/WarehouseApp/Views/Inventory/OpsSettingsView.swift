import SwiftUI

struct OpsSettingsView: View {
    @State private var vm = OpsSettingsViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else {
                Form {
                    OpsSettingsForm(
                        preorderMinLeadDays: $vm.preorderMinLeadDays,
                        preorderMaxLeadDays: $vm.preorderMaxLeadDays,
                        policy: $vm.policy,
                        showStockCounts: $vm.showStockCounts,
                        clearOrderLineMin: $vm.clearOrderLineMin,
                        orderLineMin: $vm.orderLineMin,
                        clearOrderLineMax: $vm.clearOrderLineMax,
                        orderLineMax: $vm.orderLineMax,
                        expressEnabled: $vm.expressEnabled,
                        expressStockFloor: $vm.expressStockFloor,
                        clearFeeRules: $vm.clearFeeRules,
                        feeBaseMinor: $vm.feeBaseMinor,
                        feeCurrency: $vm.feeCurrency,
                        feeTiers: $vm.feeTiers,
                        enforceOrderAcceptance: $vm.enforceOrderAcceptance,
                        scheduleIs24h: $vm.scheduleIs24h,
                        scheduleTimezone: $vm.scheduleTimezone,
                        weekdayOpen: $vm.weekdayOpen,
                        weekdayClose: $vm.weekdayClose,
                        scheduleJSON: $vm.scheduleJSON,
                        scheduleError: vm.scheduleError
                    )

                    if let saveMessage = vm.saveMessage {
                        Section {
                            Text(saveMessage)
                                .foregroundStyle(saveMessage.contains("saved") ? .green : .red)
                        }
                    }

                    Section {
                        Button(vm.saving ? "Saving…" : "Save settings") {
                            PegasusHaptics.medium()
                            Task { await vm.save() }
                        }
                        .disabled(vm.saving)
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .accessibilityLabel("Save warehouse operations settings")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("mobile_warehouse.ui.ops_settings")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh ops settings")
            }
        }
        .task { await vm.load() }
    }
}
