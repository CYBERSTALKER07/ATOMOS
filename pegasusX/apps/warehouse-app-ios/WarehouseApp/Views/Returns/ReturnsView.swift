import SwiftUI

struct ReturnsView: View {
    private enum Tab: String, CaseIterable {
        case queue = "Gate queue"
        case history = "History"
        case reverse = "Credit-note"
    }

    @State private var tab: Tab = .queue
    @State private var returns: [InboundReturnRow] = []
    @State private var history: [InboundReturnRow] = []
    @State private var reverseTasks: [ReverseLogisticsTask] = []
    @State private var loading = true
    @State private var error: String?
    @State private var barcode = ""
    @State private var sessionId: String?
    @State private var scanning = false
    @State private var selected = Set<String>()
    @State private var statusMessage: String?
    @State private var scannerEnabled = true
    @State private var receivingTaskId: String?

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                if tab == .queue {
                    scanSection
                }
                Picker("View", selection: $tab) {
                    ForEach(Tab.allCases, id: \.self) { t in
                        Text(t.rawValue).tag(t)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, LabTheme.spacingMD)
                .padding(.vertical, LabTheme.spacingSM)

                if let statusMessage, tab != .queue {
                    Text(statusMessage)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, LabTheme.spacingMD)
                }

                Group {
                    if loading {
                        ProgressView()
                            .frame(maxWidth: .infinity, maxHeight: .infinity)
                    } else if let error {
                        ContentUnavailableView {
                            Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                        } description: {
                            Text(error)
                        } actions: {
                            Button("common.action.retry") { Task { await load() } }
                        }
                    } else {
                        switch tab {
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
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { Task { await load() } }
                }
                if tab == .queue, !selected.isEmpty {
                    ToolbarItem(placement: .bottomBar) {
                        HStack {
                            Button("mobile_warehouse.ui.restock") { Task { await confirm(disposition: "RESTOCK") } }
                            Spacer()
                            Button("supplier_portal.returns.text.write_off", role: .destructive) { Task { await confirm(disposition: "WRITE_OFF") } }
                        }
                    }
                }
            }
            .task { await load() }
            .refreshable { await load() }
        }
    }

    private var scanSection: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            EANBarcodeScannerView(
                onBarcode: { code in
                    barcode = code
                    Task { await handleScan() }
                },
                enabled: scannerEnabled && !scanning
            )
            HStack {
                TextField("mobile_warehouse.ui.ean_return_id", text: $barcode)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button(scanning ? "…" : "Scan") {
                    Task { await handleScan() }
                }
                .disabled(scanning || barcode.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            if let statusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(LabTheme.spacingMD)
    }

    private var queueList: some View {
        ReturnsList(items: returns, isQueueTab: true, selected: $selected)
    }

    private var historyList: some View {
        ReturnsList(items: history, isQueueTab: false, selected: $selected)
    }

    private var reverseList: some View {
        Group {
            if reverseTasks.isEmpty {
                ContentUnavailableView(
                    "No open credit-note reverse tasks",
                    systemImage: "arrow.uturn.backward.circle"
                )
            } else {
                List {
                    ForEach(reverseTasks) { task in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(task.taskId)
                                    .font(.headline.monospaced())
                                Text(L10n.format("mobile_warehouse.ui.order_orderid_status", "\(task.orderId)", "\(task.status)"))
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            Button(receivingTaskId == task.taskId ? "…" : "Receive") {
                                Task { await receive(task) }
                            }
                            .disabled(receivingTaskId != nil)
                            .buttonStyle(.borderedProminent)
                        }
                    }
                }
                .listStyle(.plain)
            }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let wh = TokenStore.shared.warehouseId
            async let inbound = WarehouseService.inboundReturns(physicalStatus: "OPEN")
            async let hist = WarehouseService.returnsHistory(limit: 50)
            async let reverse = WarehouseService.reverseLogistics(status: "OPEN", warehouseId: wh)
            let (inboundResp, histResp, reverseResp) = try await (inbound, hist, reverse)
            returns = inboundResp.data
            history = histResp.data
            reverseTasks = reverseResp.tasks
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func ensureSession() async throws -> String {
        if let sessionId { return sessionId }
        let sid = try await WarehouseService.createInboundSession()
        sessionId = sid
        return sid
    }

    private func handleScan() async {
        scanning = true
        statusMessage = nil
        defer { scanning = false }
        do {
            let sid = try await ensureSession()
            let trimmed = barcode.trimmingCharacters(in: .whitespaces)
            let key = WarehouseIdempotency.inboundScan(barcode: trimmed, sessionId: sid)
            _ = try await WarehouseService.scanInboundBarcode(
                barcode: trimmed,
                qty: 1,
                sessionId: sid,
                idempotencyKey: key
            )
            barcode = ""
            statusMessage = "Scan recorded"
            await load()
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    private func confirm(disposition: String) async {
        do {
            let sid = try await ensureSession()
            let key = WarehouseIdempotency.inboundConfirm(returnIds: Array(selected), disposition: disposition)
            _ = try await WarehouseService.confirmInboundReturns(
                returnIds: Array(selected),
                disposition: disposition,
                sessionId: sid,
                idempotencyKey: key
            )
            selected.removeAll()
            statusMessage = "\(disposition) confirmed"
            await load()
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    private func receive(_ task: ReverseLogisticsTask) async {
        guard !task.taskId.isEmpty else { return }
        receivingTaskId = task.taskId
        defer { receivingTaskId = nil }
        do {
            let wh = TokenStore.shared.warehouseId?.trimmingCharacters(in: .whitespacesAndNewlines)
            let warehouseId = (wh?.isEmpty == false ? wh! : nil)
                ?? (task.warehouseId.isEmpty ? "warehouse" : task.warehouseId)
            let qty = Self.parseReceivedQty(task.expectedQtyJson)
            _ = try await WarehouseService.receiveReverseLogistics(
                taskId: task.taskId,
                warehouseId: warehouseId,
                receivedQty: qty
            )
            statusMessage = "Received \(task.taskId)"
            await load()
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    private static func parseReceivedQty(_ raw: String?) -> [String: Int] {
        guard let raw, let data = raw.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
        else { return [:] }
        var out: [String: Int] = [:]
        for (k, v) in obj {
            if let n = v as? Int { out[k] = n }
            else if let n = v as? NSNumber { out[k] = n.intValue }
        }
        return out
    }
}
