//
//  InboundReturnsViewModel.swift
//  payload-app-ios
//
//  Phase 3 MVVM decomposition: reactive state and async services for inbound returns.
//

import Foundation
import Observation

@Observable
@MainActor
final class InboundReturnsViewModel {
    enum Tab: String, CaseIterable, Sendable {
        case queue = "Queue"
        case history = "History"
    }

    var tab: Tab = .queue
    var rows: [InboundReturnRow] = []
    var history: [InboundReturnRow] = []
    var loading = true
    var error: String?
    var barcode = ""
    var sessionId: String?
    var scanning = false
    var selected = Set<String>()
    var statusMessage: String?
    var queuedScans = 0
    var scannerEnabled = true

    private let api: APIClient
    private let offlineQueue: OfflineQueue

    init(api: APIClient = .shared, offlineQueue: OfflineQueue? = nil) {
        self.api = api
        self.offlineQueue = offlineQueue ?? .shared
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            async let inbound = api.inboundReturns()
            async let hist = api.returnsHistory(limit: 50)
            let (inboundResp, histResp) = try await (inbound, hist)
            rows = inboundResp.data
            history = histResp.data
            queuedScans = offlineQueue.read().filter { $0.endpoint.contains("returns/inbound/scan") }.count
        } catch {
            self.error = error.localizedDescription
        }
    }

    func ensureSession() async throws -> String {
        if let sessionId { return sessionId }
        let resp = try await api.createInboundSession()
        sessionId = resp.sessionId
        return resp.sessionId
    }

    func handleScan(online: Bool) async {
        scanning = true
        statusMessage = nil
        defer { scanning = false }
        let trimmed = barcode.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else { return }

        if !online {
            enqueueOfflineScan(barcode: trimmed)
            barcode = ""
            statusMessage = "Scan queued (offline)"
            queuedScans = offlineQueue.read().count
            return
        }

        do {
            let sid = try await ensureSession()
            let resp = try await api.scanInboundBarcode(barcode: trimmed, sessionId: sid)
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

    func enqueueOfflineScan(barcode: String) {
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
        offlineQueue.enqueue(action)
    }

    func confirm(disposition: String) async {
        let targets = rows.filter { selected.contains($0.returnId) || selected.isEmpty }
        guard !targets.isEmpty else {
            statusMessage = "Select lines or scan first"
            return
        }
        do {
            let sid = try await ensureSession()
            _ = try await api.confirmInboundReturns(
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

    func toggleSelect(_ id: String) {
        if selected.contains(id) {
            selected.remove(id)
        } else {
            selected.insert(id)
        }
    }
}
