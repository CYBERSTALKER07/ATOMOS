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
            .navigationTitle("Inbound Returns")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Close") { dismiss() }
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
            Text("EAN / barcode")
                .font(.caption)
                .foregroundStyle(TermTheme.secondary)
            HStack(spacing: TermTheme.s8) {
                TextField("Scan or type EAN", text: $barcode)
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
                Button("Restock") { Task { await confirm(disposition: "RESTOCK") } }
                    .frame(maxWidth: .infinity)
                    .buttonStyle(.borderedProminent)
                    .tint(.green)
                Button("Write off") { Task { await confirm(disposition: "WRITE_OFF") } }
                    .frame(maxWidth: .infinity)
                    .buttonStyle(.borderedProminent)
                    .tint(.red)
            }
            if queuedScans > 0 {
                Text("\(queuedScans) scan(s) queued offline")
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
                ScrollView {
                    LazyVStack(spacing: TermTheme.s12) {
                        ForEach(rows) { row in
                            inboundCard(row)
                        }
                    }
                    .padding(TermTheme.s16)
                }
            }
        }
    }

    private var historyList: some View {
        Group {
            if history.isEmpty {
                ContentUnavailableView("No history", systemImage: "clock.arrow.circlepath")
            } else {
                ScrollView {
                    LazyVStack(spacing: TermTheme.s12) {
                        ForEach(history) { row in
                            inboundCard(row, selectable: false)
                        }
                    }
                    .padding(TermTheme.s16)
                }
            }
        }
    }

    private func inboundCard(_ row: InboundReturnRow, selectable: Bool = true) -> some View {
        let isSelected = selected.contains(row.returnId)
        return Button {
            guard selectable else { return }
            if isSelected { selected.remove(row.returnId) }
            else { selected.insert(row.returnId) }
        } label: {
            VStack(alignment: .leading, spacing: 4) {
                Text(row.productName)
                    .font(.headline)
                    .foregroundStyle(TermTheme.accent)
                Text("\(row.driverName.isEmpty ? "Driver" : row.driverName) · \(row.reason) · \(row.receivedQty)/\(row.expectedQty)")
                    .font(.subheadline)
                    .foregroundStyle(TermTheme.secondary)
                Text("\(row.returnId.prefix(8)) · suggest \(row.suggestedDisposition)")
                    .font(.caption.monospaced())
                    .foregroundStyle(TermTheme.tertiary)
                if let barcode = row.barcode, !barcode.isEmpty {
                    Text("EAN \(barcode)")
                        .font(.caption2.monospaced())
                        .foregroundStyle(TermTheme.secondary)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(TermTheme.s16)
            .background(isSelected ? TermTheme.accent.opacity(0.12) : TermTheme.card)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isSelected ? TermTheme.accent : TermTheme.separator, lineWidth: 1)
            )
            .clipShape(.rect(cornerRadius: 12))
        }
        .buttonStyle(.plain)
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
