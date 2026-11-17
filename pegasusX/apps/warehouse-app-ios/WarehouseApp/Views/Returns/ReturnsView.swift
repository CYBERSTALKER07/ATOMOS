import SwiftUI

struct ReturnsView: View {
    @State private var vm = ReturnsViewModel()

    var body: some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            if vm.tab == .queue {
                scanSection
            }
            Picker("View", selection: $vm.tab) {
                ForEach(ReturnsTab.allCases) { t in
                    Text(t.rawValue).tag(t)
                }
            }
            .pickerStyle(.segmented)
            .padding(.horizontal, LabTheme.spacingMD)
            .padding(.vertical, LabTheme.spacingSM)
            .onChange(of: vm.tab) { _, _ in
                PegasusHaptics.selection()
            }

            if let statusMessage = vm.statusMessage, vm.tab != .queue {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, LabTheme.spacingMD)
            }

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
                    switch vm.tab {
                    case .queue:
                        queueList
                    case .history:
                        historyList
                    case .reverse:
                        reverseList
                    }
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("warehouse_portal.returns.text.inbound_returns")
        .scrollDismissesKeyboard(.interactively)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh returns")
            }
            if vm.tab == .queue, !vm.selected.isEmpty {
                ToolbarItem(placement: .bottomBar) {
                    HStack {
                        Button("mobile_warehouse.ui.restock") {
                            PegasusHaptics.medium()
                            Task { await vm.confirm(disposition: "RESTOCK") }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        Spacer()
                        Button("supplier_portal.returns.text.write_off", role: .destructive) {
                            PegasusHaptics.warning()
                            Task { await vm.confirm(disposition: "WRITE_OFF") }
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                    }
                }
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }

    private var scanSection: some View {
        @Bindable var vm = vm
        return VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            EANBarcodeScannerView(
                onBarcode: { code in
                    Task { await vm.handleScannedCode(code) }
                },
                enabled: vm.scannerEnabled && !vm.scanning
            )
            HStack {
                TextField("mobile_warehouse.ui.ean_return_id", text: $vm.barcode)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button(vm.scanning ? "…" : "Scan") {
                    PegasusHaptics.light()
                    Task { await vm.handleScan() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .disabled(vm.scanning || vm.trimmedBarcode.isEmpty)
                .accessibilityLabel("Scan return barcode")
            }
            if let statusMessage = vm.statusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(LabTheme.spacingMD)
    }

    private var queueList: some View {
        @Bindable var vm = vm
        return ReturnsList(items: vm.returns, isQueueTab: true, selected: $vm.selected)
    }

    private var historyList: some View {
        @Bindable var vm = vm
        return ReturnsList(items: vm.history, isQueueTab: false, selected: $vm.selected)
    }

    private var reverseList: some View {
        Group {
            if vm.reverseTasks.isEmpty {
                ContentUnavailableView(
                    "No open credit-note reverse tasks",
                    systemImage: "arrow.uturn.backward.circle"
                )
            } else {
                List {
                    ForEach(vm.reverseTasks) { task in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(task.taskId)
                                    .font(.headline.monospaced())
                                Text(L10n.format("mobile_warehouse.ui.order_orderid_status", "\(task.orderId)", "\(task.status)"))
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            Button(vm.receivingTaskId == task.taskId ? "…" : "Receive") {
                                PegasusHaptics.medium()
                                Task { await vm.receive(task) }
                            }
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                            .disabled(vm.receivingTaskId != nil)
                            .buttonStyle(.borderedProminent)
                            .accessibilityLabel("Receive reverse task \(task.taskId)")
                        }
                    }
                }
                .listStyle(.plain)
            }
        }
    }
}
