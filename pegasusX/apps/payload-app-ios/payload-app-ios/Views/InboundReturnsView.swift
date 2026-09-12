//
//  InboundReturnsView.swift
//  payload-app-ios
//
//  Mirrors payload-terminal inboundReturns.tsx: queue + history, EAN scan,
//  tap-to-select rows, restock / write-off confirm.
//

import SwiftUI

struct InboundReturnsView: View {
    var online: Bool = true
    @Environment(\.dismiss) private var dismiss
    @State private var tab: Tab = .queue
    @State private var rows: [InboundReturnRow] = []
    @State private var history: [InboundReturnRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var barcode = ""
    @State private var sessionId: String?
    @State private var scanning = false
    @State private var selected = Set<String>()
    @State private var statusMessage: String?
    @State private var queuedScans = 0
    @State private var scannerEnabled = true

    private enum Tab: String, CaseIterable {
        case queue = "Queue"
        case history = "History"
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                scanHeader
                Picker("View", selection: $tab) {
                    ForEach(Tab.allCases, id: \.self) { t in
                        Text(t.rawValue).tag(t)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, TermTheme.s16)
                .padding(.bottom, TermTheme.s8)

                Group {
                    if loading {
                        PayloadLoadingView(title: "Loading inbound returns…")
                    } else if let error {
                        PayloadErrorView(message: error) { Task { await load() } }
                    } else if tab == .queue {
                        queueList
                    } else {
                        historyList
                    }
                }
            }
            .navigationTitle("warehouse_portal.returns.text.inbound_returns")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.close") { dismiss() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button { Task { await load() } } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .task { await load() }
            .refreshable { await load() }
        }
    }

    private var scanHeader: some View {
        VStack(alignment: .leading, spacing: TermTheme.s8) {
            EANBarcodeScannerView(
                onBarcode: { code in
                    barcode = code
                    Task { await handleScan() }
                },
                enabled: scannerEnabled && !scanning
            )
            Text("mobile_payload.ui.ean_barcode_2")
                .font(.caption)
                .foregroundStyle(TermTheme.secondary)
            HStack(spacing: TermTheme.s8) {
                TextField("mobile_payload.ui.scan_or_type_ean", text: $barcode)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .padding(TermTheme.s12)
                    .background(TermTheme.card)
                    .clipShape(.rect(cornerRadius: 8))
                    .onSubmit { Task { await handleScan() } }
                Button(scanning ? "…" : "Scan") {
                    Task { await handleScan() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(scanning || barcode.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            HStack(spacing: TermTheme.s8) {
                Button("mobile_payload.ui.restock") { Task { await confirm(disposition: "RESTOCK") } }
                    .frame(maxWidth: .infinity)
                    .buttonStyle(.borderedProminent)
                    .tint(.green)
                Button("supplier_portal.returns.text.write_off") { Task { await confirm(disposition: "WRITE_OFF") } }
                    .frame(maxWidth: .infinity)
                    .buttonStyle(.borderedProminent)
                    .tint(.red)
            }
            if queuedScans > 0 {
                Text(L10n.format("mobile_payload.ui.queuedscans_scan_s_queued_offline_2", "\(queuedScans)"))
                    .font(.caption2)
                    .foregroundStyle(.orange)
            }
            if let statusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
            .padding(TermTheme.s16)
            .background(TermTheme.bg)
    }

    private var queueList: some View {
        Group {
            if rows.isEmpty {
                ContentUnavailableView("No trucks at gate", systemImage: "arrow.uturn.backward.circle")
            } else {
                InboundReturnsList(rows: rows, selectable: true, selected: $selected)
            }
        }
    }

    private var historyList: some View {
        Group {
            if history.isEmpty {
                ContentUnavailableView("No history", systemImage: "clock.arrow.circlepath")
            } else {
                InboundReturnsList(rows: history, selectable: false, selected: $selected)
            }
        }
    }



    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            async let inbound = APIClient.shared.inboundReturns()
            async let hist = APIClient.shared.returnsHistory(limit: 50)
            let (inboundResp, histResp) = try await (inbound, hist)
            rows = inboundResp.data
            history = histResp.data
            queuedScans = OfflineQueue.shared.read().filter { $0.endpoint.contains("returns/inbound/scan") }.count
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func ensureSession() async throws -> String {
        if let sessionId { return sessionId }
        let resp = try await APIClient.shared.createInboundSession()
        sessionId = resp.sessionId
        return resp.sessionId
    }

    private func handleScan() async {
        scanning = true
        statusMessage = nil
        defer { scanning = false }
        let trimmed = barcode.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else { return }

        if !online {
            enqueueOfflineScan(barcode: trimmed)
            barcode = ""
            statusMessage = "Scan queued (offline)"
            queuedScans = OfflineQueue.shared.read().count
            return
        }

        do {
            let sid = try await ensureSession()
            let resp = try await APIClient.shared.scanInboundBarcode(barcode: trimmed, sessionId: sid)
            barcode = ""
            scannerEnabled = false
            statusMessage = resp.message ?? (resp.variance ? "Variance flagged" : "Scanned \(resp.returnId ?? "")")
            await load()
            try? await Task.sleep(for: .milliseconds(1500))
            scannerEnabled = true
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    private func enqueueOfflineScan(barcode: String) {
        struct ScanQueueBody: Encodable {
            let barcode: String
            let qty: Int
            let sessionId: String
            enum CodingKeys: String, CodingKey {
                case barcode, qty
                case sessionId = "session_id"
            }
        }
        let bodyData = (try? JSONEncoder().encode(ScanQueueBody(barcode: barcode, qty: 1, sessionId: sessionId ?? ""))) ?? Data()
        let action = QueuedActionModel(
            id: PayloadIdempotency.inboundScan(barcode: barcode, sessionId: sessionId ?? "offline"),
            endpoint: "v1/returns/inbound/scan",
            method: "POST",
            body: String(data: bodyData, encoding: .utf8) ?? "{}",
            createdAt: Date().timeIntervalSince1970
        )
        OfflineQueue.shared.enqueue(action)
    }

    private func confirm(disposition: String) async {
        let targets = rows.filter { selected.contains($0.returnId) || selected.isEmpty }
        guard !targets.isEmpty else {
            statusMessage = "Select lines or scan first"
            return
        }
        do {
            let sid = try await ensureSession()
            _ = try await APIClient.shared.confirmInboundReturns(
                returnIds: targets.map(\.returnId),
                disposition: disposition,
                sessionId: sid,
                quantities: Dictionary(uniqueKeysWithValues: targets.map { ($0.returnId, max($0.receivedQty, $0.expectedQty)) })
            )
            selected.removeAll()
            statusMessage = "\(disposition) confirmed"
            await load()
        } catch {
            statusMessage = error.localizedDescription
        }
    }
}
