import Foundation
import Observation

enum ReturnsTab: String, CaseIterable, Identifiable {
    case queue = "Gate queue"
    case history = "History"
    case reverse = "Credit-note"

    var id: String { rawValue }
}

@Observable
@MainActor
final class ReturnsViewModel {
    typealias WarehouseIdProvider = @MainActor () -> String?
    typealias FetchInboundReturns = @MainActor (String) async throws -> InboundReturnListResponse
    typealias FetchReturnsHistory = @MainActor (Int) async throws -> InboundReturnListResponse
    typealias FetchReverseLogistics = @MainActor (String, String?) async throws -> ReverseLogisticsListResponse
    typealias CreateInboundSession = @MainActor () async throws -> String
    typealias ScanInboundBarcode = @MainActor (String, Int, String?, String) async throws -> InboundScanResponse
    typealias ConfirmInboundReturns = @MainActor ([String], String, String, String) async throws -> InboundConfirmResponse
    typealias ReceiveReverseLogistics = @MainActor (String, String, [String: Int]) async throws -> [String: String]

    var tab: ReturnsTab = .queue
    var returns: [InboundReturnRow] = []
    var history: [InboundReturnRow] = []
    var reverseTasks: [ReverseLogisticsTask] = []
    var loading = true
    var error: String?
    var barcode = ""
    var sessionId: String?
    var scanning = false
    var selected: Set<String> = []
    var statusMessage: String?
    var scannerEnabled = true
    var receivingTaskId: String?

    private let warehouseIdProvider: WarehouseIdProvider
    private let fetchInboundReturns: FetchInboundReturns
    private let fetchReturnsHistory: FetchReturnsHistory
    private let fetchReverseLogistics: FetchReverseLogistics
    private let createInboundSessionOp: CreateInboundSession
    private let scanInboundBarcodeOp: ScanInboundBarcode
    private let confirmInboundReturnsOp: ConfirmInboundReturns
    private let receiveReverseLogisticsOp: ReceiveReverseLogistics

    init(
        warehouseIdProvider: WarehouseIdProvider? = nil,
        fetchInboundReturns: FetchInboundReturns? = nil,
        fetchReturnsHistory: FetchReturnsHistory? = nil,
        fetchReverseLogistics: FetchReverseLogistics? = nil,
        createInboundSession: CreateInboundSession? = nil,
        scanInboundBarcode: ScanInboundBarcode? = nil,
        confirmInboundReturns: ConfirmInboundReturns? = nil,
        receiveReverseLogistics: ReceiveReverseLogistics? = nil
    ) {
        self.warehouseIdProvider = warehouseIdProvider ?? {
            TokenStore.shared.warehouseId
        }
        self.fetchInboundReturns = fetchInboundReturns ?? { physicalStatus in
            try await WarehouseService.inboundReturns(physicalStatus: physicalStatus)
        }
        self.fetchReturnsHistory = fetchReturnsHistory ?? { limit in
            try await WarehouseService.returnsHistory(limit: limit)
        }
        self.fetchReverseLogistics = fetchReverseLogistics ?? { status, warehouseId in
            try await WarehouseService.reverseLogistics(status: status, warehouseId: warehouseId)
        }
        self.createInboundSessionOp = createInboundSession ?? {
            try await WarehouseService.createInboundSession()
        }
        self.scanInboundBarcodeOp = scanInboundBarcode ?? { barcode, qty, sessionId, idempotencyKey in
            try await WarehouseService.scanInboundBarcode(
                barcode: barcode,
                qty: qty,
                sessionId: sessionId,
                idempotencyKey: idempotencyKey
            )
        }
        self.confirmInboundReturnsOp = confirmInboundReturns ?? { returnIds, disposition, sessionId, idempotencyKey in
            try await WarehouseService.confirmInboundReturns(
                returnIds: returnIds,
                disposition: disposition,
                sessionId: sessionId,
                idempotencyKey: idempotencyKey
            )
        }
        self.receiveReverseLogisticsOp = receiveReverseLogistics ?? { taskId, warehouseId, receivedQty in
            try await WarehouseService.receiveReverseLogistics(
                taskId: taskId,
                warehouseId: warehouseId,
                receivedQty: receivedQty
            )
        }
    }

    var trimmedBarcode: String {
        barcode.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let wh = warehouseIdProvider()
            let inboundResp = try await fetchInboundReturns("OPEN")
            let histResp = try await fetchReturnsHistory(50)
            let reverseResp = try await fetchReverseLogistics("OPEN", wh)
            returns = inboundResp.data
            history = histResp.data
            reverseTasks = reverseResp.tasks
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func ensureSession() async throws -> String {
        if let sessionId { return sessionId }
        let sid = try await createInboundSessionOp()
        sessionId = sid
        return sid
    }

    func handleScannedCode(_ code: String) async {
        barcode = code
        await handleScan()
    }

    func handleScan() async {
        let trimmed = trimmedBarcode
        guard !trimmed.isEmpty, !scanning else { return }
        scanning = true
        statusMessage = nil
        defer { scanning = false }
        do {
            let sid = try await ensureSession()
            let key = WarehouseIdempotency.inboundScan(barcode: trimmed, sessionId: sid)
            _ = try await scanInboundBarcodeOp(trimmed, 1, sid, key)
            barcode = ""
            statusMessage = "Scan recorded"
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func confirm(disposition: String) async {
        guard !selected.isEmpty else { return }
        do {
            let sid = try await ensureSession()
            let ids = Array(selected)
            let key = WarehouseIdempotency.inboundConfirm(returnIds: ids, disposition: disposition)
            _ = try await confirmInboundReturnsOp(ids, disposition, sid, key)
            selected.removeAll()
            statusMessage = "\(disposition) confirmed"
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func receive(_ task: ReverseLogisticsTask) async {
        guard !task.taskId.isEmpty else { return }
        receivingTaskId = task.taskId
        defer { receivingTaskId = nil }
        do {
            let wh = warehouseIdProvider()?.trimmingCharacters(in: .whitespacesAndNewlines)
            let warehouseId = (wh?.isEmpty == false ? wh : nil)
                ?? (task.warehouseId.isEmpty ? "warehouse" : task.warehouseId)
            let qty = Self.parseReceivedQty(task.expectedQtyJson)
            _ = try await receiveReverseLogisticsOp(task.taskId, warehouseId, qty)
            statusMessage = "Received \(task.taskId)"
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    static func parseReceivedQty(_ raw: String?) -> [String: Int] {
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
