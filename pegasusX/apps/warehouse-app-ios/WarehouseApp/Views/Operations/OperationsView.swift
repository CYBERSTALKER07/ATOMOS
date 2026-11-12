import SwiftUI

struct OperationsView: View {
    @State private var vm = OperationsViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.templates.isEmpty && vm.error == nil {
                WarehouseLoadingView(
                    title: "Loading operations",
                    message: "Fetching broadcast templates and depot tools."
                )
            } else if let error = vm.error, vm.templates.isEmpty {
                WarehouseErrorView(message: error) {
                    Task { await vm.loadTemplates() }
                }
            } else {
                operationsForm
            }
        }
        .background(LabTheme.background)
        .navigationTitle("warehouse_portal.operations.text.depot_operations")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    Task { await vm.loadTemplates(silent: true) }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh operations")
            }
        }
        .task { await vm.loadTemplates() }
        .refreshable { await vm.loadTemplates(silent: true) }
    }

    private var operationsForm: some View {
        @Bindable var vm = vm
        return Form {
            if let statusMessage = vm.statusMessage {
                Section {
                    Text(statusMessage)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }

            OperationsBroadcastForm(
                broadcastRoles: vm.broadcastRoles,
                templates: vm.templates,
                templateDate: $vm.templateDate,
                customReason: $vm.customReason,
                title: $vm.title,
                broadcastRole: $vm.broadcastRole,
                bodyText: $vm.bodyText,
                saveAsTemplate: $vm.saveAsTemplate,
                broadcasting: vm.broadcasting,
                savingTemplate: vm.savingTemplate,
                onApplyTemplate: { template in vm.applyTemplate(template) },
                onDeleteTemplate: { template in await vm.deleteTemplate(template) },
                onSendBroadcast: { await vm.sendBroadcast() }
            )

            OperationsPricingPreview(
                productId: $vm.productId,
                retailerId: $vm.retailerId,
                proposedPrice: $vm.proposedPrice,
                previewLoading: vm.previewLoading,
                preview: vm.preview,
                onSchedulePreview: { vm.schedulePreview() }
            )
        }
        .scrollDismissesKeyboard(.interactively)
    }
}
