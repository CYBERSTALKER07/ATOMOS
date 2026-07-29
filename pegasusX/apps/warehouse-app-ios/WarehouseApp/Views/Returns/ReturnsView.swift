import SwiftUI

struct ReturnsView: View {
    private enum Tab: String, CaseIterable {
        case queue = "Gate queue"
        case history = "History"
    }

    @State private var tab: Tab = .queue
    @State private var returns: [InboundReturnRow] = []
    @State private var history: [InboundReturnRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var barcode = ""
    @State private var sessionId: String?
    @State private var scanning = false
    @State private var selected = Set<String>()
    @State private var statusMessage: String?
    @State private var scannerEnabled = true

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

                Group {
                    if loading {
                        ProgressView()
                            .frame(maxWidth: .infinity, maxHeight: .infinity)
                    } else if let error {
                        ContentUnavailableView {
                            Label("Error", systemImage: "exclamationmark.triangle")
                        } description: {
                            Text(error)
                        } actions: {
                            Button("Retry") { Task { await load() } }
                        }
                    } else if tab == .queue {
                        queueList
                    } else {
                        historyList
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Inbound Returns")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { Task { await load() } }
                }
                if tab == .queue, !selected.isEmpty {
                    ToolbarItem(placement: .bottomBar) {
                        HStack {
                            Button("Restock") { Task { await confirm(disposition: "RESTOCK") } }
                            Spacer()
                            Button("Write off", role: .destructive) { Task { await confirm(disposition: "WRITE_OFF") } }
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
                TextField("EAN / return ID", text: $barcode)
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

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            async let inbound = WarehouseService.inboundReturns()
            async let hist = WarehouseService.returnsHistory(limit: 50)
            let (inboundResp, histResp) = try await (inbound, hist)
            returns = inboundResp.data
            history = histResp.data
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
            scannerEnabled = false
            statusMessage = "Scan recorded"
            await load()
            try? await Task.sleep(for: .milliseconds(1500))
            scannerEnabled = true
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
}
